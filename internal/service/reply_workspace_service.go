package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	appruntime "bilibili-up-admin/internal/runtime"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

const (
	ReplyChannelComment = "comment"
	ReplyChannelMessage = "message"

	replyLogTypeDraft   = "draft"
	replyLogTypeSummary = "summary"

	replyConversationTokenThreshold = 3200
	replyConversationKeepTurns      = 4
)

type ReplyWorkspaceService struct {
	runtime      *appruntime.Store
	commentRepo  *repository.CommentRepository
	messageRepo  *repository.MessageRepository
	templateRepo *repository.ReplyTemplateRepository
	exampleRepo  *repository.ReplyExampleRepository
	draftRepo    *repository.ReplyDraftRepository
	llmLogRepo   *repository.LLMChatLogRepository
}

func NewReplyWorkspaceService(
	runtime *appruntime.Store,
	commentRepo *repository.CommentRepository,
	messageRepo *repository.MessageRepository,
	templateRepo *repository.ReplyTemplateRepository,
	exampleRepo *repository.ReplyExampleRepository,
	draftRepo *repository.ReplyDraftRepository,
	llmLogRepo *repository.LLMChatLogRepository,
) *ReplyWorkspaceService {
	return &ReplyWorkspaceService{
		runtime:      runtime,
		commentRepo:  commentRepo,
		messageRepo:  messageRepo,
		templateRepo: templateRepo,
		exampleRepo:  exampleRepo,
		draftRepo:    draftRepo,
		llmLogRepo:   llmLogRepo,
	}
}

type ReplyWorkspaceTarget struct {
	Channel        string `json:"channel"`
	TargetID       int64  `json:"target_id"`
	Title          string `json:"title"`
	InputContent   string `json:"input_content"`
	AuthorName     string `json:"author_name"`
	VideoBVID      string `json:"video_bvid,omitempty"`
	VideoTitle     string `json:"video_title,omitempty"`
	VideoDesc      string `json:"video_desc,omitempty"`
	ConversationID int64  `json:"conversation_id,omitempty"`
	ReplyStatus    int    `json:"reply_status"`
	ReplyContent   string `json:"reply_content,omitempty"`
}

type ReplyWorkspaceData struct {
	Target    *ReplyWorkspaceTarget `json:"target"`
	Draft     *model.ReplyDraft     `json:"draft"`
	Templates []model.ReplyTemplate `json:"templates"`
	Examples  []model.ReplyExample  `json:"examples"`
	Logs      []model.LLMChatLog    `json:"logs"`
}

type GenerateReplyDraftRequest struct {
	Channel          string `json:"channel"`
	TargetID         int64  `json:"target_id"`
	ConversationID   int64  `json:"conversation_id"`
	TemplateID       uint   `json:"template_id"`
	TemplateContent  string `json:"template_content"`
	ExtraInstruction string `json:"extra_instruction"`
}

type SaveReplyDraftRequest struct {
	Channel          string `json:"channel"`
	TargetID         int64  `json:"target_id"`
	ConversationID   int64  `json:"conversation_id"`
	Content          string `json:"content"`
	SourceType       string `json:"source_type"`
	TemplateContent  string `json:"template_content"`
	ExtraInstruction string `json:"extra_instruction"`
}

type SendReplyDraftRequest struct {
	Channel          string `json:"channel"`
	TargetID         int64  `json:"target_id"`
	ConversationID   int64  `json:"conversation_id"`
	Content          string `json:"content"`
	SourceType       string `json:"source_type"`
	TemplateContent  string `json:"template_content"`
	ExtraInstruction string `json:"extra_instruction"`
	SaveAsExample    bool   `json:"save_as_example"`
	ExampleTitle     string `json:"example_title"`
	ExampleNotes     string `json:"example_notes"`
}

func (s *ReplyWorkspaceService) biliClient() (*bilibili.Client, error) {
	if s.runtime == nil || s.runtime.BilibiliClient() == nil {
		return nil, fmt.Errorf("bilibili login is not configured")
	}
	return s.runtime.BilibiliClient(), nil
}

func (s *ReplyWorkspaceService) llmProvider() (llm.Provider, error) {
	if s.runtime == nil || s.runtime.LLMManager() == nil {
		return nil, fmt.Errorf("llm is not configured")
	}
	return s.runtime.LLMManager().Default()
}

