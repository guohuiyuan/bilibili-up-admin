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
		ActionTime:   time.Now(),
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
		ActionTime:   time.Now(),
	})

	return &InteractionResult{Success: result.Success, Message: result.Message}, nil
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
		ActionTime:   time.Now(),
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
