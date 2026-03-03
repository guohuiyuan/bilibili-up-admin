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
	return nil, ErrNotImplemented
}

func (c *Client) UnlikeVideo(ctx context.Context, bvID string) (*LikeResult, error) {
	return nil, ErrNotImplemented
}

func (c *Client) CoinVideo(ctx context.Context, bvID string, multiply int) (*CoinResult, error) {
	return nil, ErrNotImplemented
}

func (c *Client) FavoriteVideo(ctx context.Context, bvID string, mediaID int64) error {
	return ErrNotImplemented
}

func (c *Client) GetVideoInfo(ctx context.Context, bvID string) (*VideoInfo, error) {
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return nil, fmt.Errorf("get video info failed: %w", err)
	}

	return &VideoInfo{
		BVID:     info.BVID,
		AVID:     info.AID,
		Title:    info.Title,
		Desc:     info.Desc,
		Owner:    info.Owner.Name,
		OwnerID:  info.Owner.Mid,
		PubDate:  info.PubDate,
		Pic:      info.Pic,
		View:     int(info.Stat.View),
		Danmaku:  int(info.Stat.Danmaku),
		Reply:    int(info.Stat.Reply),
		Favorite: int(info.Stat.Favorite),
		Coin:     int(info.Stat.Coin),
		Share:    int(info.Stat.Share),
		Like:     int(info.Stat.Like),
	}, nil
}

func (c *Client) IsLiked(ctx context.Context, bvID string) (bool, error) {
	return false, ErrNotImplemented
}

func (c *Client) IsCoined(ctx context.Context, bvID string) (bool, error) {
	return false, ErrNotImplemented
}

func (c *Client) TripleAction(ctx context.Context, bvID string) error {
	return ErrNotImplemented
}