func (s *ReplyWorkspaceService) GetWorkspace(ctx context.Context, channel string, targetID, conversationID int64) (*ReplyWorkspaceData, error) {
	target, err := s.getTarget(ctx, channel, targetID, conversationID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureSeedTemplates(ctx, channel); err != nil {
		return nil, err
	}

	templates, err := s.templateRepo.List(ctx, channel, 8)
	if err != nil {
		return nil, err
	}
	examples, err := s.exampleRepo.List(ctx, channel, 6)
	if err != nil {
		return nil, err
	}
	conversationMeta := s.conversationMeta(channel, target)
	logs, err := s.llmLogRepo.ListByConversation(ctx, conversationMeta.Key, 12)
	if err != nil {
		return nil, err
	}

	return &ReplyWorkspaceData{
		Target:    target,
		Draft:     nil,
		Templates: templates,
		Examples:  examples,
		Logs:      logs,
	}, nil
}

func (s *ReplyWorkspaceService) GenerateDraft(ctx context.Context, req GenerateReplyDraftRequest) (*model.ReplyDraft, error) {
	target, err := s.getTarget(ctx, req.Channel, req.TargetID, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if target.ReplyStatus != 0 {
		return nil, fmt.Errorf("target already replied")
	}

	templateText := strings.TrimSpace(req.TemplateContent)
	if req.TemplateID > 0 {
		item, err := s.templateRepo.GetByID(ctx, req.TemplateID)
		if err != nil {
			return nil, err
		}
		if item != nil {
			templateText = strings.TrimSpace(item.Content)
			_ = s.templateRepo.TouchUsage(ctx, item.ID)
		}
	}

	examples, _ := s.exampleRepo.List(ctx, req.Channel, 3)
	provider, err := s.llmProvider()
	if err != nil {
		return nil, err
	}

	conversationMeta := s.conversationMeta(req.Channel, target)
	history, err := s.llmLogRepo.ListByConversationOldestFirst(ctx, conversationMeta.Key, 64)
	if err != nil {
		return nil, err
	}

	systemPrompt := s.buildSystemPrompt(req.Channel)
	messages, err := s.buildConversationMessages(
		ctx,
		provider,
		conversationMeta,
		target,
		history,
		templateText,
		strings.TrimSpace(req.ExtraInstruction),
		examples,
	)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := provider.ChatWithSystem(ctx, systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("llm chat failed: %w", err)
	}
	duration := time.Since(start).Milliseconds()

	draft := &model.ReplyDraft{
		Channel:          req.Channel,
		TargetID:         req.TargetID,
		Content:          strings.TrimSpace(resp.Content),
		Status:           "generated",
		SourceType:       "ai",
		TemplateSnapshot: templateText,
		ExtraInstruction: strings.TrimSpace(req.ExtraInstruction),
		ModelProvider:    provider.Name(),
		ModelName:        resp.Model,
		GeneratedAt:      ptrTime(time.Now()),
	}

	if s.llmLogRepo != nil {
		_ = s.llmLogRepo.Create(ctx, &model.LLMChatLog{
			Provider:          provider.Name(),
			Model:             resp.Model,
			LogType:           replyLogTypeDraft,
			InputType:         req.Channel,
			InputID:           req.TargetID,
			ConversationKey:   conversationMeta.Key,
			ConversationTitle: conversationMeta.Title,
			InputContent:      target.InputContent,
			SystemPrompt:      systemPrompt,
			RequestMessages:   marshalLLMMessages(messages),
			OutputContent:     resp.Content,
			PromptTokens:      resp.PromptTokens,
			OutputTokens:      resp.TokensUsed,
			TotalTokens:       resp.TotalTokens,
			Success:           true,
			Duration:          duration,
		})
	}

	return draft, nil
}

func (s *ReplyWorkspaceService) SaveDraft(ctx context.Context, req SaveReplyDraftRequest) (*model.ReplyDraft, error) {
	return nil, fmt.Errorf("draft saving has been removed")
}

func (s *ReplyWorkspaceService) SendDraft(ctx context.Context, req SendReplyDraftRequest) error {
	target, err := s.getTarget(ctx, req.Channel, req.TargetID, req.ConversationID)
	if err != nil {
		return err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return fmt.Errorf("reply content is empty")
	}
	if target.ReplyStatus != 0 {
		return fmt.Errorf("target already replied")
	}

	client, err := s.biliClient()
	if err != nil {
		return err
	}

	switch req.Channel {
	case ReplyChannelComment:
		comment, err := s.commentRepo.GetByCommentID(ctx, req.TargetID)
		if err != nil {
			return fmt.Errorf("get comment failed: %w", err)
		}
		if err := client.ReplyComment(ctx, comment.VideoAID, req.TargetID, content); err != nil {
			return fmt.Errorf("send reply failed: %w", err)
		}
		if err := s.commentRepo.UpdateReplyStatus(ctx, req.TargetID, 1, content); err != nil {
			return fmt.Errorf("update comment status failed: %w", err)
		}
		comment.IsAIReply = strings.TrimSpace(req.SourceType) == "ai"
		comment.ReplyContent = content
		_ = s.commentRepo.Update(ctx, comment)
	case ReplyChannelMessage:
		message, err := s.resolveMessageTarget(ctx, req.TargetID, req.ConversationID)
		if err != nil {
			return fmt.Errorf("get message failed: %w", err)
		}
		if err := client.SendMessage(ctx, message.SenderID, content); err != nil {
			return fmt.Errorf("send message failed: %w", err)
		}
		if err := s.messageRepo.UpdateReplyStatus(ctx, message.MessageID, 1, content, strings.TrimSpace(req.SourceType) == "ai"); err != nil {
			return fmt.Errorf("update message status failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported channel: %s", req.Channel)
	}

	if req.SaveAsExample {
		_ = s.exampleRepo.Create(ctx, &model.ReplyExample{
			Channel:      req.Channel,
			Title:        defaultString(strings.TrimSpace(req.ExampleTitle), s.defaultExampleTitle(target)),
			UserInput:    target.InputContent,
			ReplyContent: content,
			Notes:        strings.TrimSpace(req.ExampleNotes),
			SourceType:   defaultString(strings.TrimSpace(req.SourceType), "manual"),
			SourceID:     req.TargetID,
			QualityScore: 90,
		})
	}
	return nil
}

func (s *ReplyWorkspaceService) ListTemplates(ctx context.Context, channel string) ([]model.ReplyTemplate, error) {
	if err := s.ensureSeedTemplates(ctx, channel); err != nil {
		return nil, err
	}
	return s.templateRepo.List(ctx, channel, 50)
}

func (s *ReplyWorkspaceService) CreateTemplate(ctx context.Context, channel, title, content, scene string) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if channel == "" || title == "" || content == "" {
		return fmt.Errorf("channel, title and content are required")
	}
	return s.templateRepo.Create(ctx, &model.ReplyTemplate{
		Channel: channel,
		Title:   title,
		Content: content,
		Scene:   strings.TrimSpace(scene),
	})
}

func (s *ReplyWorkspaceService) DeleteTemplate(ctx context.Context, id uint) error {
	return s.templateRepo.Delete(ctx, id)
}

func (s *ReplyWorkspaceService) getTarget(ctx context.Context, channel string, targetID, conversationID int64) (*ReplyWorkspaceTarget, error) {
	switch channel {
	case ReplyChannelComment:
		comment, err := s.commentRepo.GetByCommentID(ctx, targetID)
		if err != nil {
			return nil, fmt.Errorf("get comment failed: %w", err)
		}

		videoTitle := strings.TrimSpace(comment.VideoBVID)
		videoDesc := ""
		if strings.TrimSpace(comment.VideoBVID) != "" {
			if client, clientErr := s.biliClient(); clientErr == nil {
				if info, infoErr := client.GetVideoInfo(ctx, comment.VideoBVID); infoErr == nil && info != nil {
					if strings.TrimSpace(info.Title) != "" {
						videoTitle = strings.TrimSpace(info.Title)
					}
					videoDesc = strings.TrimSpace(info.Desc)
				}
			}
		}

		return &ReplyWorkspaceTarget{
			Channel:      channel,
			TargetID:     targetID,
			Title:        defaultString(videoTitle, "comment-reply"),
			InputContent: comment.Content,
			AuthorName:   defaultString(strings.TrimSpace(comment.AuthorName), "viewer"),
			VideoBVID:    strings.TrimSpace(comment.VideoBVID),
			VideoTitle:   videoTitle,
			VideoDesc:    videoDesc,
			ReplyStatus:  comment.ReplyStatus,
			ReplyContent: comment.ReplyContent,
		}, nil
	case ReplyChannelMessage:
		message, err := s.resolveMessageTarget(ctx, targetID, conversationID)
		if err != nil {
			return nil, fmt.Errorf("get message failed: %w", err)
		}
		return &ReplyWorkspaceTarget{
			Channel:        channel,
			TargetID:       message.MessageID,
			Title:          defaultString(strings.TrimSpace(message.ConversationName), "private-message"),
			InputContent:   message.Content,
			AuthorName:     defaultString(strings.TrimSpace(message.SenderName), "viewer"),
			ConversationID: message.ConversationUID,
			ReplyStatus:    message.ReplyStatus,
			ReplyContent:   message.ReplyContent,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported channel: %s", channel)
	}
}

func (s *ReplyWorkspaceService) buildSystemPrompt(channel string) string {
	if channel == ReplyChannelMessage {
		return "你是创作者的私信回复助手。请直接用自然、简洁的中文生成可发送的回复，优先解决对方问题；如果信息不足，只补一个简短追问。不要解释推理过程。"
	}
	return "你是创作者的评论回复助手。请用符合 B 站语境的自然中文生成可直接发布的回复，保持简洁、友好、不油腻。不要解释推理过程。"
}

func (s *ReplyWorkspaceService) buildUserPrompt(target *ReplyWorkspaceTarget, templateText, extraInstruction string, examples []model.ReplyExample) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("渠道：%s\n", target.Channel))
	b.WriteString(fmt.Sprintf("用户：%s\n", target.AuthorName))
	if target.Channel == ReplyChannelComment {
		b.WriteString(fmt.Sprintf("评论用户昵称：%s\n", target.AuthorName))
		if target.VideoTitle != "" {
			b.WriteString(fmt.Sprintf("视频标题：%s\n", target.VideoTitle))
		}
		if target.VideoBVID != "" {
			b.WriteString(fmt.Sprintf("视频 BVID：%s\n", target.VideoBVID))
		}
		if target.VideoDesc != "" {
			b.WriteString("视频简介：\n")
			b.WriteString(target.VideoDesc)
			b.WriteString("\n")
		}
	}
	b.WriteString("当前用户消息：\n")
	b.WriteString(strings.TrimSpace(target.InputContent))
	b.WriteString("\n")
	if templateText != "" {
		b.WriteString("\n可参考模板：\n")
		b.WriteString(templateText)
		b.WriteString("\n")
	}
	if len(examples) > 0 {
		b.WriteString("\n高质量示例：\n")
		for i, item := range examples {
			b.WriteString(fmt.Sprintf("%d. 用户：%s\n", i+1, singleLine(item.UserInput)))
			b.WriteString(fmt.Sprintf("   回复：%s\n", singleLine(item.ReplyContent)))
		}
	}
	if extraInstruction != "" {
		b.WriteString("\n补充要求：\n")
		b.WriteString(extraInstruction)
		b.WriteString("\n")
	}
	if target.Channel == ReplyChannelMessage {
		b.WriteString("\n要求：30 到 120 个中文字符，先解决对方问题；如果缺信息，只补一个简短追问。\n")
	} else {
		b.WriteString("\n要求：20 到 80 个中文字符，自然、友好、不油腻。\n")
	}
	b.WriteString("只输出最终回复正文，不要加解释。")
	return b.String()
}

func (s *ReplyWorkspaceService) resolveMessageTarget(ctx context.Context, targetID, conversationID int64) (*model.Message, error) {
	if targetID > 0 {
		message, err := s.messageRepo.GetByMessageID(ctx, targetID)
		if err == nil {
			return message, nil
		}
	}
	if conversationID > 0 {
		message, err := s.messageRepo.FindReplyTarget(ctx, conversationID)
		if err == nil {
			return message, nil
		}
	}
	if targetID > 0 {
		return nil, fmt.Errorf("message %d not found", targetID)
	}
	return nil, fmt.Errorf("message target not found")
}

func (s *ReplyWorkspaceService) buildConversationMessages(
	ctx context.Context,
	provider llm.Provider,
	conversationMeta llmConversationMeta,
	target *ReplyWorkspaceTarget,
	history []model.LLMChatLog,
	templateText, extraInstruction string,
	examples []model.ReplyExample,
) ([]llm.Message, error) {
	history = compactConversationLogs(history)
	summary, turns, err := s.ensureConversationSummary(ctx, provider, conversationMeta, target, history)
	if err != nil {
		return nil, err
	}

	messages := make([]llm.Message, 0, len(turns)*2+4)
	if summary != "" {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: "历史摘要：\n" + summary,
		})
	}
	if target.Channel == ReplyChannelComment && len(turns) == 0 {
		if videoContext := s.buildVideoContext(target); videoContext != "" {
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: videoContext,
			})
		}
	}
	for _, item := range turns {
		if text := strings.TrimSpace(item.InputContent); text != "" {
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: "上一轮用户消息：\n" + text,
			})
		}
		if text := strings.TrimSpace(item.OutputContent); text != "" {
			messages = append(messages, llm.Message{
				Role:    "assistant",
				Content: text,
			})
		}
	}
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: s.buildUserPrompt(target, templateText, extraInstruction, examples),
	})
	return messages, nil
}

