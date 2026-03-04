package bilibili

import (
	"context"
	"fmt"
)

type VideoInfo struct {
	BVID     string
	AVID     int64
	Title    string
	Desc     string
	Owner    string
	OwnerID  int64
	Duration int
	View     int
	Danmaku  int
	Reply    int
	Favorite int
	Coin     int
	Share    int
	Like     int
	PubDate  int64
	Tags     []string
	Pic      string
}

type LikeResult struct {
	Success bool
	Message string
}

type CoinResult struct {
	Success bool
	Message string
	Coins   int
}

func (c *Client) LikeVideo(ctx context.Context, bvID string) (*LikeResult, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return nil, err
	}

	_, err = c.inner.LikeVideo(info.AID, 1)
	if err != nil {
		return &LikeResult{Success: false, Message: err.Error()}, nil
	}
	return &LikeResult{Success: true, Message: "点赞成功"}, nil
}

func (c *Client) UnlikeVideo(ctx context.Context, bvID string) (*LikeResult, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return nil, err
	}

	_, err = c.inner.LikeVideo(info.AID, 2)
	if err != nil {
		return &LikeResult{Success: false, Message: err.Error()}, nil
	}
	return &LikeResult{Success: true, Message: "取消点赞成功"}, nil
}

func (c *Client) CoinVideo(ctx context.Context, bvID string, multiply int) (*CoinResult, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return nil, err
	}

	if multiply < 1 || multiply > 2 {
		multiply = 1
	}

	_, err = c.inner.CoinVideo(info.AID, int32(multiply))
	if err != nil {
		return &CoinResult{Success: false, Message: err.Error()}, nil
	}
	return &CoinResult{Success: true, Message: "投币成功", Coins: multiply}, nil
}

func (c *Client) FavoriteVideo(ctx context.Context, bvID string, mediaID int64) error {
	if err := c.ensureAvailable(); err != nil {
		return err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return err
	}

	_, err = c.inner.FavVideo(int(info.AID), int(mediaID))
	if err != nil {
		return fmt.Errorf("favorite video failed: %w", err)
	}
	return nil
}

func (c *Client) GetVideoInfo(ctx context.Context, bvID string) (*VideoInfo, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return nil, fmt.Errorf("get video info failed: %w", err)
	}

	result := &VideoInfo{
		BVID:    info.BVID,
		AVID:    info.AID,
		Title:   info.Title,
		Desc:    info.Desc,
		Owner:   info.Owner.Name,
		OwnerID: info.Owner.Mid,
		PubDate: info.PubDate,
		Pic:     info.Pic,
	}
	result.View = int(info.Stat.View)
	result.Danmaku = int(info.Stat.Danmaku)
	result.Reply = int(info.Stat.Reply)
	result.Favorite = int(info.Stat.Favorite)
	result.Coin = int(info.Stat.Coin)
	result.Share = int(info.Stat.Share)
	result.Like = int(info.Stat.Like)
	if len(info.Pages) > 0 {
		result.Duration = info.Pages[0].Duration
	}

	return result, nil
}

func (c *Client) IsLiked(ctx context.Context, bvID string) (bool, error) {
	if err := c.ensureAvailable(); err != nil {
		return false, err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return false, err
	}

	status, err := c.inner.GetVideoRelation(info.AID)
	if err != nil {
		return false, fmt.Errorf("get video relation failed: %w", err)
	}
	return status.Like, nil
}

func (c *Client) IsCoined(ctx context.Context, bvID string) (bool, error) {
	if err := c.ensureAvailable(); err != nil {
		return false, err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return false, err
	}

	status, err := c.inner.GetVideoRelation(info.AID)
	if err != nil {
		return false, fmt.Errorf("get video relation failed: %w", err)
	}
	return status.Coin > 0, nil
}

func (c *Client) TripleAction(ctx context.Context, bvID string) error {
	if err := c.ensureAvailable(); err != nil {
		return err
	}
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return err
	}

	_, err = c.inner.TripleAction(info.AID)
	if err != nil {
		return fmt.Errorf("triple action failed: %w", err)
	}
	return nil
}
