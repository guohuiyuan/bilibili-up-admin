package bilibili

import (
	"context"
	"fmt"
	"strconv"
)

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
	zones := trendTagZones()
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

func trendTagZones() []trendZone {
	return []trendZone{
		{rid: 13, category: "番剧"},
		{rid: 167, category: "国创"},
		{rid: 177, category: "纪录片"},
		{rid: 23, category: "电影"},
		{rid: 11, category: "电视剧"},
		{rid: 71, category: "综艺"},
		{rid: 1001, category: "影视"},
		{rid: 1002, category: "娱乐"},
		{rid: 1003, category: "音乐"},
		{rid: 1004, category: "舞蹈"},
		{rid: 1005, category: "动画"},
		{rid: 1006, category: "绘画"},
		{rid: 1007, category: "鬼畜"},
		{rid: 1008, category: "游戏"},
		{rid: 1009, category: "资讯"},
		{rid: 1010, category: "知识"},
		{rid: 1011, category: "人工智能"},
		{rid: 1012, category: "科技数码"},
		{rid: 1013, category: "汽车"},
		{rid: 1014, category: "时尚美妆"},
		{rid: 1015, category: "家装房产"},
		{rid: 1016, category: "户外潮流"},
		{rid: 1017, category: "健身"},
		{rid: 1018, category: "体育运动"},
		{rid: 1019, category: "手工"},
		{rid: 1020, category: "美食"},
		{rid: 1021, category: "小剧场"},
		{rid: 1022, category: "旅游出行"},
		{rid: 1023, category: "三农"},
		{rid: 1024, category: "动物"},
		{rid: 1025, category: "亲子"},
		{rid: 1026, category: "健康"},
		{rid: 1027, category: "情感"},
		{rid: 1029, category: "vlog"},
		{rid: 1030, category: "生活兴趣"},
		{rid: 1031, category: "生活经验"},
	}
}

func resolveTrendZone(category string) (trendZone, bool) {
	rid, err := strconv.Atoi(category)
	if err != nil {
		return trendZone{}, false
	}
	for _, zone := range trendTagZones() {
		if zone.rid == int32(rid) {
			return zone, true
		}
	}

	return trendZone{}, false
}

func (c *Client) getTrendingTagsFromZones(ctx context.Context, zones []trendZone, limit int) ([]TrendingTag, error) {
	if limit <= 0 {
		if len(zones) <= 1 {
			limit = 30
		} else {
			limit = len(zones) * 3
		}
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

	return result, nil
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
	tid64, err := strconv.ParseInt(categoryName, 10, 32)
	if categoryName == "" {
		tid64 = 0
	} else if err != nil {
		return nil, fmt.Errorf("unsupported category: %s", categoryName)
	}
	tid := int32(tid64)
	pgcSeasonByTID := map[int32]int32{13: 1, 167: 4, 177: 3, 23: 2, 11: 5, 71: 7}
	if seasonType, ok := pgcSeasonByTID[tid]; ok {
		list, err := c.inner.GetPGCRanking(seasonType, 3)
		if err != nil {
			return nil, fmt.Errorf("get video ranking failed: %w", err)
		}

		out := &VideoRanking{Videos: make([]VideoInfo, 0, len(list)), Tid: int(tid)}
		for _, v := range list {
			out.Videos = append(out.Videos, VideoInfo{
				Title:    v.Title,
				View:     int(v.Stat.View),
				Danmaku:  int(v.Stat.Danmaku),
				Favorite: int(v.Stat.Follow),
				Pic:      v.Cover,
			})
		}
		if limit > 0 && len(out.Videos) > limit {
			out.Videos = out.Videos[:limit]
		}
		return out, nil
	}

	rank, err := c.inner.GetRankingWithType(tid, "all")
	if err != nil {
		return nil, fmt.Errorf("get video ranking failed: %w", err)
	}

	out := &VideoRanking{Videos: make([]VideoInfo, 0, len(rank)), Tid: int(tid)}
	for _, v := range rank {
		out.Videos = append(out.Videos, VideoInfo{
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

	if limit > 0 && len(out.Videos) > limit {
		out.Videos = out.Videos[:limit]
	}
	return out, nil
}

type FansVideo struct {
	UserID     int64
	UserName   string
	VideoCount int
	Videos     []VideoInfo
}

type FanProfile struct {
	UserID     int64  `json:"user_id"`
	UserName   string `json:"user_name"`
	UserFace   string `json:"user_face"`
	FollowTime int64  `json:"follow_time"`
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

func (c *Client) ListFans(ctx context.Context, page, pageSize int) ([]FanProfile, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	fans, err := c.inner.GetFans(int32(page), int32(pageSize))
	if err != nil {
		return nil, fmt.Errorf("get fans failed: %w", err)
	}
	out := make([]FanProfile, 0, len(fans))
	for _, f := range fans {
		out = append(out, FanProfile{UserID: f.Mid, UserName: f.Uname, UserFace: f.Face, FollowTime: f.MTime})
	}
	return out, nil
}

func (c *Client) ListUserVideos(ctx context.Context, mid int64, page, pageSize int) ([]VideoInfo, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	videos, err := c.inner.GetUserVideos(mid, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get user videos failed: %w", err)
	}
	out := make([]VideoInfo, 0, len(videos))
	for _, v := range videos {
		pubDate := v.PubDate
		if pubDate == 0 {
			pubDate = v.Created
		}
		out = append(out, VideoInfo{
			BVID:    v.BVID,
			Title:   v.Title,
			View:    int(v.Play),
			Reply:   int(v.Comment),
			PubDate: pubDate,
			Pic:     v.Pic,
		})
	}
	return out, nil
}
