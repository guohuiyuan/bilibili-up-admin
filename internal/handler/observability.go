package handler

import (
	"net/http"

	"bilibili-up-admin/internal/polling"
	"bilibili-up-admin/internal/repository"

	"github.com/gin-gonic/gin"
)

// ObservabilityHandler 可观测性接口
type ObservabilityHandler struct {
	settingRepo *repository.SettingRepository
}

func NewObservabilityHandler(settingRepo *repository.SettingRepository) *ObservabilityHandler {
	return &ObservabilityHandler{settingRepo: settingRepo}
}

func (h *ObservabilityHandler) PollingStats(c *gin.Context) {
	if h.settingRepo == nil {
		c.JSON(http.StatusOK, gin.H{
			"started":      false,
			"task_count":   0,
			"generated_at": nil,
			"tasks":        []any{},
		})
		return
	}

	var snapshot polling.Snapshot
	err := h.settingRepo.GetJSON(c.Request.Context(), polling.SnapshotSettingKey, &snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if snapshot.GeneratedAt.IsZero() && len(snapshot.Tasks) == 0 && !snapshot.Started {
		c.JSON(http.StatusOK, gin.H{
			"started":      false,
			"task_count":   0,
			"generated_at": nil,
			"tasks":        []any{},
		})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}
