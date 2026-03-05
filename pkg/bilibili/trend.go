package bilibili

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const tagInfoMinInterval = 1200 * time.Millisecond

type TagRanking struct {
	TagName    string      `json:"tag_name"`
	TagID      int64       `json:"tag_id"`
	HotValue   int64       `json:"hot_value"`
	VideoCount int         `json:"video_count"`
	ViewCount  int64       `json:"view_count"`
	Trending   bool        `json:"trending"`
	TopVideos  []VideoInfo `json:"top_videos"`
}

type TrendingTag struct {
	TagID       int64  `json:"tag_id"`
	Name        string `json:"name"`
	HotValue    int64  `json:"hot_value"`
	Rank        int    `json:"rank"`
	Category    string `json:"category"`
	UseCount    int64  `json:"use_count"`
	FollowCount int64  `json:"follow_count"`
}

type VideoRanking struct {
	Videos  []VideoInfo `json:"videos"`
	Rank    int         `json:"rank"`
	Tid     int         `json:"tid"`
	Keyword string      `json:"keyword"`
}

type RankingPeriod string

const (
	RankingDaily   RankingPeriod = "day"
	RankingWeekly  RankingPeriod = "week"
	RankingMonthly RankingPeriod = "month"
)

type TagInfo struct {
	TagID       int64 `json:"tag_id"`
	HotValue    int64 `json:"hot_value"`
	UseCount    int64 `json:"use_count"`
	FollowCount int64 `json:"follow_count"`
}

func (c *Client) GetTagInfo(ctx context.Context, tagName string) (*TagInfo, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	info, err := c.inner.GetTagInfo(tagName)
	if err != nil {
		return nil, fmt.Errorf("get tag info failed: %w", err)
	}
	if info == nil {
		return nil, fmt.Errorf("get tag info failed: empty result")
	}
	return &TagInfo{
		TagID:       info.TagID,
		HotValue:    info.Hot,
		UseCount:    info.Count.Use,
		FollowCount: info.Count.Atten,
	}, nil
}

func (c *Client) GetTrendingTags(ctx context.Context, limit int) ([]TrendingTag, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	zones := trendingZones()
	return c.getTrendingTagsFromZones(ctx, zones, limit)
}

func (c *Client) GetTrendingTagsByCategory(ctx context.Context, category string, limit int) ([]TrendingTag, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	if category == "" {
		return c.GetTrendingTags(ctx, limit)
	}

	zone, ok := resolveTrendZone(category)
	if !ok {
		return nil, fmt.Errorf("unsupported category: %s", category)
	}

	return c.getTrendingTagsFromZones(ctx, []trendZone{zone}, limit)
}

type trendZone struct {
	rid      int32
	category string
}

func trendingZones() []trendZone {
	return []trendZone{
		{rid: 3, category: "音乐"},
		{rid: 4, category: "游戏"},
		{rid: 36, category: "知识"},
		{rid: 160, category: "生活"},
		{rid: 188, category: "科技"},
		{rid: 1, category: "动画"},
	}
}

func resolveTrendZone(category string) (trendZone, bool) {
	for _, zone := range trendingZones() {
		if zone.category == category {
			return zone, true
		}
	}

	rid, err := strconv.Atoi(category)
	if err == nil {
		for _, zone := range trendingZones() {
			if zone.rid == int32(rid) {
				return zone, true
			}
		}
		return trendZone{rid: int32(rid), category: fmt.Sprintf("分区%d", rid)}, true
	}

	return trendZone{}, false
}

