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

// InteractionService 互动服务
type InteractionService struct {
	runtime *appruntime.Store
	repo    *repository.InteractionRepository
}

// NewInteractionService 创建互动服务
func NewInteractionService(
	runtime *appruntime.Store,
	repo *repository.InteractionRepository,
) *InteractionService {
	return &InteractionService{
		runtime: runtime,
		repo:    repo,
	}
}

func (s *InteractionService) biliClient() (*bilibili.Client, error) {
	if s.runtime == nil || s.runtime.BilibiliClient() == nil {
		return nil, fmt.Errorf("bilibili login is not configured")
	}
	return s.runtime.BilibiliClient(), nil
}

// InteractionResult 互动结果
type InteractionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type FanVideoSelection struct {
	FanID    int64                `json:"fan_id"`
	FanName  string               `json:"fan_name"`
	Videos   []bilibili.VideoInfo `json:"videos"`
	VideoCnt int                  `json:"video_cnt"`
}

type FanListResult struct {
	Items    []bilibili.FanProfile `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	HasMore  bool                  `json:"has_more"`
}

type FanVideosResult struct {
	FanID    int64                `json:"fan_id"`
	Items    []bilibili.VideoInfo `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	HasMore  bool                 `json:"has_more"`
}

type AutoInteractSummary struct {
	LikedCount     int `json:"liked_count"`
	CoinedCount    int `json:"coined_count"`
	FavoritedCount int `json:"favorited_count"`
	TotalCount     int `json:"total_count"`
}

type VideoEngagementSnapshot struct {
	Video    *bilibili.VideoInfo           `json:"video"`
	Relation *bilibili.VideoRelationStatus `json:"relation"`
	UPCoin   float64                       `json:"up_coin"`
	SyncedAt time.Time                     `json:"synced_at"`
}

// LikeVideo 点赞视频
func (s *InteractionService) LikeVideo(ctx context.Context, bvID string) (*InteractionResult, error) {
	// 检查是否已点赞
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	liked, err := client.IsLiked(ctx, bvID)
	if err != nil {
		return nil, err
	}
	if liked {
		return &InteractionResult{Success: false, Message: "已经点赞过了"}, nil
	}

	// 获取视频信息
	info, err := client.GetVideoInfo(ctx, bvID)
	if err != nil {
		return nil, err
	}

	// 点赞
	result, err := client.LikeVideo(ctx, bvID)
	if err != nil {
		return nil, err
	}

	// 记录
	s.repo.Create(ctx, &model.Interaction{
		VideoBVID:    bvID,
		VideoTitle:   info.Title,
		VideoOwnerID: info.OwnerID,
		VideoOwner:   info.Owner,
		ActionType:   "like",
		Success:      result.Success,
		ErrorMessage: result.Message,
		ActionTime:   &[]time.Time{time.Now()}[0],
	})

	return &InteractionResult{Success: result.Success, Message: result.Message}, nil
}

// CoinVideo 投币视频
func (s *InteractionService) CoinVideo(ctx context.Context, bvID string, coinCount int) (*InteractionResult, error) {
	// 检查是否已投币
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	coined, err := client.IsCoined(ctx, bvID)
	if err != nil {
		return nil, err
	}
	if coined {
		return &InteractionResult{Success: false, Message: "已经投币过了"}, nil
	}

	// 获取视频信息
	info, err := client.GetVideoInfo(ctx, bvID)
	if err != nil {
		return nil, err
	}

	// 投币
	result, err := client.CoinVideo(ctx, bvID, coinCount)
	if err != nil {
		return nil, err
	}

	// 记录
	s.repo.Create(ctx, &model.Interaction{
		VideoBVID:    bvID,
		VideoTitle:   info.Title,
		VideoOwnerID: info.OwnerID,
		VideoOwner:   info.Owner,
		ActionType:   "coin",
		CoinCount:    result.Coins,
		Success:      result.Success,
		ErrorMessage: result.Message,
		ActionTime:   &[]time.Time{time.Now()}[0],
	})

	return &InteractionResult{Success: result.Success, Message: result.Message}, nil
}

