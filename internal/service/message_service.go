package service

import (
	"context"
	"fmt"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	appruntime "bilibili-up-admin/internal/runtime"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

// MessageService 私信服务
type MessageService struct {
	runtime    *appruntime.Store
	repo       *repository.MessageRepository
	llmLogRepo *repository.LLMChatLogRepository
}

// NewMessageService 创建私信服务
func NewMessageService(
	runtime *appruntime.Store,
	repo *repository.MessageRepository,
	llmLogRepo *repository.LLMChatLogRepository,
) *MessageService {
	return &MessageService{
		runtime:    runtime,
		repo:       repo,
		llmLogRepo: llmLogRepo,
	}
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
func (s *MessageService) SyncMessages(ctx context.Context, page, pageSize int) (int, error) {
	client, err := s.biliClient()
	if err != nil {
		return 0, err
	}
	list, err := client.GetMessages(ctx, page, pageSize)
	if err != nil {
		return 0, fmt.Errorf("get messages failed: %w", err)
	}

	count := 0
	for _, session := range list.Sessions {
		// 获取聊天记录
		chat, err := client.GetChatHistory(ctx, session.UserID, 1, 20)
		if err != nil {
			continue
		}

		for _, m := range chat.Messages {
			existing, _ := s.repo.GetByMessageID(ctx, m.ID)
			if existing != nil {
				continue
			}

			message := &model.Message{
				MessageID:   m.ID,
				SenderID:    m.SenderID,
				SenderName:  m.SenderName,
				Content:     m.Content,
				ReplyStatus: 0,
				IsRead:      m.IsRead,
				MessageTime: time.Unix(m.Time, 0),
			}

			if err := s.repo.Create(ctx, message); err != nil {
				continue
			}
			count++
		}
	}

	return count, nil
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
	log := &model.LLMChatLog{
		Provider:      provider.Name(),
		InputType:     "message",
		InputID:       messageID,
		InputContent:  message.Content,
		OutputContent: resp.Content,
		PromptTokens:  resp.PromptTokens,
		OutputTokens:  resp.TokensUsed,
		TotalTokens:   resp.TotalTokens,
		Success:       true,
		Duration:      duration,
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
	message.ReplyStatus = 1
	message.ReplyContent = resp.Content
	message.IsAIReply = true
	s.repo.Create(ctx, message)

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
	return s.repo.Create(ctx, &model.Message{
		MessageID:    messageID,
		SenderID:     senderID,
		ReplyStatus:  1,
		ReplyContent: content,
	})
}

// Ignore 忽略私信
func (s *MessageService) Ignore(ctx context.Context, messageID int64) error {
	message, err := s.repo.GetByMessageID(ctx, messageID)
	if err != nil {
		return err
	}
	message.ReplyStatus = 2
	return s.repo.Create(ctx, message)
}

// GetUnreadCount 获取未读数量
func (s *MessageService) GetUnreadCount(ctx context.Context) (int, error) {
	client, err := s.biliClient()
	if err != nil {
		return 0, err
	}
	return client.GetUnreadMessageCount(ctx)
}