func (s *ReplyWorkspaceService) ensureConversationSummary(
	ctx context.Context,
	provider llm.Provider,
	conversationMeta llmConversationMeta,
	target *ReplyWorkspaceTarget,
	history []model.LLMChatLog,
) (string, []model.LLMChatLog, error) {
	summary := ""
	turns := make([]model.LLMChatLog, 0, len(history))
	for _, item := range history {
		switch item.LogType {
		case replyLogTypeSummary:
			summary = strings.TrimSpace(item.OutputContent)
			turns = turns[:0]
		case "", replyLogTypeDraft:
			if item.Success {
				turns = append(turns, item)
			}
		}
	}

	if !needsConversationSummary(turns) {
		return summary, turns, nil
	}

	cutoff := len(turns) - replyConversationKeepTurns
	if cutoff < 1 {
		return summary, turns, nil
	}

	older := turns[:cutoff]
	recent := append([]model.LLMChatLog(nil), turns[cutoff:]...)
	prompt := buildConversationSummaryPrompt(target, summary, older)

	start := time.Now()
	resp, err := provider.ChatWithSystem(
		ctx,
		"请把较早的对话压缩成精炼中文摘要，保留用户诉求、关键事实、已经承诺过的事情，以及尚未解决的问题。",
		[]llm.Message{{Role: "user", Content: prompt}},
	)
	if err != nil {
		return summary, turns, fmt.Errorf("summarize conversation failed: %w", err)
	}
	duration := time.Since(start).Milliseconds()

	newSummary := strings.TrimSpace(resp.Content)
	if summary != "" {
		newSummary = strings.TrimSpace(summary + "\n" + newSummary)
	}
	if s.llmLogRepo != nil {
		_ = s.llmLogRepo.Create(ctx, &model.LLMChatLog{
			Provider:          provider.Name(),
			Model:             resp.Model,
			LogType:           replyLogTypeSummary,
			InputType:         target.Channel,
			InputID:           target.TargetID,
			ConversationKey:   conversationMeta.Key,
			ConversationTitle: conversationMeta.Title,
			InputContent:      target.InputContent,
			SystemPrompt:      "对话摘要压缩",
			RequestMessages:   marshalLLMMessages([]llm.Message{{Role: "user", Content: prompt}}),
			OutputContent:     newSummary,
			PromptTokens:      resp.PromptTokens,
			OutputTokens:      resp.TokensUsed,
			TotalTokens:       resp.TotalTokens,
			Success:           true,
			Duration:          duration,
		})
	}

	return newSummary, recent, nil
}

