package bilibili

import (
	"context"
	"fmt"
)

type TagRanking struct {
	TagName    string
	TagID      int64
	HotValue   int64
	VideoCount int
	ViewCount  int64
	Trending   bool
	TopVideos  []VideoInfo
}

type TrendingTag struct {
	Name     string
	HotValue int64
	Rank     int
	Category string
}

type VideoRanking struct {
	Videos  []VideoInfo
	Rank    int
	Tid     int
	Keyword string
}

type RankingPeriod string

const (
	RankingDaily   RankingPeriod = "day"
	RankingWeekly  RankingPeriod = "week"
	RankingMonthly RankingPeriod = "month"
)

func (c *Client) GetTrendingTags(ctx context.Context, limit int) ([]TrendingTag, error) {
	tags, err := c.inner.GetHotTags(0)
	if err != nil {
		return nil, fmt.Errorf("get trending tags failed: %w", err)
	}

	result := make([]TrendingTag, 0, len(tags))
	for i, t := range tags {
		if limit > 0 && i >= limit {
			break
		}
		result = append(result, TrendingTag{
			Name:     t.Name,
			HotValue: t.Hot,
			Rank:     i + 1,
		})
	}
	return result, nil
}

func (c *Client) GetTagRanking(ctx context.Context, tagName string, page, pageSize int) (*TagRanking, error) {
	tagInfo, err := c.inner.GetTagInfo(tagName)
	if err != nil {
		return nil, fmt.Errorf("get tag info failed: %w", err)
	}

	result := &TagRanking{
		TagName:    tagName,
		TagID:      tagInfo.TagID,
		HotValue:   tagInfo.Hot,
		VideoCount: int(tagInfo.Count),
		TopVideos:  make([]VideoInfo, 0),
	}

	videos, err := c.inner.GetTagVideos(tagName, int32(page))
	if err == nil {
		for _, v := range videos {
			result.TopVideos = append(result.TopVideos, VideoInfo{
				BVID:     v.BVID,
				AVID:     v.AID,
				Title:    v.Title,
				Owner:    v.Owner.Name,
				OwnerID:  v.Owner.Mid,
				Duration: 0,
				View:     int(v.Stat.View),
				Like:     int(v.Stat.Like),
				Coin:     int(v.Stat.Coin),
				Pic:      v.Pic,
			})
		}
	}

	return result, nil
}

func (c *Client) GetVideoRanking(ctx context.Context, tid int, period RankingPeriod) (*VideoRanking, error) {
	rank, err := c.inner.GetRanking(int32(tid))
	if err != nil {
		return nil, fmt.Errorf("get video ranking failed: %w", err)
	}

	result := &VideoRanking{
		Videos: make([]VideoInfo, 0, len(rank)),
		Tid:    tid,
	}
	for _, v := range rank {
		result.Videos = append(result.Videos, VideoInfo{
			BVID:     v.BVID,
			AVID:     v.AID,
			Title:    v.Title,
			Owner:    v.Owner.Name,
			OwnerID:  v.Owner.Mid,
			View:     int(v.Stat.View),
			Danmaku:  int(v.Stat.Danmaku),
			Reply:    int(v.Stat.Reply),
			Favorite: int(v.Stat.Favorite),
			Coin:     int(v.Stat.Coin),
			Share:    int(v.Stat.Share),
			Like:     int(v.Stat.Like),
			Pic:      v.Pic,
		})
	}
	return result, nil
}

func (c *Client) SearchTag(ctx context.Context, keyword string, page, pageSize int) ([]TagRanking, error) {
	results, err := c.inner.SearchType(keyword, "topic", int32(page))
	if err != nil {
		return nil, fmt.Errorf("search tag failed: %w", err)
	}

	tags := make([]TagRanking, 0, len(results))
	for _, r := range results {
		tags = append(tags, TagRanking{
			TagName:    r.Title,
			HotValue:   r.Play,
			VideoCount: int(r.VideoCount),
		})
	}
	return tags, nil
}

func (c *Client) GetCategoryRanking(ctx context.Context, categoryName string, limit int) (*VideoRanking, error) {
	categoryTIDs := map[string]int{
		"游戏": 4,
		"生活": 160,
		"娱乐": 5,
		"音乐": 3,
		"舞蹈": 129,
		"动画": 1,
		"科技": 188,
		"数码": 95,
		"汽车": 223,
		"时尚": 155,
		"美食": 211,
		"影视": 181,
		"知识": 36,
	}

	tid := 0
	if v, ok := categoryTIDs[categoryName]; ok {
		tid = v
	}

	rank, err := c.GetVideoRanking(ctx, tid, RankingDaily)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(rank.Videos) > limit {
		rank.Videos = rank.Videos[:limit]
	}
	return rank, nil
}

type FansVideo struct {
	UserID     int64
	UserName   string
	VideoCount int
	Videos     []VideoInfo
}

func (c *Client) GetFansVideos(ctx context.Context, page, pageSize int) ([]FansVideo, error) {
	fans, err := c.inner.GetFans(int32(page), int32(pageSize))
	if err != nil {
		return nil, fmt.Errorf("get fans failed: %w", err)
	}

	result := make([]FansVideo, 0, len(fans))
	for _, f := range fans {
		videos, err := c.inner.GetUserVideos(f.Mid, 1, 5)
		if err != nil {
			continue
		}

		item := FansVideo{
			UserID:     f.Mid,
			UserName:   f.Uname,
			VideoCount: len(videos),
			Videos:     make([]VideoInfo, 0, len(videos)),
		}
		for _, v := range videos {
			item.Videos = append(item.Videos, VideoInfo{
				BVID:     v.BVID,
				AVID:     v.AID,
				Title:    v.Title,
				Duration: 0,
				View:     int(v.Stat.View),
				Like:     int(v.Stat.Like),
				Coin:     int(v.Stat.Coin),
				Pic:      v.Pic,
			})
		}
		result = append(result, item)
	}
	return result, nil
}
