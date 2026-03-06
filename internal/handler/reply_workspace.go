package handler

import (
	"net/http"
	"strconv"

	"bilibili-up-admin/internal/service"

	"github.com/gin-gonic/gin"
)

type ReplyWorkspaceHandler struct {
	svc *service.ReplyWorkspaceService
}

func NewReplyWorkspaceHandler(svc *service.ReplyWorkspaceService) *ReplyWorkspaceHandler {
	return &ReplyWorkspaceHandler{svc: svc}
}

func (h *ReplyWorkspaceHandler) Workspace(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Query("target_id"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target id"})
		return
	}
	conversationID, _ := strconv.ParseInt(c.Query("conversation_id"), 10, 64)
	data, err := h.svc.GetWorkspace(c.Request.Context(), c.Query("channel"), targetID, conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *ReplyWorkspaceHandler) GenerateDraft(c *gin.Context) {
	var req service.GenerateReplyDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	draft, err := h.svc.GenerateDraft(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft": draft})
}

func (h *ReplyWorkspaceHandler) SaveDraft(c *gin.Context) {
	var req service.SaveReplyDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	draft, err := h.svc.SaveDraft(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft": draft})
}

func (h *ReplyWorkspaceHandler) SendDraft(c *gin.Context) {
	var req service.SendReplyDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SendDraft(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reply sent"})
}

func (h *ReplyWorkspaceHandler) ListTemplates(c *gin.Context) {
	items, err := h.svc.ListTemplates(c.Request.Context(), c.Query("channel"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": items})
}

func (h *ReplyWorkspaceHandler) CreateTemplate(c *gin.Context) {
	var req struct {
		Channel string `json:"channel" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Scene   string `json:"scene"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreateTemplate(c.Request.Context(), req.Channel, req.Title, req.Content, req.Scene); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "template created"})
}

func (h *ReplyWorkspaceHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}
	if err := h.svc.DeleteTemplate(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}
