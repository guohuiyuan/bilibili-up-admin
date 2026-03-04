package service

import (
	"context"
	"fmt"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	appruntime "bilibili-up-admin/internal/runtime"
	"bilibili-up-admin/pkg/bilibili"
)

// TrendService 热度服务
type TrendService struct {
	runtime *appruntime.Store
	repo    *repository.TagRankingRepository
}

// NewTrendService 创建热度服务
func NewTrendService(
	runtime *appruntime.Store,
	repo *repository.TagRankingRepository,
) *TrendService {
	return &TrendService{
		runtime: runtime,
		repo:    repo,
	}
}

func (s *TrendService) biliClient() (*bilibili.Client, error) {
	if s.runtime == nil || s.runtime.BilibiliClient() == nil {
		return nil, fmt.Errorf("bilibili login is not configured")
	}
	return s.runtime.BilibiliClient(), nil
}

// TagRankingResult 标签排行结果
type TagRankingResult struct {
	Tags     []model.TagRanking `json:"tags"`
	Date     string             `json:"date"`
	Category string             `json:"category"`
}

// GetTrendingTags 获取热门标签
func (s *TrendService) GetTrendingTags(ctx context.Context, limit int) ([]bilibili.TrendingTag, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	return client.GetTrendingTags(ctx, limit)
}

// GetTagDetail 获取标签详情
func (s *TrendService) GetTagDetail(ctx context.Context, tagName string, page, pageSize int) (*bilibili.TagRanking, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	return client.GetTagRanking(ctx, tagName, page, pageSize)
}

// GetVideoRanking 获取视频排行
func (s *TrendService) GetVideoRanking(ctx context.Context, category string, limit int) (*bilibili.VideoRanking, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	return client.GetCategoryRanking(ctx, category, limit)
}

// SaveTagRankings 保存标签排行
func (s *TrendService) SaveTagRankings(ctx context.Context, tags []bilibili.TrendingTag) error {
	now := time.Now()
	rankings := make([]model.TagRanking, 0, len(tags))

	for i, tag := range tags {
		rankings = append(rankings, model.TagRanking{
			TagName:    tag.Name,
			HotValue:   tag.HotValue,
			Rank:       i + 1,
			Category:   tag.Category,
			RecordDate: now,
		})
	}

	return s.repo.BatchCreate(ctx, rankings)
}

// GetHistoricalRankings 获取历史排行
func (s *TrendService) GetHistoricalRankings(ctx context.Context, date string, limit int) ([]model.TagRanking, error) {
	recordDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	return s.repo.ListByDate(ctx, recordDate, limit)
}

// GetLatestRankings 获取最新排行
func (s *TrendService) GetLatestRankings(ctx context.Context, limit int) ([]model.TagRanking, error) {
	return s.repo.GetLatest(ctx, limit)
}

// TrendStats 热度统计
type TrendStats struct {
	TotalTags    int64  `json:"total_tags"`
	TotalRecords int64  `json:"total_records"`
	LatestDate   string `json:"latest_date"`
}

// GetStats 获取热度统计
func (s *TrendService) GetStats(ctx context.Context) (*TrendStats, error) {
	var totalTags int64
	var totalRecords int64
	var latestDate time.Time

	// 这里简化处理，实际可以使用SQL查询
	rankings, _ := s.repo.GetLatest(ctx, 1)
	if len(rankings) > 0 {
		latestDate = rankings[0].RecordDate
	}

	return &TrendStats{
		TotalTags:    totalTags,
		TotalRecords: totalRecords,
		LatestDate:   latestDate.Format("2006-01-02"),
	}, nil
}

// SearchTag 搜索标签
func (s *TrendService) SearchTag(ctx context.Context, keyword string, page, pageSize int) ([]bilibili.TagRanking, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	return client.SearchTag(ctx, keyword, page, pageSize)
}

// VideoInfo 视频信息
type VideoInfo struct {
	BVID     string   `json:"bvid"`
	Title    string   `json:"title"`
	Owner    string   `json:"owner"`
	OwnerID  int64    `json:"owner_id"`
	Duration int      `json:"duration"`
	View     int      `json:"view"`
	Like     int      `json:"like"`
	Coin     int      `json:"coin"`
	Tags     []string `json:"tags"`
	Pic      string   `json:"pic"`
}

// GetVideoInfo 获取视频信息
func (s *TrendService) GetVideoInfo(ctx context.Context, bvID string) (*bilibili.VideoInfo, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	return client.GetVideoInfo(ctx, bvID)
}

// DailySync 每日同步热度数据
func (s *TrendService) DailySync(ctx context.Context) error {
	// 获取热门标签
	client, err := s.biliClient()
	if err != nil {
		return err
	}
	tags, err := client.GetTrendingTags(ctx, 50)
	if err != nil {
		return fmt.Errorf("get trending tags failed: %w", err)
	}

	// 保存到数据库
	return s.SaveTagRankings(ctx, tags)
}
