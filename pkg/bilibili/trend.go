package bilibili

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
		if _, hasAlias := resolveRankTypeAlias(category); hasAlias {
			return c.GetTrendingTags(ctx, limit)
		}
		return nil, fmt.Errorf("unsupported category: %s", category)
	}

	return c.getTrendingTagsFromZones(ctx, []trendZone{zone}, limit)
}

type trendZone struct {
	rid      int32
	category string
}

type rankTypeAlias struct {
	Key         string
	Label       string
	VideoRID    int32
	VideoType   string
	TagCategory string
}

func rankTypeAliases() []rankTypeAlias {
	return []rankTypeAlias{
		{Key: "All", Label: "全部", VideoRID: 0, VideoType: "all", TagCategory: ""},
		{Key: "Bangumi", Label: "番剧", VideoRID: 0, VideoType: "all", TagCategory: ""},
		{Key: "GuochuangAnime", Label: "国产动画", VideoRID: 0, VideoType: "all", TagCategory: "动画"},
		{Key: "Guochuang", Label: "国创相关", VideoRID: 168, VideoType: "all", TagCategory: "动画"},
		{Key: "Documentary", Label: "纪录片", VideoRID: 0, VideoType: "all", TagCategory: "影视"},
		{Key: "Douga", Label: "动画", VideoRID: 1005, VideoType: "all", TagCategory: "动画"},
		{Key: "Music", Label: "音乐", VideoRID: 1003, VideoType: "all", TagCategory: "音乐"},
		{Key: "Dance", Label: "舞蹈", VideoRID: 1004, VideoType: "all", TagCategory: "舞蹈"},
		{Key: "Game", Label: "游戏", VideoRID: 1008, VideoType: "all", TagCategory: "游戏"},
		{Key: "Knowledge", Label: "知识", VideoRID: 1010, VideoType: "all", TagCategory: "知识"},
		{Key: "Technology", Label: "科技数码", VideoRID: 1012, VideoType: "all", TagCategory: "科技"},
		{Key: "Sports", Label: "运动", VideoRID: 1018, VideoType: "all", TagCategory: "运动"},
		{Key: "Car", Label: "汽车", VideoRID: 1013, VideoType: "all", TagCategory: "汽车"},
		{Key: "Life", Label: "生活", VideoRID: 160, VideoType: "all", TagCategory: "生活"},
		{Key: "Food", Label: "美食", VideoRID: 1020, VideoType: "all", TagCategory: "美食"},
		{Key: "Animal", Label: "动物圈", VideoRID: 1024, VideoType: "all", TagCategory: "动物圈"},
		{Key: "Kichiku", Label: "鬼畜", VideoRID: 1007, VideoType: "all", TagCategory: "鬼畜"},
		{Key: "Fashion", Label: "时尚美妆", VideoRID: 1014, VideoType: "all", TagCategory: "时尚"},
		{Key: "Ent", Label: "娱乐", VideoRID: 1002, VideoType: "all", TagCategory: "娱乐"},
		{Key: "Cinephile", Label: "影视", VideoRID: 1001, VideoType: "all", TagCategory: "影视"},
		{Key: "Movie", Label: "电影", VideoRID: 0, VideoType: "all", TagCategory: ""},
		{Key: "TV", Label: "电视剧", VideoRID: 0, VideoType: "all", TagCategory: ""},
		{Key: "Variety", Label: "综艺", VideoRID: 0, VideoType: "all", TagCategory: ""},
		{Key: "Original", Label: "原创", VideoRID: 0, VideoType: "origin", TagCategory: ""},
		{Key: "Rookie", Label: "新人", VideoRID: 0, VideoType: "rookie", TagCategory: ""},
	}
}

func resolveRankTypeAlias(input string) (rankTypeAlias, bool) {
	for _, alias := range rankTypeAliases() {
		if strings.EqualFold(alias.Key, input) || alias.Label == input {
			return alias, true
		}
	}
	return rankTypeAlias{}, false
}

func trendingZones() []trendZone {
	return []trendZone{
		{rid: 1, category: "动画"},
		{rid: 3, category: "音乐"},
		{rid: 4, category: "游戏"},
		{rid: 5, category: "娱乐"},
		{rid: 36, category: "知识"},
		{rid: 119, category: "鬼畜"},
		{rid: 129, category: "舞蹈"},
		{rid: 155, category: "时尚"},
		{rid: 160, category: "生活"},
		{rid: 181, category: "影视"},
		{rid: 188, category: "科技"},
		{rid: 211, category: "美食"},
		{rid: 217, category: "动物圈"},
		{rid: 223, category: "汽车"},
		{rid: 234, category: "运动"},
	}
}

func resolveTrendZone(category string) (trendZone, bool) {
	if alias, ok := resolveRankTypeAlias(category); ok {
		if alias.TagCategory == "" {
			return trendZone{}, false
		}
		category = alias.TagCategory
	}

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
	rankType := "all"
	tid := 0
	if alias, ok := resolveRankTypeAlias(categoryName); ok {
		tid = int(alias.VideoRID)
		rankType = alias.VideoType
	} else {
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
		if v, ok := categoryTIDs[categoryName]; ok {
			tid = v
		}
	}

	rank, err := c.inner.GetRankingWithType(int32(tid), rankType)
	if err != nil {
		return nil, fmt.Errorf("get video ranking failed: %w", err)
	}

	out := &VideoRanking{Videos: make([]VideoInfo, 0, len(rank)), Tid: tid}
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