func compactConversationLogs(history []model.LLMChatLog) []model.LLMChatLog {
	out := make([]model.LLMChatLog, 0, len(history))
	for _, item := range history {
		if !item.Success {
			continue
		}
		if item.LogType == "" || item.LogType == replyLogTypeDraft || item.LogType == replyLogTypeSummary {
			out = append(out, item)
		}
	}
	return out
}

func needsConversationSummary(turns []model.LLMChatLog) bool {
	if len(turns) <= replyConversationKeepTurns {
		return false
	}
	total := 0
	for _, item := range turns {
		if item.TotalTokens > 0 {
			total += item.TotalTokens
		} else {
			total += approximateTokens(item.InputContent) + approximateTokens(item.OutputContent)
		}
	}
	return total >= replyConversationTokenThreshold
}

func buildConversationSummaryPrompt(target *ReplyWorkspaceTarget, previousSummary string, turns []model.LLMChatLog) string {
	var b strings.Builder
	if previousSummary != "" {
		b.WriteString("已有摘要：\n")
		b.WriteString(previousSummary)
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("渠道：%s\n", target.Channel))
	if target.Channel == ReplyChannelComment {
		if target.VideoTitle != "" {
			b.WriteString(fmt.Sprintf("视频标题：%s\n", target.VideoTitle))
		}
		if target.VideoDesc != "" {
			b.WriteString("视频简介：\n")
			b.WriteString(target.VideoDesc)
			b.WriteString("\n")
		}
	}
	b.WriteString("需要压缩的较早轮次：\n")
	for i, item := range turns {
		b.WriteString(fmt.Sprintf("%d. 用户：%s\n", i+1, singleLine(item.InputContent)))
		b.WriteString(fmt.Sprintf("   回复：%s\n", singleLine(item.OutputContent)))
	}
	b.WriteString("请写出一段供后续回复生成使用的精炼中文摘要。")
	return b.String()
}

