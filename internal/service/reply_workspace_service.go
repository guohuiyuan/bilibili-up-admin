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
	logs, err := s.llmLogRepo.ListByConversation(ctx, conversationMeta.Key, 12)
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
	if err := s.draftRepo.Save(ctx, draft); err != nil {
		return nil, err
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
		message, err := s.messageRepo.GetByMessageID(ctx, targetID)
		if err != nil {
			return nil, fmt.Errorf("get message failed: %w", err)
		}
		return &ReplyWorkspaceTarget{
			Channel:        channel,
			TargetID:       targetID,
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
		return "You are the creator's DM assistant. Reply in concise, natural Chinese. Keep it direct, useful, and ready to send. Do not explain your reasoning."
	}
	return "You are the creator's comment assistant. Reply in natural Chinese that matches Bilibili comment tone. Keep it concise, friendly, and ready to post. Do not explain your reasoning."
}

func (s *ReplyWorkspaceService) buildUserPrompt(target *ReplyWorkspaceTarget, templateText, extraInstruction string, examples []model.ReplyExample) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Channel: %s\n", target.Channel))
	b.WriteString(fmt.Sprintf("User: %s\n", target.AuthorName))
	if target.Channel == ReplyChannelComment {
		if target.VideoTitle != "" {
			b.WriteString(fmt.Sprintf("Video title: %s\n", target.VideoTitle))
		}
		if target.VideoBVID != "" {
			b.WriteString(fmt.Sprintf("Video BVID: %s\n", target.VideoBVID))
		}
	}
	b.WriteString("Current user message:\n")
	b.WriteString(strings.TrimSpace(target.InputContent))
	b.WriteString("\n")
	if templateText != "" {
		b.WriteString("\nReference template:\n")
		b.WriteString(templateText)
		b.WriteString("\n")
	}
	if len(examples) > 0 {
		b.WriteString("\nGood examples:\n")
		for i, item := range examples {
			b.WriteString(fmt.Sprintf("%d. User: %s\n", i+1, singleLine(item.UserInput)))
			b.WriteString(fmt.Sprintf("   Reply: %s\n", singleLine(item.ReplyContent)))
		}
	}
	if extraInstruction != "" {
		b.WriteString("\nExtra instructions:\n")
		b.WriteString(extraInstruction)
		b.WriteString("\n")
	}
	if target.Channel == ReplyChannelMessage {
		b.WriteString("\nConstraints: 30-120 Chinese characters. Solve the user's issue first. If information is missing, ask a short follow-up question.\n")
	} else {
		b.WriteString("\nConstraints: 20-80 Chinese characters. Stay natural, warm, and not greasy.\n")
	}
	b.WriteString("Return only the final reply text.")
	return b.String()
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
			Content: "Conversation summary:\n" + summary,
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
				Content: "Previous user message:\n" + text,
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
		"Compress prior conversation into a compact Chinese summary. Preserve user intent, factual context, commitments already made, and unresolved questions.",
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
			SystemPrompt:      "conversation-summary",
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
		b.WriteString("Existing summary:\n")
		b.WriteString(previousSummary)
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("Channel: %s\n", target.Channel))
	if target.Channel == ReplyChannelComment {
		if target.VideoTitle != "" {
			b.WriteString(fmt.Sprintf("Video title: %s\n", target.VideoTitle))
		}
		if target.VideoDesc != "" {
			b.WriteString("Video description:\n")
			b.WriteString(target.VideoDesc)
			b.WriteString("\n")
		}
	}
	b.WriteString("Older turns to compress:\n")
	for i, item := range turns {
		b.WriteString(fmt.Sprintf("%d. User: %s\n", i+1, singleLine(item.InputContent)))
		b.WriteString(fmt.Sprintf("   Reply: %s\n", singleLine(item.OutputContent)))
	}
	b.WriteString("Write a compact Chinese summary for future reply generation.")
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
	b.WriteString("This is a brand new comment conversation. Use the video context first.\n")
	if target.VideoTitle != "" {
		b.WriteString("Video title: " + target.VideoTitle + "\n")
	}
	if target.VideoBVID != "" {
		b.WriteString("BVID: " + target.VideoBVID + "\n")
	}
	if target.VideoDesc != "" {
		b.WriteString("Video description:\n")
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
			{Channel: channel, Title: "Polite ack", Scene: "General DM", Content: "I saw your message. Thanks for reaching out. If you can share a bit more detail, I can answer more precisely."},
			{Channel: channel, Title: "Business intake", Scene: "Business", Content: "Thanks for reaching out. Please send the brand, brief, budget range, timeline, and expected deliverables first."},
			{Channel: channel, Title: "Delayed handling", Scene: "Delay", Content: "I saw this message. I cannot handle it fully right now, but I will follow up as soon as I can."},
		}
	case ReplyChannelComment:
		return []model.ReplyTemplate{
			{Channel: channel, Title: "Thanks", Scene: "General", Content: "Thanks for watching and leaving a comment. I appreciate it."},
			{Channel: channel, Title: "Answer question", Scene: "Question", Content: "Good catch. The short answer is this is how I handled it. If needed I can make a dedicated follow-up."},
			{Channel: channel, Title: "Take feedback", Scene: "Feedback", Content: "Got it. This feedback is useful. I will keep improving the next version."},
		}
	default:
		return nil
	}
}

func (s *ReplyWorkspaceService) defaultExampleTitle(target *ReplyWorkspaceTarget) string {
	if target == nil {
		return "example"
	}
	if target.Channel == ReplyChannelMessage {
		return fmt.Sprintf("dm-example-%s", target.AuthorName)
	}
	return fmt.Sprintf("comment-example-%s", target.AuthorName)
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
