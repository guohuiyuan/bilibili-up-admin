package handler

import (
	"net/http"
	"strconv"

	"bilibili-up-admin/internal/service"
	"bilibili-up-admin/pkg/llm"

	"github.com/gin-gonic/gin"
)

type LLMHandler struct {
	svc *service.LLMService
}

func NewLLMHandler(svc *service.LLMService) *LLMHandler {
	return &LLMHandler{svc: svc}
}

type ChatRequest struct {
	Provider     string        `json:"provider"`
	SystemPrompt string        `json:"system_prompt"`
	Messages     []llm.Message `json:"messages" binding:"required"`
}

func (h *LLMHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		resp *llm.Response
		err  error
	)
	if req.SystemPrompt != "" {
		resp, err = h.svc.ChatWithSystem(c.Request.Context(), req.Provider, req.SystemPrompt, req.Messages)
	} else {
		resp, err = h.svc.Chat(c.Request.Context(), req.Provider, req.Messages)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *LLMHandler) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"providers":        h.svc.GetProviders(),
		"default_provider": h.svc.GetDefaultProvider(),
	})
}

func (h *LLMHandler) SetDefault(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.svc.SetDefaultProvider(req.Provider)
	c.JSON(http.StatusOK, gin.H{"message": "设置成功"})
}

func (h *LLMHandler) Test(c *gin.Context) {
	success, message := h.svc.TestProvider(c.Request.Context(), c.Param("provider"))
	c.JSON(http.StatusOK, gin.H{"success": success, "message": message})
}

func (h *LLMHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context(), 7)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *LLMHandler) Logs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	result, err := h.svc.ListLogs(
		c.Request.Context(),
		c.Query("input_type"),
		c.Query("conversation_key"),
		c.Query("log_type"),
		page,
		pageSize,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
