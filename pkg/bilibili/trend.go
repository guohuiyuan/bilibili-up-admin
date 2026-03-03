package bilibili

import "context"

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
	return []TrendingTag{}, nil
}

func (c *Client) GetTagRanking(ctx context.Context, tagName string, page, pageSize int) (*TagRanking, error) {
	return nil, ErrNotImplemented
}

func (c *Client) GetVideoRanking(ctx context.Context, tid int, period RankingPeriod) (*VideoRanking, error) {
	popular, err := c.inner.Video().Popular(ctx, 1, 50)
	if err != nil {
		return nil, err
	}

	out := &VideoRanking{
		Videos: make([]VideoInfo, 0, len(popular.List)),
		Tid:    tid,
	}
	for _, v := range popular.List {
		out.Videos = append(out.Videos, VideoInfo{
			BVID:     v.BVID,
			AVID:     v.AID,
			Title:    v.Title,
			Desc:     v.Desc,
			Owner:    v.Owner.Name,
			OwnerID:  v.Owner.Mid,
			PubDate:  v.PubDate,
			Pic:      v.Pic,
			View:     int(v.Stat.View),
			Danmaku:  int(v.Stat.Danmaku),
			Reply:    int(v.Stat.Reply),
			Favorite: int(v.Stat.Favorite),
			Coin:     int(v.Stat.Coin),
			Share:    int(v.Stat.Share),
			Like:     int(v.Stat.Like),
		})
	}
	return out, nil
}

func (c *Client) SearchTag(ctx context.Context, keyword string, page, pageSize int) ([]TagRanking, error) {
	results, err := c.inner.Search().Suggest(ctx, keyword)
	if err != nil {
		return nil, err
	}

	out := make([]TagRanking, 0, len(results))
	for _, item := range results {
		out = append(out, TagRanking{TagName: item.Value})
	}
	return out, nil
}

func (c *Client) GetCategoryRanking(ctx context.Context, categoryName string, limit int) (*VideoRanking, error) {
	ranking, err := c.GetVideoRanking(ctx, 0, RankingDaily)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(ranking.Videos) > limit {
		ranking.Videos = ranking.Videos[:limit]
	}
	ranking.Keyword = categoryName
	return ranking, nil
}

type FansVideo struct {
	UserID     int64
	UserName   string
	VideoCount int
	Videos     []VideoInfo
}

func (c *Client) GetFansVideos(ctx context.Context, page, pageSize int) ([]FansVideo, error) {
	return nil, ErrNotImplemented
}
