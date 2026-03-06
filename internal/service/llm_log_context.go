package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/pkg/llm"
)

type llmConversationMeta struct {
	Key   string
	Title string
}

func llmConversationForComment(comment *model.Comment) llmConversationMeta {
	if comment == nil {
		return llmConversationMeta{}
	}
	title := strings.TrimSpace(comment.VideoBVID)
	if title == "" {
		title = fmt.Sprintf("comment-%d", comment.CommentID)
	}
	return llmConversationMeta{
		Key:   fmt.Sprintf("comment:%s", title),
		Title: title,
	}
}

func llmConversationForMessage(message *model.Message) llmConversationMeta {
	if message == nil {
		return llmConversationMeta{}
	}
	conversationUID := message.ConversationUID
	if conversationUID == 0 {
		conversationUID = message.SenderID
	}
	title := strings.TrimSpace(message.ConversationName)
	if title == "" {
		title = strings.TrimSpace(message.SenderName)
	}
	if title == "" {
		title = fmt.Sprintf("viewer-%d", conversationUID)
	}
	return llmConversationMeta{
		Key:   fmt.Sprintf("message:%d", conversationUID),
		Title: title,
	}
}

func marshalLLMMessages(messages []llm.Message) string {
	data, err := json.Marshal(messages)
	if err != nil {
		return ""
	}
	return string(data)
}
