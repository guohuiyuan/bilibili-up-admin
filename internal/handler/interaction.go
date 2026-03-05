package handler

import (
	"net/http"
	"strconv"

	"bilibili-up-admin/internal/service"

	"github.com/gin-gonic/gin"
)

// InteractionHandler 互动处理器
type InteractionHandler struct {
	svc *service.InteractionService
}

// NewInteractionHandler 创建互动处理器
func NewInteractionHandler(svc *service.InteractionService) *InteractionHandler {
	return &InteractionHandler{svc: svc}
}

// Like 点赞视频
func (h *InteractionHandler) Like(c *gin.Context) {
	bvID := c.Param("id")

	result, err := h.svc.LikeVideo(c.Request.Context(), bvID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Coin 投币视频
func (h *InteractionHandler) Coin(c *gin.Context) {
	bvID := c.Param("id")

	var req struct {
		CoinCount int `json:"coin_count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.CoinCount = 1
	}

	result, err := h.svc.CoinVideo(c.Request.Context(), bvID, req.CoinCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Favorite 收藏视频
func (h *InteractionHandler) Favorite(c *gin.Context) {
	bvID := c.Param("id")

	var req struct {
		MediaID int64 `json:"media_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.FavoriteVideo(c.Request.Context(), bvID, req.MediaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Triple 三连
func (h *InteractionHandler) Triple(c *gin.Context) {
	bvID := c.Param("id")

	result, err := h.svc.TripleAction(c.Request.Context(), bvID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BatchInteract 批量互动
func (h *InteractionHandler) BatchInteract(c *gin.Context) {
	var req struct {
		BVIDs     []string `json:"bv_ids" binding:"required"`
		Action    string   `json:"action" binding:"required"` // like, coin, triple
		CoinCount int      `json:"coin_count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.svc.BatchInteract(c.Request.Context(), req.BVIDs, req.Action, req.CoinCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// InteractFans 互动粉丝视频
func (h *InteractionHandler) InteractFans(c *gin.Context) {
	actionType := c.DefaultQuery("action", "like")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	count, err := h.svc.InteractFansVideos(c.Request.Context(), actionType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "互动完成",
		"count":   count,
	})
}

// Stats 获取统计
func (h *InteractionHandler) Stats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	stats, err := h.svc.GetStats(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// List 获取互动记录列表
func (h *InteractionHandler) List(c *gin.Context) {
	actionType := c.Query("action_type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.List(c.Request.Context(), actionType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// FansVideos 获取粉丝及投稿列表
func (h *InteractionHandler) FansVideos(c *gin.Context) {
	fanLimit, _ := strconv.Atoi(c.DefaultQuery("fan_limit", "20"))
	videoPerFan, _ := strconv.Atoi(c.DefaultQuery("video_per_fan", "5"))

	items, err := h.svc.ListFansVideos(c.Request.Context(), fanLimit, videoPerFan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

// FansList 分页获取粉丝列表
func (h *InteractionHandler) FansList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.ListFans(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// FanVideos 分页获取某粉丝投稿
func (h *InteractionHandler) FanVideos(c *gin.Context) {
	fanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || fanID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fan id"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := h.svc.ListFanVideos(c.Request.Context(), fanID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SyncVideoEngagement 同步视频真实互动数据
func (h *InteractionHandler) SyncVideoEngagement(c *gin.Context) {
	bvID := c.Param("id")
	if bvID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bv id"})
		return
	}

	snapshot, err := h.svc.SyncVideoEngagement(c.Request.Context(), bvID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}