func (s *ReplyWorkspaceService) buildVideoContext(target *ReplyWorkspaceTarget) string {
	if target == nil || target.Channel != ReplyChannelComment {
		return ""
	}
	if strings.TrimSpace(target.VideoTitle) == "" && strings.TrimSpace(target.VideoDesc) == "" && strings.TrimSpace(target.VideoBVID) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("这是一条全新的评论对话，请优先参考视频上下文。\n")
	if target.VideoTitle != "" {
		b.WriteString("视频标题：" + target.VideoTitle + "\n")
	}
	if target.VideoBVID != "" {
		b.WriteString("BVID：" + target.VideoBVID + "\n")
	}
	if target.VideoDesc != "" {
		b.WriteString("视频简介：\n")
		b.WriteString(target.VideoDesc)
		b.WriteString("\n")
	}
	return b.String()
}

func (s *ReplyWorkspaceService) ensureSeedTemplates(ctx context.Context, channel string) error {
	if channel == "" {
		return nil
	}
	items, err := s.templateRepo.List(ctx, channel, 1)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return nil
	}
	for _, item := range defaultTemplates(channel) {
		tmp := item
		if err := s.templateRepo.Create(ctx, &tmp); err != nil {
			return err
		}
	}
	return nil
}

func defaultTemplates(channel string) []model.ReplyTemplate {
	switch channel {
	case ReplyChannelMessage:
		return []model.ReplyTemplate{
			{Channel: channel, Title: "礼貌确认", Scene: "通用私信", Content: "我看到你的消息了，感谢联系我。如果你愿意再补充一点细节，我可以回复得更准确。"},
			{Channel: channel, Title: "商务收集", Scene: "商务合作", Content: "感谢联系。可以先发一下品牌、合作需求、预算范围、时间安排和预期交付内容，我这边再统一看。"},
			{Channel: channel, Title: "延后处理", Scene: "暂缓回复", Content: "我已经看到这条消息了，当前没法马上完整处理，稍后我会继续跟进回复你。"},
		}
	case ReplyChannelComment:
		return []model.ReplyTemplate{
			{Channel: channel, Title: "感谢支持", Scene: "通用评论", Content: "谢谢你来看视频还专门留言，真的很有帮助。"},
			{Channel: channel, Title: "回答问题", Scene: "问题解答", Content: "你提到的这个点很关键，我这里的处理方式就是这样；如果你需要，我后面可以单独补一期详细说明。"},
			{Channel: channel, Title: "接收反馈", Scene: "反馈建议", Content: "收到，这个反馈很有价值，我后面会继续优化。"},
		}
	default:
		return nil
	}
}

