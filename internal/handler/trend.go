package handler

import (
	"bilibili-up-admin/internal/service"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TrendHandler 热度处理器
type TrendHandler struct {
	svc *service.TrendService
}

// NewTrendHandler 创建热度处理器
func NewTrendHandler(svc *service.TrendService) *TrendHandler {
	return &TrendHandler{svc: svc}
}

// TrendingTags 获取热门标签
func (h *TrendHandler) TrendingTags(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	category := c.Query("category")

	tags, err := h.svc.GetTrendingTags(c.Request.Context(), category, limit)
	if err != nil {
		log.Printf("[trend.tags] category=%q limit=%d err=%v", category, limit, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(tags) == 0 {
		log.Printf("[trend.tags] category=%q limit=%d empty result", category, limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"tags":     tags,
		"count":    len(tags),
	})
}

// TagDetail 获取标签详情
func (h *TrendHandler) TagDetail(c *gin.Context) {
	tagName := c.Param("name")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	detail, err := h.svc.GetTagDetail(c.Request.Context(), tagName, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// VideoRanking 获取视频排行
func (h *TrendHandler) VideoRanking(c *gin.Context) {
	category := c.DefaultQuery("category", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	ranking, err := h.svc.GetVideoRanking(c.Request.Context(), category, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ranking)
}

// HistoricalRankings 获取历史排行
func (h *TrendHandler) HistoricalRankings(c *gin.Context) {
	date := c.Query("date")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	rankings, err := h.svc.GetHistoricalRankings(c.Request.Context(), date, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":     date,
		"rankings": rankings,
	})
}

// LatestRankings 获取最新排行
func (h *TrendHandler) LatestRankings(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	category := c.DefaultQuery("category", "")

	rankings, err := h.svc.GetLatestRankings(c.Request.Context(), category, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"rankings": rankings,
	})
}

// SearchTag 搜索标签
func (h *TrendHandler) SearchTag(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tags, err := h.svc.SearchTag(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"keyword": keyword,
		"tags":    tags,
	})
}

// Sync 同步热度数据
func (h *TrendHandler) Sync(c *gin.Context) {
	if err := h.svc.DailySync(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "同步成功"})
}

// Stats 获取热度统计
func (h *TrendHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
