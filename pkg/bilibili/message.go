package bilibili

import (
	"context"
	"fmt"
	"strings"
)

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

func (c *Client) resolveMessageUserProfile(ctx context.Context, userID int64) (string, string) {
	if userID <= 0 || c == nil || c.inner == nil {
		return "", ""
	}
	info, err := c.inner.User().Info(ctx, userID)
	if err != nil || info == nil {
		return "", ""
	}
	return strings.TrimSpace(info.Name), strings.TrimSpace(info.Face)
}

func (c *Client) GetMessages(ctx context.Context, page, pageSize int) (*MessageList, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	msgs, err := c.inner.GetMsgFeed(int32(page))
	if err != nil {
		return nil, fmt.Errorf("get messages failed: %w", err)
	}

	result := &MessageList{
		Sessions: make([]MessageSession, 0, len(msgs)),
		Total:    len(msgs),
		HasMore:  len(msgs) >= pageSize,
	}

	for _, m := range msgs {
		userName := strings.TrimSpace(m.Uname)
		userFace := strings.TrimSpace(m.Avatar)
		if userName == "" || userFace == "" {
			resolvedName, resolvedFace := c.resolveMessageUserProfile(ctx, m.Mid)
			if userName == "" {
				userName = resolvedName
			}
			if userFace == "" {
				userFace = resolvedFace
			}
		}
		result.Sessions = append(result.Sessions, MessageSession{
			UserID:   m.Mid,
			UserName: userName,
			UserFace: userFace,
			LastMsg:  m.LastMsg,
			Unread:   int(m.Unfollow),
		})
	}

	return result, nil
}

func (c *Client) GetChatHistory(ctx context.Context, userID int64, page, pageSize int) (*MessageSession, error) {
	if err := c.ensureAvailable(); err != nil {
		return nil, err
	}
	history, err := c.inner.GetChatHistory(userID, int32(page))
	if err != nil {
		return nil, fmt.Errorf("get chat history failed: %w", err)
	}

	result := &MessageSession{
		UserID:   userID,
		Messages: make([]Message, 0, len(history)),
	}

	for _, m := range history {
		result.Messages = append(result.Messages, Message{
			ID:         m.MsgID,
			Content:    m.Content,
			SenderID:   m.SenderUID,
			SenderName: m.SenderName,
			Time:       m.Timestamp,
			IsRead:     true,
		})
	}

	return result, nil
}

func (c *Client) SendMessage(ctx context.Context, userID int64, content string) error {
	if err := c.ensureAvailable(); err != nil {
		return err
	}
	_, err := c.inner.SendMsg(userID, content)
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}
	return nil
}

func (c *Client) MarkMessageRead(ctx context.Context, userID int64) error {
	if err := c.ensureAvailable(); err != nil {
		return err
	}
	_, err := c.inner.ReadMsg(userID)
	if err != nil {
		return fmt.Errorf("mark message read failed: %w", err)
	}
	return nil
}

func (c *Client) GetUnreadMessageCount(ctx context.Context) (int, error) {
	if err := c.ensureAvailable(); err != nil {
		return 0, err
	}
	unread, err := c.inner.GetUnreadMsg()
	if err != nil {
		return 0, fmt.Errorf("get unread count failed: %w", err)
	}
	return int(unread), nil
}