func (s *ReplyWorkspaceService) defaultExampleTitle(target *ReplyWorkspaceTarget) string {
	if target == nil {
		return "示例回复"
	}
	if target.Channel == ReplyChannelMessage {
		return fmt.Sprintf("私信示例-%s", target.AuthorName)
	}
	return fmt.Sprintf("评论示例-%s", target.AuthorName)
}

func (s *ReplyWorkspaceService) conversationMeta(channel string, target *ReplyWorkspaceTarget) llmConversationMeta {
	switch channel {
	case ReplyChannelComment:
		title := defaultString(strings.TrimSpace(target.VideoTitle), strings.TrimSpace(target.VideoBVID))
		if title == "" {
			title = target.Title
		}
		return llmConversationMeta{
			Key:   fmt.Sprintf("comment:%s:%d", defaultString(strings.TrimSpace(target.VideoBVID), "video"), target.TargetID),
			Title: fmt.Sprintf("%s #%d", title, target.TargetID),
		}
	case ReplyChannelMessage:
		conversationID := target.ConversationID
		if conversationID == 0 {
			conversationID = target.TargetID
		}
		return llmConversationMeta{
			Key:   fmt.Sprintf("message:%d", conversationID),
			Title: target.Title,
		}
	default:
		return llmConversationMeta{}
	}
}

func approximateTokens(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	runes := len([]rune(v))
	if runes < 4 {
		return 1
	}
	return runes / 4
}

func singleLine(v string) string {
	v = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	if len([]rune(v)) > 60 {
		return string([]rune(v)[:60]) + "..."
	}
	return v
}

func defaultString(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