func (c *Client) getTrendingTagsFromZones(ctx context.Context, zones []trendZone, limit int) ([]TrendingTag, error) {
	if limit <= 0 {
		limit = 50
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("get trending tags failed: empty zones")
	}

	result := make([]TrendingTag, 0, limit)
	seenTagIDs := make(map[int64]struct{})
	seenNames := make(map[string]struct{})
	var lastErr error

	tagsByZone := make([][]struct {
		TagID int64
		Name  string
		Hot   int64
	}, len(zones))

	for i, z := range zones {
		tags, err := c.inner.GetHotTags(z.rid)
		if err != nil {
			lastErr = err
			continue
		}
		zoneTags := make([]struct {
			TagID int64
			Name  string
			Hot   int64
		}, 0, len(tags))
		for _, t := range tags {
			zoneTags = append(zoneTags, struct {
				TagID int64
				Name  string
				Hot   int64
			}{TagID: t.TagID, Name: t.Name, Hot: t.Hot})
		}
		tagsByZone[i] = zoneTags
	}

	baseQuota := limit / len(zones)
	extraQuota := limit % len(zones)

	zoneCursors := make([]int, len(zones))
	for i, z := range zones {
		quota := baseQuota
		if i < extraQuota {
			quota++
		}
		if quota <= 0 {
			continue
		}

		added := 0
		for zoneCursors[i] < len(tagsByZone[i]) {
			t := tagsByZone[i][zoneCursors[i]]
			zoneCursors[i]++

			if t.Name == "" {
				continue
			}
			if t.TagID != 0 {
				if _, ok := seenTagIDs[t.TagID]; ok {
					continue
				}
			}
			if _, ok := seenNames[t.Name]; ok {
				continue
			}

			if t.TagID != 0 {
				seenTagIDs[t.TagID] = struct{}{}
			}
			seenNames[t.Name] = struct{}{}

			result = append(result, TrendingTag{
				TagID:    t.TagID,
				Name:     t.Name,
				HotValue: t.Hot,
				Rank:     len(result) + 1,
				Category: z.category,
			})
			added++

			if len(result) >= limit || added >= quota {
				break
			}
		}
		if len(result) >= limit {
			break
		}
	}

	for len(result) < limit {
		addedInRound := 0
		for i, z := range zones {
			for zoneCursors[i] < len(tagsByZone[i]) {
				t := tagsByZone[i][zoneCursors[i]]
				zoneCursors[i]++

				if t.Name == "" {
					continue
				}
				if t.TagID != 0 {
					if _, ok := seenTagIDs[t.TagID]; ok {
						continue
					}
				}
				if _, ok := seenNames[t.Name]; ok {
					continue
				}

				if t.TagID != 0 {
					seenTagIDs[t.TagID] = struct{}{}
				}
				seenNames[t.Name] = struct{}{}

				result = append(result, TrendingTag{
					TagID:    t.TagID,
					Name:     t.Name,
					HotValue: t.Hot,
					Rank:     len(result) + 1,
					Category: z.category,
				})
				addedInRound++
				break
			}

			if len(result) >= limit {
				break
			}
		}
		if addedInRound == 0 {
			break
		}
	}

	if len(result) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("get trending tags failed: %w", lastErr)
		}
		return nil, fmt.Errorf("get trending tags failed: empty result")
	}

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return c.enrichTrendingTagsWithInfo(ctx, result, 5), nil
}

func (c *Client) enrichTrendingTagsWithInfo(ctx context.Context, tags []TrendingTag, maxConcurrency int) []TrendingTag {
	if len(tags) == 0 {
		return tags
	}

	enriched := make([]TrendingTag, len(tags))
	copy(enriched, tags)
	for i := range enriched {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return enriched
			default:
			}
		}

		if i > 0 {
			if ctx == nil {
				time.Sleep(tagInfoMinInterval)
			} else {
				timer := time.NewTimer(tagInfoMinInterval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return enriched
				case <-timer.C:
				}
			}
		}

		info, err := c.inner.GetTagInfo(enriched[i].Name)
		if err != nil || info == nil {
			continue
		}

		enriched[i].TagID = info.TagID
		enriched[i].HotValue = info.Hot
		enriched[i].UseCount = info.Count.Use
		enriched[i].FollowCount = info.Count.Atten
	}
	return enriched
}

func (c *Client) GetTagRanking(ctx context.Context, tagName string, page, pageSize int) (*TagRanking, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	tagInfo, err := c.inner.GetTagInfo(tagName)
	if err != nil {
		return nil, fmt.Errorf("get tag info failed: %w", err)
	}

	result := &TagRanking{
		TagName:    tagName,
		TagID:      tagInfo.TagID,
		HotValue:   tagInfo.Hot,
		VideoCount: int(tagInfo.Count.View),
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
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
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
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
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
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
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
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
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
				AVID:     0,
				Title:    v.Title,
				Duration: 0,
				View:     int(v.Play),
				Like:     0,
				Coin:     0,
				Pic:      v.Pic,
			})
		}
		result = append(result, item)
	}
	return result, nil
}
