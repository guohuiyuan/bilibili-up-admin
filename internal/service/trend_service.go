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

const DefaultTrendCacheTTL = 30 * time.Minute

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
func (s *TrendService) GetTrendingTags(ctx context.Context, category string, limit int) ([]bilibili.TrendingTag, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	return client.GetTrendingTagsByCategory(ctx, category, limit)
}

// GetTrendingTagsSmart 优先读取缓存，不存在或过期时回源并刷新
func (s *TrendService) GetTrendingTagsSmart(ctx context.Context, category string, limit int, ttl time.Duration) ([]bilibili.TrendingTag, error) {
	rankings, _, err := s.EnsureLatestTags(ctx, category, limit, ttl)
	if err != nil {
		return nil, err
	}
	out := make([]bilibili.TrendingTag, 0, len(rankings))
	for _, row := range rankings {
		out = append(out, bilibili.TrendingTag{
			Name:     row.TagName,
			HotValue: row.HotValue,
			Rank:     row.Rank,
			Category: row.Category,
		})
	}
	return out, nil
}

// EnsureLatestTags 确保缓存可用，必要时刷新；返回缓存内容与是否发生刷新
func (s *TrendService) EnsureLatestTags(ctx context.Context, category string, limit int, ttl time.Duration) ([]model.TagRanking, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if ttl <= 0 {
		ttl = DefaultTrendCacheTTL
	}

	latestAt, err := s.repo.LatestRecordAt(ctx, category)
	if err == nil && latestAt != nil && time.Since(*latestAt) <= ttl {
		cached, listErr := s.repo.GetLatestByCategory(ctx, category, limit)
		if listErr == nil && len(cached) > 0 {
			return cached, false, nil
		}
	}

	tags, fetchErr := s.GetTrendingTags(ctx, category, limit)
	if fetchErr != nil {
		cached, listErr := s.repo.GetLatestByCategory(ctx, category, limit)
		if listErr == nil && len(cached) > 0 {
			return cached, false, nil
		}
		return nil, false, fetchErr
	}
	if saveErr := s.SaveTagRankings(ctx, tags); saveErr != nil {
		return nil, false, saveErr
	}
	cached, err := s.repo.GetLatestByCategory(ctx, category, limit)
	if err != nil {
		return nil, true, err
	}
	return cached, true, nil
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
	_, _, err := s.EnsureLatestTags(ctx, "", 50, 0)
	if err != nil {
		return fmt.Errorf("daily sync trending tags failed: %w", err)
	}
	return nil
}
