package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	appruntime "bilibili-up-admin/internal/runtime"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

// MessageService 私信服务
type MessageService struct {
	runtime      *appruntime.Store
	repo         *repository.MessageRepository
	llmLogRepo   *repository.LLMChatLogRepository
	fanReplyRepo *repository.FanAutoReplyRecordRepository
}

// NewMessageService 创建私信服务
func NewMessageService(
	runtime *appruntime.Store,
	repo *repository.MessageRepository,
	llmLogRepo *repository.LLMChatLogRepository,
	fanReplyRepo *repository.FanAutoReplyRecordRepository,
) *MessageService {
	return &MessageService{
		runtime:      runtime,
		repo:         repo,
		llmLogRepo:   llmLogRepo,
		fanReplyRepo: fanReplyRepo,
	}
}

type FollowAutoReplySummary struct {
	ScannedFans int `json:"scanned_fans"`
	NewFans     int `json:"new_fans"`
	Replied     int `json:"replied"`
	Failed      int `json:"failed"`
	Seeded      int `json:"seeded"`
}

func (s *MessageService) biliClient() (*bilibili.Client, error) {
	if s.runtime == nil || s.runtime.BilibiliClient() == nil {
		return nil, fmt.Errorf("bilibili login is not configured")
	}
	return s.runtime.BilibiliClient(), nil
}

func (s *MessageService) llmProvider() (llm.Provider, error) {
	if s.runtime == nil || s.runtime.LLMManager() == nil {
		return nil, fmt.Errorf("llm is not configured")
	}
	return s.runtime.LLMManager().Default()
}