func (s *InteractionService) FavoriteVideo(ctx context.Context, bvID string, mediaID int64) (*InteractionResult, error) {
	if mediaID <= 0 {
		return &InteractionResult{Success: false, Message: "收藏夹ID无效"}, nil
	}

	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	relation, err := client.GetVideoRelationStatus(ctx, bvID)
	if err != nil {
		return nil, err
	}
	if relation.Favorite {
		return &InteractionResult{Success: false, Message: "已经收藏过了"}, nil
	}

	info, err := client.GetVideoInfo(ctx, bvID)
	if err != nil {
		return nil, err
	}

	if err := client.FavoriteVideo(ctx, bvID, mediaID); err != nil {
		return &InteractionResult{Success: false, Message: err.Error()}, nil
	}

	s.repo.Create(ctx, &model.Interaction{
		VideoBVID:    bvID,
		VideoTitle:   info.Title,
		VideoOwnerID: info.OwnerID,
		VideoOwner:   info.Owner,
		ActionType:   "favorite",
		Success:      true,
		ActionTime:   &[]time.Time{time.Now()}[0],
	})

	return &InteractionResult{Success: true, Message: "收藏成功"}, nil
}

// TripleAction 三连
func (s *InteractionService) TripleAction(ctx context.Context, bvID string) (*InteractionResult, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	info, err := client.GetVideoInfo(ctx, bvID)
	if err != nil {
		return nil, err
	}

	err = client.TripleAction(ctx, bvID)
	if err != nil {
		return &InteractionResult{Success: false, Message: err.Error()}, nil
	}

	// 记录
	s.repo.Create(ctx, &model.Interaction{
		VideoBVID:    bvID,
		VideoTitle:   info.Title,
		VideoOwnerID: info.OwnerID,
		VideoOwner:   info.Owner,
		ActionType:   "triple",
		Success:      true,
		ActionTime:   &[]time.Time{time.Now()}[0],
	})

	return &InteractionResult{Success: true, Message: "三连成功"}, nil
}

// BatchInteract 批量互动
func (s *InteractionService) BatchInteract(ctx context.Context, bvIDs []string, actionType string, coinCount int) ([]InteractionResult, error) {
	results := make([]InteractionResult, 0, len(bvIDs))

	for _, bvID := range bvIDs {
		var result *InteractionResult
		var err error

		switch actionType {
		case "like":
			result, err = s.LikeVideo(ctx, bvID)
		case "coin":
			result, err = s.CoinVideo(ctx, bvID, coinCount)
		case "triple":
			result, err = s.TripleAction(ctx, bvID)
		default:
			result = &InteractionResult{Success: false, Message: "unknown action type"}
		}

		if err != nil {
			results = append(results, InteractionResult{Success: false, Message: err.Error()})
		} else {
			results = append(results, *result)
		}

		// 延迟避免请求过快
		time.Sleep(time.Second * 2)
	}

	return results, nil
}

// InteractFansVideos 互动粉丝视频
func (s *InteractionService) InteractFansVideos(ctx context.Context, actionType string, limit int) (int, error) {
	// 获取粉丝视频
	client, err := s.biliClient()
	if err != nil {
		return 0, err
	}
	fansVideos, err := client.GetFansVideos(ctx, 1, 20)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, fv := range fansVideos {
		for _, v := range fv.Videos {
			if count >= limit {
				return count, nil
			}

			var result *InteractionResult
			switch actionType {
			case "like":
				result, _ = s.LikeVideo(ctx, v.BVID)
			case "coin":
				result, _ = s.CoinVideo(ctx, v.BVID, 1)
			}

			if result != nil && result.Success {
				count++
			}

			time.Sleep(time.Second * 3)
		}
	}

	return count, nil
}

// GetStats 获取互动统计
func (s *InteractionService) GetStats(ctx context.Context, days int) (map[string]int64, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	return s.repo.GetStats(ctx, startTime, endTime)
}

