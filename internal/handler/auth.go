package handler

import (
	"net/http"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/service"

	"github.com/gin-gonic/gin"
)

const AuthCookieName = "bu_admin_token"

// AuthHandler 管理员认证处理器
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Service() *service.AuthService {
	return h.svc
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	maxAge := 7 * 24 * 60 * 60
	c.SetCookie(AuthCookieName, result.Token, maxAge, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"username":             result.Username,
		"must_change_password": result.MustChangePassword,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(AuthCookieName)
	_ = h.svc.Logout(c.Request.Context(), token)
	c.SetCookie(AuthCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := CurrentAdmin(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                   user.ID,
		"username":             user.Username,
		"must_change_password": user.MustChangePassword,
		"last_login_at":        user.LastLoginAt,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, ok := CurrentAdmin(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

func CurrentAdmin(c *gin.Context) (*model.AdminUser, bool) {
	v, ok := c.Get("current_admin")
	if !ok {
		return nil, false
	}
	user, ok := v.(*model.AdminUser)
	return user, ok
}
