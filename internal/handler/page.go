package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PageHandler struct{}

func NewPageHandler() *PageHandler {
	return &PageHandler{}
}

func (h *PageHandler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"title": "B站UP主运营管理平台"})
}

func (h *PageHandler) Comments(c *gin.Context) {
	c.HTML(http.StatusOK, "comment/comments.html", gin.H{"title": "评论管理"})
}

func (h *PageHandler) Messages(c *gin.Context) {
	c.HTML(http.StatusOK, "message/messages.html", gin.H{"title": "私信管理"})
}

func (h *PageHandler) Interaction(c *gin.Context) {
	c.HTML(http.StatusOK, "like/dashboard.html", gin.H{"title": "互动管理"})
}

func (h *PageHandler) Trends(c *gin.Context) {
	c.HTML(http.StatusOK, "trend/ranking.html", gin.H{"title": "热度分析"})
}

func (h *PageHandler) Settings(c *gin.Context) {
	c.HTML(http.StatusOK, "settings/app.html", gin.H{"title": "系统设置"})
}

func (h *PageHandler) Bilibili(c *gin.Context) {
	c.HTML(http.StatusOK, "settings/bilibili.html", gin.H{"title": "B站配置"})
}
