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
	TemplateID       uint   `json:"template_id"`
	TemplateContent  string `json:"template_content"`
	ExtraInstruction string `json:"extra_instruction"`
}

type SaveReplyDraftRequest struct {
	Channel          string `json:"channel"`
	TargetID         int64  `json:"target_id"`
	Content          string `json:"content"`
	SourceType       string `json:"source_type"`
	TemplateContent  string `json:"template_content"`
	ExtraInstruction string `json:"extra_instruction"`
}

type SendReplyDraftRequest struct {
	Channel          string `json:"channel"`
	TargetID         int64  `json:"target_id"`
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

func (s *ReplyWorkspaceService) GetWorkspace(ctx context.Context, channel string, targetID int64) (*ReplyWorkspaceData, error) {
	target, err := s.getTarget(ctx, channel, targetID)
	if err != nil {
		return nil, err
	}

	draft, err := s.draftRepo.GetByTarget(ctx, channel, targetID)
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
	logs, err := s.llmLogRepo.ListByConversation(ctx, conversationMeta.Key, 8)
	if err != nil {
		return nil, err
	}

	return &ReplyWorkspaceData{
		Target:    target,
		Draft:     draft,
		Templates: templates,
		Examples:  examples,
		Logs:      logs,
	}, nil
}

func (s *ReplyWorkspaceService) GenerateDraft(ctx context.Context, req GenerateReplyDraftRequest) (*model.ReplyDraft, error) {
	target, err := s.getTarget(ctx, req.Channel, req.TargetID)
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

	systemPrompt := s.buildSystemPrompt(req.Channel)
	userPrompt := s.buildUserPrompt(target, templateText, strings.TrimSpace(req.ExtraInstruction), examples)

	provider, err := s.llmProvider()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := provider.ChatWithSystem(ctx, systemPrompt, []llm.Message{{Role: "user", Content: userPrompt}})
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
	if err := s.draftRepo.Save(ctx, draft); err != nil {
		return nil, err
	}

	if s.llmLogRepo != nil {
		conversationMeta := s.conversationMeta(req.Channel, target)
		_ = s.llmLogRepo.Create(ctx, &model.LLMChatLog{
			Provider:          provider.Name(),
			Model:             resp.Model,
			InputType:         req.Channel,
			InputID:           req.TargetID,
			ConversationKey:   conversationMeta.Key,
			ConversationTitle: conversationMeta.Title,
			InputContent:      target.InputContent,
			SystemPrompt:      systemPrompt,
			RequestMessages:   marshalLLMMessages([]llm.Message{{Role: "user", Content: userPrompt}}),
			OutputContent:     resp.Content,
			PromptTokens:      resp.PromptTokens,
			OutputTokens:      resp.TokensUsed,
			TotalTokens:       resp.TotalTokens,
			Success:           true,
			Duration:          duration,
		})
	}

	return s.draftRepo.GetByTarget(ctx, req.Channel, req.TargetID)
}

func (s *ReplyWorkspaceService) SaveDraft(ctx context.Context, req SaveReplyDraftRequest) (*model.ReplyDraft, error) {
	if _, err := s.getTarget(ctx, req.Channel, req.TargetID); err != nil {
		return nil, err
	}
	draft := &model.ReplyDraft{
		Channel:          req.Channel,
		TargetID:         req.TargetID,
		Content:          strings.TrimSpace(req.Content),
		Status:           "draft",
		SourceType:       defaultString(strings.TrimSpace(req.SourceType), "manual"),
		TemplateSnapshot: strings.TrimSpace(req.TemplateContent),
		ExtraInstruction: strings.TrimSpace(req.ExtraInstruction),
	}
	if err := s.draftRepo.Save(ctx, draft); err != nil {
		return nil, err
	}
	return s.draftRepo.GetByTarget(ctx, req.Channel, req.TargetID)
}

func (s *ReplyWorkspaceService) SendDraft(ctx context.Context, req SendReplyDraftRequest) error {
	target, err := s.getTarget(ctx, req.Channel, req.TargetID)
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
		message, err := s.messageRepo.GetByMessageID(ctx, req.TargetID)
		if err != nil {
			return fmt.Errorf("get message failed: %w", err)
		}
		if err := client.SendMessage(ctx, message.SenderID, content); err != nil {
			return fmt.Errorf("send message failed: %w", err)
		}
		if err := s.messageRepo.UpdateReplyStatus(ctx, req.TargetID, 1, content, strings.TrimSpace(req.SourceType) == "ai"); err != nil {
			return fmt.Errorf("update message status failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported channel: %s", req.Channel)
	}

	now := time.Now()
	draft := &model.ReplyDraft{
		Channel:          req.Channel,
		TargetID:         req.TargetID,
		Content:          content,
		Status:           "sent",
		SourceType:       defaultString(strings.TrimSpace(req.SourceType), "manual"),
		TemplateSnapshot: strings.TrimSpace(req.TemplateContent),
		ExtraInstruction: strings.TrimSpace(req.ExtraInstruction),
		SentAt:           &now,
	}
	if err := s.draftRepo.Save(ctx, draft); err != nil {
		return err
	}

	if req.SaveAsExample {
		_ = s.exampleRepo.Create(ctx, &model.ReplyExample{
			Channel:      req.Channel,
			Title:        defaultString(strings.TrimSpace(req.ExampleTitle), s.defaultExampleTitle(target)),
			UserInput:    target.InputContent,
			ReplyContent: content,
			Notes:        strings.TrimSpace(req.ExampleNotes),
			SourceType:   draft.SourceType,
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

func (s *ReplyWorkspaceService) getTarget(ctx context.Context, channel string, targetID int64) (*ReplyWorkspaceTarget, error) {
	switch channel {
	case ReplyChannelComment:
		comment, err := s.commentRepo.GetByCommentID(ctx, targetID)
		if err != nil {
			return nil, fmt.Errorf("get comment failed: %w", err)
		}
		return &ReplyWorkspaceTarget{
			Channel:      channel,
			TargetID:     targetID,
			Title:        defaultString(strings.TrimSpace(comment.VideoBVID), "评论回复"),
			InputContent: comment.Content,
			AuthorName:   defaultString(strings.TrimSpace(comment.AuthorName), "用户"),
			ReplyStatus:  comment.ReplyStatus,
			ReplyContent: comment.ReplyContent,
		}, nil
	case ReplyChannelMessage:
		message, err := s.messageRepo.GetByMessageID(ctx, targetID)
		if err != nil {
			return nil, fmt.Errorf("get message failed: %w", err)
		}
		return &ReplyWorkspaceTarget{
			Channel:        channel,
			TargetID:       targetID,
			Title:          defaultString(strings.TrimSpace(message.ConversationName), "私信会话"),
			InputContent:   message.Content,
			AuthorName:     defaultString(strings.TrimSpace(message.SenderName), "用户"),
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
		return "你是B站UP主的私信助手。请生成可直接发送的回复草稿，语气真诚、清楚、克制，不要写解释，不要分点。"
	}
	return "你是B站UP主的评论区助手。请生成可直接发送的评论回复草稿，语气自然、友好、有一点站内交流感，不要写解释，不要分点。"
}

func (s *ReplyWorkspaceService) buildUserPrompt(target *ReplyWorkspaceTarget, templateText, extraInstruction string, examples []model.ReplyExample) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("回复渠道: %s\n", target.Channel))
	b.WriteString(fmt.Sprintf("对方昵称: %s\n", target.AuthorName))
	b.WriteString(fmt.Sprintf("原始内容:\n%s\n", strings.TrimSpace(target.InputContent)))
	if templateText != "" {
		b.WriteString("\n可参考模板:\n")
		b.WriteString(templateText)
		b.WriteString("\n")
	}
	if len(examples) > 0 {
		b.WriteString("\n高质量示例:\n")
		for i, item := range examples {
			b.WriteString(fmt.Sprintf("%d. 用户: %s\n", i+1, singleLine(item.UserInput)))
			b.WriteString(fmt.Sprintf("   回复: %s\n", singleLine(item.ReplyContent)))
		}
	}
	if extraInstruction != "" {
		b.WriteString("\n补充要求:\n")
		b.WriteString(extraInstruction)
		b.WriteString("\n")
	}
	if target.Channel == ReplyChannelMessage {
		b.WriteString("\n要求: 控制在 30-120 字，优先解决问题；如果信息不足，给出礼貌追问。")
	} else {
		b.WriteString("\n要求: 控制在 20-80 字，贴近B站互动口吻，但不要油腻。")
	}
	b.WriteString("\n请直接输出最终回复正文。")
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
	seeds := defaultTemplates(channel)
	for _, item := range seeds {
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
			{Channel: channel, Title: "礼貌承接", Scene: "日常咨询", Content: "收到啦，感谢你专门来私信我。这个问题我先帮你确认一下，如果有更具体的信息也可以继续发我。"},
			{Channel: channel, Title: "合作初筛", Scene: "商务合作", Content: "感谢联系，合作相关可以先把品牌、需求、预算区间和预期排期发我，我这边看完后尽快回复你。"},
			{Channel: channel, Title: "无法立即处理", Scene: "延迟回复", Content: "这条我看到了，不过我现在没法立刻完整处理，晚一点我会回来继续跟你对一下，辛苦稍等。"},
		}
	case ReplyChannelComment:
		return []model.ReplyTemplate{
			{Channel: channel, Title: "感谢支持", Scene: "普通互动", Content: "感谢支持，也谢谢你认真看到这里，后面我继续努力更新。"},
			{Channel: channel, Title: "回答提问", Scene: "问题答复", Content: "这个点你提得很准，简单说就是这样处理的，后面有机会我也可以专门做一期展开讲。"},
			{Channel: channel, Title: "承接建议", Scene: "建议反馈", Content: "收到这个建议了，确实有参考价值，我后面会继续优化，感谢你认真留言。"},
		}
	default:
		return nil
	}
}

func (s *ReplyWorkspaceService) defaultExampleTitle(target *ReplyWorkspaceTarget) string {
	if target == nil {
		return "沉淀示例"
	}
	if target.Channel == ReplyChannelMessage {
		return fmt.Sprintf("私信示例-%s", target.AuthorName)
	}
	return fmt.Sprintf("评论示例-%s", target.AuthorName)
}

func (s *ReplyWorkspaceService) conversationMeta(channel string, target *ReplyWorkspaceTarget) llmConversationMeta {
	switch channel {
	case ReplyChannelComment:
		return llmConversationMeta{
			Key:   fmt.Sprintf("comment:%s", target.Title),
			Title: target.Title,
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
