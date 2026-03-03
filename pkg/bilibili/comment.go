package bilibili

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("current biligo module does not implement this bilibili API yet")

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
	return nil, ErrNotImplemented
}

func (c *Client) ReplyComment(ctx context.Context, videoAID, commentID int64, content string) error {
	return ErrNotImplemented
}

func (c *Client) SendVideoComment(ctx context.Context, bvID, content string) error {
	return ErrNotImplemented
}

func (c *Client) DeleteComment(ctx context.Context, commentID int64) error {
	return ErrNotImplemented
}

func (c *Client) LikeComment(ctx context.Context, commentID int64, like bool) error {
	return ErrNotImplemented
}
