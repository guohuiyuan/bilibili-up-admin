package bilibili

import "context"

type Message struct {
	ID         int64
	Content    string
	SenderID   int64
	SenderName string
	SenderFace string
	Time       int64
	IsRead     bool
}

type MessageSession struct {
	UserID   int64
	UserName string
	UserFace string
	LastMsg  string
	LastTime int64
	Unread   int
	Messages []Message
}

type MessageList struct {
	Sessions []MessageSession
	Total    int
	HasMore  bool
}

func (c *Client) GetMessages(ctx context.Context, page, pageSize int) (*MessageList, error) {
	return nil, ErrNotImplemented
}

func (c *Client) GetChatHistory(ctx context.Context, userID int64, page, pageSize int) (*MessageSession, error) {
	return nil, ErrNotImplemented
}

func (c *Client) SendMessage(ctx context.Context, userID int64, content string) error {
	return ErrNotImplemented
}

func (c *Client) MarkMessageRead(ctx context.Context, userID int64) error {
	return ErrNotImplemented
}

func (c *Client) GetUnreadMessageCount(ctx context.Context) (int, error) {
	return 0, ErrNotImplemented
}
