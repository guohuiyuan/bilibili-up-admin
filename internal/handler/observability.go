package handler

import (
	"net/http"

	"bilibili-up-admin/internal/polling"

	"github.com/gin-gonic/gin"
)

// ObservabilityHandler 可观测性接口
type ObservabilityHandler struct {
	polling *polling.Manager
}

func NewObservabilityHandler(pollingManager *polling.Manager) *ObservabilityHandler {
	return &ObservabilityHandler{polling: pollingManager}
}

func (h *ObservabilityHandler) PollingStats(c *gin.Context) {
	if h.polling == nil {
		c.JSON(http.StatusOK, gin.H{
			"started":      false,
			"task_count":   0,
			"generated_at": nil,
			"tasks":        []any{},
		})
		return
	}
	c.JSON(http.StatusOK, h.polling.Snapshot())
}
