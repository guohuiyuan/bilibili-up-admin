package bilibili

import (
	"context"
	"fmt"
	"strconv"
)

type Comment struct {
	ID        int64
	Content   string
	Author    string
	AuthorID  int64
	VideoID   string
	VideoAID  int64
	Time      int64
	LikeCount int
	IsReply   bool
	ParentID  int64
}

type CommentList struct {
	Comments []Comment
	Total    int
	HasMore  bool
}

func (c *Client) GetVideoComments(ctx context.Context, bvID string, page, pageSize int) (*CommentList, error) {
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return nil, fmt.Errorf("get video info failed: %w", err)
	}

	comments, err := c.inner.GetVideoComment(info.AID, int32(page))
	if err != nil {
		return nil, fmt.Errorf("get video comments failed: %w", err)
	}

	result := &CommentList{
		Comments: make([]Comment, 0, len(comments.Replies)),
		Total:    int(comments.Page.Count),
		HasMore:  len(comments.Replies) >= pageSize,
	}

	for _, r := range comments.Replies {
		result.Comments = append(result.Comments, Comment{
			ID:        r.Rpid,
			Content:   r.Content.Message,
			Author:    r.Member.Uname,
			AuthorID:  r.Member.Mid,
			VideoID:   bvID,
			VideoAID:  info.AID,
			Time:      r.Ctime,
			LikeCount: r.Like,
			IsReply:   false,
		})
	}

	return result, nil
}

func (c *Client) ReplyComment(ctx context.Context, videoAID, commentID int64, content string) error {
	_, err := c.inner.ReplyComment(
		strconv.FormatInt(videoAID, 10),
		1,
		content,
		strconv.FormatInt(commentID, 10),
		"",
	)
	if err != nil {
		return fmt.Errorf("reply comment failed: %w", err)
	}
	return nil
}

func (c *Client) SendVideoComment(ctx context.Context, bvID, content string) error {
	info, err := c.inner.Video().InfoByBVID(ctx, bvID)
	if err != nil {
		return err
	}

	_, err = c.inner.SendVideoComment(info.AID, content)
	if err != nil {
		return fmt.Errorf("send video comment failed: %w", err)
	}
	return nil
}

func (c *Client) DeleteComment(ctx context.Context, videoAID, commentID int64) error {
	_, err := c.inner.DelComment(videoAID, 1, commentID)
	if err != nil {
		return fmt.Errorf("delete comment failed: %w", err)
	}
	return nil
}

func (c *Client) LikeComment(ctx context.Context, videoAID, commentID int64, like bool) error {
	action := 0
	if like {
		action = 1
	}

	_, err := c.inner.LikeComment(videoAID, 1, commentID, action)
	if err != nil {
		return fmt.Errorf("like comment failed: %w", err)
	}
	return nil
}
