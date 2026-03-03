package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PageHandler 页面处理器
type PageHandler struct{}

// NewPageHandler 创建页面处理器
func NewPageHandler() *PageHandler {
	return &PageHandler{}
}

// Index 首页
func (h *PageHandler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "B站UP主运营管理平台",
	})
}

// Comments 评论管理页面
func (h *PageHandler) Comments(c *gin.Context) {
	c.HTML(http.StatusOK, "comment/list.html", gin.H{
		"title": "评论管理",
	})
}

// Messages 私信管理页面
func (h *PageHandler) Messages(c *gin.Context) {
	c.HTML(http.StatusOK, "message/list.html", gin.H{
		"title": "私信管理",
	})
}

// Interaction 互动管理页面
func (h *PageHandler) Interaction(c *gin.Context) {
	c.HTML(http.StatusOK, "like/dashboard.html", gin.H{
		"title": "互动管理",
	})
}

// Trends 热度分析页面
func (h *PageHandler) Trends(c *gin.Context) {
	c.HTML(http.StatusOK, "trend/ranking.html", gin.H{
		"title": "热度分析",
	})
}