// InteractionListResult 互动记录列表结果
type InteractionListResult struct {
	Items    []model.Interaction `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// List 获取互动记录列表
func (s *InteractionService) List(ctx context.Context, actionType string, page, pageSize int) (*InteractionListResult, error) {
	items, total, err := s.repo.List(ctx, actionType, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &InteractionListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListFansVideos 获取粉丝及投稿列表（用于界面选择）
func (s *InteractionService) ListFansVideos(ctx context.Context, fanLimit, videoPerFan int) ([]FanVideoSelection, error) {
	if fanLimit <= 0 {
		fanLimit = 20
	}
	if videoPerFan <= 0 {
		videoPerFan = 5
	}

	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}

	fansVideos, err := client.GetFansVideos(ctx, 1, fanLimit)
	if err != nil {
		return nil, err
	}

	out := make([]FanVideoSelection, 0, len(fansVideos))
	for _, fan := range fansVideos {
		videos := fan.Videos
		if len(videos) > videoPerFan {
			videos = videos[:videoPerFan]
		}
		out = append(out, FanVideoSelection{
			FanID:    fan.UserID,
			FanName:  fan.UserName,
			Videos:   videos,
			VideoCnt: len(videos),
		})
	}

	return out, nil
}

func (s *InteractionService) ListFans(ctx context.Context, page, pageSize int) (*FanListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	items, err := client.ListFans(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &FanListResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		HasMore:  len(items) >= pageSize,
	}, nil
}

func (s *InteractionService) ListFanVideos(ctx context.Context, fanID int64, page, pageSize int) (*FanVideosResult, error) {
	if fanID <= 0 {
		return nil, fmt.Errorf("invalid fan id")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}
	items, err := client.ListUserVideos(ctx, fanID, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &FanVideosResult{
		FanID:    fanID,
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		HasMore:  len(items) >= pageSize,
	}, nil
}

// SyncVideoEngagement 同步真实点赞/投币/收藏状态及UP硬币余额
func (s *InteractionService) SyncVideoEngagement(ctx context.Context, bvID string) (*VideoEngagementSnapshot, error) {
	client, err := s.biliClient()
	if err != nil {
		return nil, err
	}

	video, err := client.GetVideoInfo(ctx, bvID)
	if err != nil {
		return nil, err
	}
	relation, err := client.GetVideoRelationStatus(ctx, bvID)
	if err != nil {
		return nil, err
	}
	coin, err := client.GetCoinBalance(ctx)
	if err != nil {
		return nil, err
	}

	return &VideoEngagementSnapshot{
		Video:    video,
		Relation: relation,
		UPCoin:   coin,
		SyncedAt: time.Now(),
	}, nil
}

func (s *InteractionService) AutoInteractRecentFanVideos(ctx context.Context, rules InteractionRuleSettings, maxActions int) (*AutoInteractSummary, error) {
	if maxActions <= 0 {
		maxActions = 20
	}
	if rules.CoinPlayThreshold <= 0 {
		rules.CoinPlayThreshold = 10000
	}
	if rules.FanPageSize <= 0 {
		rules.FanPageSize = 20
	}
	if rules.VideoPageSize <= 0 {
		rules.VideoPageSize = 5
	}
	if rules.RequestIntervalSeconds <= 0 {
		rules.RequestIntervalSeconds = 3
	}

	summary := &AutoInteractSummary{}
	if !rules.EnableLike && !rules.EnableCoin && !rules.EnableFavorite {
		return summary, nil
	}

	cutoff := time.Now().AddDate(0, 0, -7).Unix()
	interval := time.Duration(rules.RequestIntervalSeconds) * time.Second

	fansPage, err := s.ListFans(ctx, 1, rules.FanPageSize)
	if err != nil {
		return nil, err
	}

	for _, fan := range fansPage.Items {
		if summary.TotalCount >= maxActions {
			break
		}
		videosPage, err := s.ListFanVideos(ctx, fan.UserID, 1, rules.VideoPageSize)
		if err != nil {
			continue
		}
		for _, video := range videosPage.Items {
			if summary.TotalCount >= maxActions {
				break
			}
			if video.PubDate > 0 && video.PubDate < cutoff {
				continue
			}

			if rules.EnableLike && summary.TotalCount < maxActions {
				res, _ := s.LikeVideo(ctx, video.BVID)
				if res != nil && res.Success {
					summary.LikedCount++
					summary.TotalCount++
					time.Sleep(interval)
				}
			}

			if rules.EnableCoin && summary.TotalCount < maxActions && int64(video.View) >= rules.CoinPlayThreshold {
				res, _ := s.CoinVideo(ctx, video.BVID, 1)
				if res != nil && res.Success {
					summary.CoinedCount++
					summary.TotalCount++
					time.Sleep(interval)
				}
			}

			if rules.EnableFavorite && summary.TotalCount < maxActions && rules.FavoriteMediaID > 0 {
				res, _ := s.FavoriteVideo(ctx, video.BVID, rules.FavoriteMediaID)
				if res != nil && res.Success {
					summary.FavoritedCount++
					summary.TotalCount++
					time.Sleep(interval)
				}
			}
		}
	}

	return summary, nil
}
