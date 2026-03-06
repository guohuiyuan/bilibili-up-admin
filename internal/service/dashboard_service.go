package service

import (
	"context"
	"time"

	"bilibili-up-admin/internal/repository"
)

type DashboardSummary struct {
	CommentCount     int64     `json:"comment_count"`
	MessageCount     int64     `json:"message_count"`
	InteractionCount int64     `json:"interaction_count"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
}

type DashboardService struct {
	comments     *repository.CommentRepository
	messages     *repository.MessageRepository
	interactions *repository.InteractionRepository
}

func NewDashboardService(
	comments *repository.CommentRepository,
	messages *repository.MessageRepository,
	interactions *repository.InteractionRepository,
) *DashboardService {
	return &DashboardService{
		comments:     comments,
		messages:     messages,
		interactions: interactions,
	}
}

func (s *DashboardService) TodaySummary(ctx context.Context) (*DashboardSummary, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	commentCount, err := s.comments.CountBetween(ctx, start, now)
	if err != nil {
		return nil, err
	}
	messageCount, err := s.messages.CountBetween(ctx, start, now)
	if err != nil {
		return nil, err
	}
	interactionStats, err := s.interactions.GetStats(ctx, start, now)
	if err != nil {
		return nil, err
	}

	return &DashboardSummary{
		CommentCount:     commentCount,
		MessageCount:     messageCount,
		InteractionCount: interactionStats["like"] + interactionStats["coin"] + interactionStats["favorite"],
		StartTime:        start,
		EndTime:          now,
	}, nil
}
