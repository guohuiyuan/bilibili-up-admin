package handler

import (
	"net/http"
	"strconv"

	"bilibili-up-admin/internal/service"

	"github.com/gin-gonic/gin"
)

// MessageHandler 私信处理器
type MessageHandler struct {
	svc *service.MessageService
}

// NewMessageHandler 创建私信处理器
func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// List 获取私信列表
func (h *MessageHandler) List(c *gin.Context) {
	senderID, _ := strconv.ParseInt(c.Query("sender_uid"), 10, 64)
	replyStatus, _ := strconv.Atoi(c.DefaultQuery("reply_status", "-1"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.List(c.Request.Context(), senderID, replyStatus, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Sync 同步私信
func (h *MessageHandler) Sync(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.SyncMessages(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "同步成功",
		"count":   result.Inserted,
		"stats":   result,
	})
}

// AIReply AI回复
func (h *MessageHandler) AIReply(c *gin.Context) {
	messageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	content, err := h.svc.AIReply(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "回复成功",
		"content": content,
	})
}

// ManualReply 手动回复
func (h *MessageHandler) ManualReply(c *gin.Context) {
	messageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	var req struct {
		SenderID int64  `json:"sender_uid" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.ManualReply(c.Request.Context(), messageID, req.SenderID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "回复成功"})
}

// Ignore 忽略私信
func (h *MessageHandler) Ignore(c *gin.Context) {
	messageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	if err := h.svc.Ignore(c.Request.Context(), messageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已忽略"})
}

// UnreadCount 获取未读数量
func (h *MessageHandler) UnreadCount(c *gin.Context) {
	count, err := h.svc.GetUnreadCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}