// MessageListResult 私信列表结果
type MessageListResult struct {
	Messages []model.Message `json:"messages"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type MessageSyncResult struct {
	Inserted       int      `json:"inserted"`
	Sessions       int      `json:"sessions"`
	Fetched        int      `json:"fetched"`
	Existing       int      `json:"existing"`
	SessionErrors  int      `json:"session_errors"`
	InsertErrors   int      `json:"insert_errors"`
	ErrorSummaries []string `json:"error_summaries,omitempty"`
}

func resolveSelfIdentity(ctx context.Context, client *bilibili.Client) (int64, string) {
	if client == nil {
		return 0, ""
	}
	if cfg := client.GetConfig(); cfg != nil && cfg.UserID > 0 {
		return cfg.UserID, "我"
	}
	user, err := client.GetUserInfo(ctx)
	if err != nil || user == nil {
		return 0, "我"
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = "我"
	}
	return user.Mid, name
}

func buildMessageContext(m bilibili.Message, session bilibili.MessageSession, selfUID int64, selfName string) (bool, string, string, int64, string, string) {
	isFromSelf := selfUID > 0 && m.SenderID == selfUID

	senderName := strings.TrimSpace(m.SenderName)
	senderFace := strings.TrimSpace(m.SenderFace)
	conversationUID := session.UserID
	conversationName := strings.TrimSpace(session.UserName)
	conversationFace := strings.TrimSpace(session.UserFace)

	if isFromSelf {
		if senderName == "" {
			if selfName != "" {
				senderName = selfName
			} else {
				senderName = "我"
			}
		}
	} else {
		if senderName == "" {
			if conversationName != "" {
				senderName = conversationName
			} else {
				senderName = fmt.Sprintf("用户%d", m.SenderID)
			}
		}
		if senderFace == "" {
			senderFace = conversationFace
		}
	}

	if conversationUID == 0 && !isFromSelf {
		conversationUID = m.SenderID
	}
	if conversationName == "" {
		if !isFromSelf && senderName != "" {
			conversationName = senderName
		} else if conversationUID > 0 {
			conversationName = fmt.Sprintf("用户%d", conversationUID)
		}
	}
	if conversationFace == "" && !isFromSelf {
		conversationFace = senderFace
	}

	return isFromSelf, senderName, senderFace, conversationUID, conversationName, conversationFace
}

func applyMessageContext(record *model.Message, isFromSelf bool, senderName, senderFace string, conversationUID int64, conversationName, conversationFace string, messageTime time.Time) bool {
	changed := false
	if record.IsFromSelf != isFromSelf {
		record.IsFromSelf = isFromSelf
		changed = true
	}
	if senderName != "" && record.SenderName != senderName {
		record.SenderName = senderName
		changed = true
	}
	if senderFace != "" && record.SenderFace != senderFace {
		record.SenderFace = senderFace
		changed = true
	}
	if conversationUID > 0 && record.ConversationUID != conversationUID {
		record.ConversationUID = conversationUID
		changed = true
	}
	if conversationName != "" && record.ConversationName != conversationName {
		record.ConversationName = conversationName
		changed = true
	}
	if conversationFace != "" && record.ConversationFace != conversationFace {
		record.ConversationFace = conversationFace
		changed = true
	}
	if record.MessageTime == nil {
		record.MessageTime = &messageTime
		changed = true
	}
	return changed
}

// List 获取私信列表
func (s *MessageService) List(ctx context.Context, senderID int64, replyStatus int, page, pageSize int) (*MessageListResult, error) {
	messages, total, err := s.repo.List(ctx, senderID, replyStatus, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get messages failed: %w", err)
	}

	return &MessageListResult{
		Messages: messages,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// SyncMessages 同步私信
func (s *MessageService) SyncMessages(ctx context.Context, page, pageSize int) (*MessageSyncResult, error) {
	start := time.Now()
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}

	selfUID, selfName := resolveSelfIdentity(ctx, client)

	list, err := client.GetMessages(ctx, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get messages failed: %w", err)
	}

	result := &MessageSyncResult{
		Sessions:       len(list.Sessions),
		ErrorSummaries: make([]string, 0, 5),
	}

	for _, session := range list.Sessions {
		// 获取聊天记录
		chat, err := client.GetChatHistory(ctx, session.UserID, 1, 20)
		if err != nil {
			log.Printf("[message.sync] chat_history_failed uid=%d err=%v", session.UserID, err)
			result.SessionErrors++
			if len(result.ErrorSummaries) < 5 {
				result.ErrorSummaries = append(result.ErrorSummaries, fmt.Sprintf("uid=%d chat failed: %v", session.UserID, err))
			}
			continue
		}
		result.Fetched += len(chat.Messages)

		for _, m := range chat.Messages {
			isFromSelf, senderName, senderFace, conversationUID, conversationName, conversationFace := buildMessageContext(m, session, selfUID, selfName)
			messageTime := time.Unix(m.Time, 0)
			existing, _ := s.repo.GetByMessageID(ctx, m.ID)
			if existing != nil {
				if applyMessageContext(existing, isFromSelf, senderName, senderFace, conversationUID, conversationName, conversationFace, messageTime) {
					if err := s.repo.Update(ctx, existing); err != nil {
						log.Printf("[message.sync] update_context_failed msg_id=%d err=%v", m.ID, err)
					}
				}
				result.Existing++
				continue
			}

			message := &model.Message{
				MessageID:        m.ID,
				SenderID:         m.SenderID,
				SenderName:       senderName,
				SenderFace:       senderFace,
				ConversationUID:  conversationUID,
				ConversationName: conversationName,
				ConversationFace: conversationFace,
				IsFromSelf:       isFromSelf,
				Content:          m.Content,
				ReplyStatus:      map[bool]int{true: 1, false: 0}[isFromSelf],
				MessageTime:      &messageTime,
			}

			if err := s.repo.Create(ctx, message); err != nil {
				if isDuplicateMessageError(err) {
					result.Existing++
					continue
				}
				log.Printf("[message.sync] create_failed msg_id=%d sender_uid=%d err=%v", m.ID, m.SenderID, err)
				result.InsertErrors++
				if len(result.ErrorSummaries) < 5 {
					result.ErrorSummaries = append(result.ErrorSummaries, fmt.Sprintf("msg=%d insert failed: %v", m.ID, err))
				}
				continue
			}
			result.Inserted++
		}
	}

	log.Printf("[message.sync] done page=%d page_size=%d sessions=%d fetched=%d inserted=%d existing=%d session_errors=%d insert_errors=%d cost_ms=%d", page, pageSize, result.Sessions, result.Fetched, result.Inserted, result.Existing, result.SessionErrors, result.InsertErrors, time.Since(start).Milliseconds())

	return result, nil
}

func isDuplicateMessageError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

// AIReply 使用AI生成回复
func (s *MessageService) AIReply(ctx context.Context, messageID int64) (string, error) {
	message, err := s.repo.GetByMessageID(ctx, messageID)
	if err != nil {
		return "", fmt.Errorf("get message failed: %w", err)
	}

	if message.ReplyStatus != 0 {
		return "", fmt.Errorf("message already replied")
	}

	// AI生成回复
	systemPrompt := `你是一个B站UP主的助手。请根据用户的私信生成一个友善、有帮助的回复。

要求：
1. 回复要友善、有礼貌
2. 尽量帮助用户解决问题
3. 如果是商业合作，引导对方留下联系方式
4. 回复长度适中，不要太长或太短
5. 保持专业但亲切的语气

请直接输出回复内容。`

	messages := []llm.Message{
		{Role: "user", Content: fmt.Sprintf("用户私信：%s\n\n请生成回复：", message.Content)},
	}

	startTime := time.Now()
	provider, err := s.llmProvider()
	if err != nil {
		return "", err
	}
	resp, err := provider.ChatWithSystem(ctx, systemPrompt, messages)
	if err != nil {
		return "", fmt.Errorf("llm chat failed: %w", err)
	}
	duration := time.Since(startTime).Milliseconds()

	// 记录日志
	conversationMeta := llmConversationForMessage(message)
	log := &model.LLMChatLog{
		Provider:          provider.Name(),
		Model:             resp.Model,
		InputType:         "message",
		InputID:           messageID,
		ConversationKey:   conversationMeta.Key,
		ConversationTitle: conversationMeta.Title,
		InputContent:      message.Content,
		SystemPrompt:      systemPrompt,
		RequestMessages:   marshalLLMMessages(messages),
		OutputContent:     resp.Content,
		PromptTokens:      resp.PromptTokens,
		OutputTokens:      resp.TokensUsed,
		TotalTokens:       resp.TotalTokens,
		Success:           true,
		Duration:          duration,
	}
	s.llmLogRepo.Create(ctx, log)

	// 发送回复
	client, err := s.biliClient()
	if err != nil {
		return "", err
	}
	err = client.SendMessage(ctx, message.SenderID, resp.Content)
	if err != nil {
		return "", fmt.Errorf("send message failed: %w", err)
	}

	// 更新状态
	err = s.repo.UpdateReplyStatus(ctx, messageID, 1, resp.Content, true)
	if err != nil {
		return "", fmt.Errorf("update message status failed: %w", err)
	}

	return resp.Content, nil
}

// ManualReply 手动回复
func (s *MessageService) ManualReply(ctx context.Context, messageID int64, senderID int64, content string) error {
	// 发送回复
	client, err := s.biliClient()
	if err != nil {
		return err
	}
	err = client.SendMessage(ctx, senderID, content)
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}

	// 更新状态
	if err := s.repo.UpdateReplyStatus(ctx, messageID, 1, content, false); err == nil {
		return nil
	}

	// 兜底：如果该消息尚未入库，则补一条最小记录
	selfUID, selfName := resolveSelfIdentity(ctx, client)
	now := time.Now()
	conversationName := fmt.Sprintf("用户%d", senderID)
	if senderID == 0 {
		conversationName = "会话对象"
	}
	return s.repo.Create(ctx, &model.Message{
		MessageID:        messageID,
		SenderID:         selfUID,
		SenderName:       selfName,
		ConversationUID:  senderID,
		ConversationName: conversationName,
		IsFromSelf:       true,
		Content:          content,
		ReplyStatus:      1,
		ReplyContent:     content,
		MessageTime:      &now,
	})
}

// Ignore 忽略私信
func (s *MessageService) Ignore(ctx context.Context, messageID int64) error {
	return s.repo.UpdateReplyStatus(ctx, messageID, 2, "", false)
}

// GetUnreadCount 获取未读数量
func (s *MessageService) GetUnreadCount(ctx context.Context) (int, error) {
	client, err := s.biliClient()
	if err != nil {
		return 0, err
	}
	return client.GetUnreadMessageCount(ctx)
}

func (s *MessageService) AutoReplyNewFollowers(ctx context.Context, rules InteractionRuleSettings) (*FollowAutoReplySummary, error) {
	summary := &FollowAutoReplySummary{}
	if s.fanReplyRepo == nil || !rules.EnableFollowAutoReply {
		log.Printf("[fans.auto_reply] skipped enabled=%v repo_ready=%v", rules.EnableFollowAutoReply, s.fanReplyRepo != nil)
		return summary, nil
	}
	content := strings.TrimSpace(rules.FollowAutoReplyContent)
	if content == "" {
		log.Printf("[fans.auto_reply] skipped empty_content")
		return summary, nil
	}

	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}

	if rules.FanPageSize <= 0 {
		rules.FanPageSize = 20
	}
	if rules.RequestIntervalSeconds <= 0 {
		rules.RequestIntervalSeconds = 3
	}
	interval := time.Duration(rules.RequestIntervalSeconds) * time.Second

	const followWindow = 10 * time.Minute
	recordCount, err := s.fanReplyRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("[fans.auto_reply] start fan_page_size=%d interval_sec=%d follow_window_min=10 record_count=%d", rules.FanPageSize, rules.RequestIntervalSeconds, recordCount)
	digest := sha256.Sum256([]byte(content))
	replyDigest := hex.EncodeToString(digest[:])

	for page := 1; page <= 3; page++ {
		fans, err := client.ListFans(ctx, page, rules.FanPageSize)
		if err != nil {
			return nil, err
		}
		if len(fans) == 0 {
			break
		}

		now := time.Now()
		summary.ScannedFans += len(fans)
		for _, fan := range fans {
			nowUnix := now.Unix()
			withinWindow := fan.FollowTime > 0 && nowUnix >= fan.FollowTime && (nowUnix-fan.FollowTime) <= int64(followWindow/time.Second)

			record, err := s.fanReplyRepo.GetByFanUID(ctx, fan.UserID)
			if err != nil {
				log.Printf("[fans.auto_reply] lookup_failed uid=%d err=%v", fan.UserID, err)
				continue
			}

			if record == nil {
				record = &model.FanAutoReplyRecord{
					FanUID:     fan.UserID,
					FanName:    fan.UserName,
					LastSeenAt: &now,
					Replied:    false,
				}
				if err := s.fanReplyRepo.Create(ctx, record); err != nil {
					log.Printf("[fans.auto_reply] create_record_failed uid=%d err=%v", fan.UserID, err)
					continue
				}
				summary.NewFans++
				log.Printf("[fans.auto_reply] discovered uid=%d uname=%q follow_time=%d within_window=%v", fan.UserID, fan.UserName, fan.FollowTime, withinWindow)
			}

			record.FanName = fan.UserName
			record.LastSeenAt = &now
			if !withinWindow {
				record.LastError = "follow_time_outside_10m_window"
				record.Replied = false
				record.RepliedAt = nil
				record.ReplyDigest = ""
				_ = s.fanReplyRepo.Update(ctx, record)
				continue
			}
			if record.Replied && record.ReplyDigest == replyDigest {
				_ = s.fanReplyRepo.Update(ctx, record)
				continue
			}

			if err := client.SendMessage(ctx, fan.UserID, content); err != nil {
				log.Printf("[fans.auto_reply] send_failed uid=%d uname=%q err=%v", fan.UserID, fan.UserName, err)
				record.LastError = err.Error()
				record.Replied = false
				record.RepliedAt = nil
				record.ReplyDigest = ""
				summary.Failed++
				_ = s.fanReplyRepo.Update(ctx, record)
				continue
			}

			record.LastError = ""
			record.Replied = true
			record.RepliedAt = &now
			record.ReplyDigest = replyDigest
			summary.Replied++
			log.Printf("[fans.auto_reply] replied uid=%d uname=%q", fan.UserID, fan.UserName)
			_ = s.fanReplyRepo.Update(ctx, record)
			time.Sleep(interval)
		}
	}

	log.Printf("[fans.auto_reply] done scanned=%d new=%d replied=%d failed=%d seeded=%d", summary.ScannedFans, summary.NewFans, summary.Replied, summary.Failed, summary.Seeded)

	return summary, nil
}
