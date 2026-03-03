package service

import (
	"context"
	"fmt"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

// CommentService 评论服务
type CommentService struct {
	biliClient  *bilibili.Client
	llmProvider llm.Provider
	repo        *repository.CommentRepository
	llmLogRepo  *repository.LLMChatLogRepository
}

// NewCommentService 创建评论服务
func NewCommentService(
	biliClient *bilibili.Client,
	llmProvider llm.Provider,
	repo *repository.CommentRepository,
	llmLogRepo *repository.LLMChatLogRepository,
) *CommentService {
	return &CommentService{
		biliClient:  biliClient,
		llmProvider: llmProvider,
		repo:        repo,
		llmLogRepo:  llmLogRepo,
	}
}

// CommentListResult 评论列表结果
type CommentListResult struct {
	Comments []model.Comment `json:"comments"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// List 获取评论列表
func (s *CommentService) List(ctx context.Context, videoBVID string, replyStatus int, page, pageSize int) (*CommentListResult, error) {
	comments, total, err := s.repo.List(ctx, videoBVID, replyStatus, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get comments failed: %w", err)
	}

	return &CommentListResult{
		Comments: comments,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// SyncFromVideo 从视频同步评论
func (s *CommentService) SyncFromVideo(ctx context.Context, bvID string, page, pageSize int) (int, error) {
	list, err := s.biliClient.GetVideoComments(ctx, bvID, page, pageSize)
	if err != nil {
		return 0, fmt.Errorf("get video comments failed: %w", err)
	}

	count := 0
	for _, c := range list.Comments {
		// 检查是否已存在
		existing, _ := s.repo.GetByCommentID(ctx, c.ID)
		if existing != nil {
			continue
		}

		comment := &model.Comment{
			CommentID:   c.ID,
			VideoBVID:   c.VideoID,
			VideoAID:    c.VideoAID,
			Content:     c.Content,
			AuthorID:    c.AuthorID,
			AuthorName:  c.Author,
			ReplyStatus: 0,
			CommentTime: time.Unix(c.Time, 0),
		}

		if err := s.repo.Create(ctx, comment); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// AIReply 使用AI生成回复
func (s *CommentService) AIReply(ctx context.Context, commentID int64) (string, error) {
	// 获取评论
	comment, err := s.repo.GetByCommentID(ctx, commentID)
	if err != nil {
		return "", fmt.Errorf("get comment failed: %w", err)
	}

	if comment.ReplyStatus != 0 {
		return "", fmt.Errorf("comment already replied")
	}

	// AI生成回复
	systemPrompt := `你是一个B站UP主的助手。请根据用户的评论生成一个友善、有趣的回复。

要求：
1. 回复要友善、有礼貌，表示感谢
2. 可以适当使用B站特色用语，如"感谢投喂"、"妙啊"等
3. 回复长度控制在20-80字之间
4. 如果是提问，尽量给出有用的回答
5. 不要使用过于生硬的语言

请直接输出回复内容，不要有任何前缀或说明。`

	messages := []llm.Message{
		{Role: "user", Content: fmt.Sprintf("用户评论：%s\n\n请生成回复：", comment.Content)},
	}

	startTime := time.Now()
	resp, err := s.llmProvider.ChatWithSystem(ctx, systemPrompt, messages)
	if err != nil {
		return "", fmt.Errorf("llm chat failed: %w", err)
	}
	duration := time.Since(startTime).Milliseconds()

	// 记录日志
	log := &model.LLMChatLog{
		Provider:      s.llmProvider.Name(),
		InputType:     "comment",
		InputID:       commentID,
		InputContent:  comment.Content,
		OutputContent: resp.Content,
		PromptTokens:  resp.PromptTokens,
		OutputTokens:  resp.TokensUsed,
		TotalTokens:   resp.TotalTokens,
		Success:       true,
		Duration:      duration,
	}
	s.llmLogRepo.Create(ctx, log)

	// 发送回复到B站
	err = s.biliClient.ReplyComment(ctx, comment.VideoAID, commentID, resp.Content)
	if err != nil {
		return "", fmt.Errorf("send reply failed: %w", err)
	}

	// 更新状态
	err = s.repo.UpdateReplyStatus(ctx, commentID, 1, resp.Content)
	if err != nil {
		return "", fmt.Errorf("update reply status failed: %w", err)
	}

	// 更新评论记录
	comment.IsAIReply = true
	s.repo.Update(ctx, comment)

	return resp.Content, nil
}

// ManualReply 手动回复
func (s *CommentService) ManualReply(ctx context.Context, commentID int64, content string) error {
	comment, err := s.repo.GetByCommentID(ctx, commentID)
	if err != nil {
		return fmt.Errorf("get comment failed: %w", err)
	}

	// 发送回复
	err = s.biliClient.ReplyComment(ctx, comment.VideoAID, commentID, content)
	if err != nil {
		return fmt.Errorf("send reply failed: %w", err)
	}

	// 更新状态
	return s.repo.UpdateReplyStatus(ctx, commentID, 1, content)
}

// Ignore 忽略评论
func (s *CommentService) Ignore(ctx context.Context, commentID int64) error {
	return s.repo.UpdateReplyStatus(ctx, commentID, 2, "")
}

// BatchAIReply 批量AI回复
func (s *CommentService) BatchAIReply(ctx context.Context, limit int) (int, error) {
	comments, err := s.repo.GetUnreplied(ctx, limit)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, c := range comments {
		_, err := s.AIReply(ctx, c.CommentID)
		if err != nil {
			continue
		}
		count++
		// 添加延迟避免请求过快
		time.Sleep(time.Second * 2)
	}

	return count, nil
}
