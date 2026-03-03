package handler

import (
	"net/http"

	"bilibili-up-admin/internal/service"
	"bilibili-up-admin/pkg/llm"

	"github.com/gin-gonic/gin"
)

// LLMHandler 大模型处理器
type LLMHandler struct {
	svc *service.LLMService
}

// NewLLMHandler 创建大模型处理器
func NewLLMHandler(svc *service.LLMService) *LLMHandler {
	return &LLMHandler{svc: svc}
}

// ChatRequest 对话请求
type ChatRequest struct {
	Provider     string        `json:"provider"`
	SystemPrompt string        `json:"system_prompt"`
	Messages     []llm.Message `json:"messages" binding:"required"`
}

// Chat 对话
func (h *LLMHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var resp *llm.Response
	var err error

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

// Providers 获取提供者列表
func (h *LLMHandler) Providers(c *gin.Context) {
	providers := h.svc.GetProviders()
	c.JSON(http.StatusOK, gin.H{
		"providers":        providers,
		"default_provider": h.svc.GetDefaultProvider(),
	})
}

// SetDefault 设置默认提供者
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

// Test 测试提供者
func (h *LLMHandler) Test(c *gin.Context) {
	provider := c.Param("provider")

	success, message := h.svc.TestProvider(c.Request.Context(), provider)
	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"message": message,
	})
}

// Stats 获取统计
func (h *LLMHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context(), 7)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
