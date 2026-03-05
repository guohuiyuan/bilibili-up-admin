package handler

import (
	"net/http"
	"strings"

	"bilibili-up-admin/internal/service"

	"github.com/gin-gonic/gin"
)

func AdminAuthRequired(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(AuthCookieName)
		if strings.TrimSpace(token) == "" {
			handleUnauthorized(c)
			return
		}

		user, err := auth.ValidateSession(c.Request.Context(), token)
		if err != nil || user == nil {
			handleUnauthorized(c)
			return
		}

		c.Set("current_admin", user)
		c.Next()
	}
}

func RequirePasswordChanged() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentAdmin(c)
		if !ok || user == nil {
			handleUnauthorized(c)
			return
		}

		if !user.MustChangePassword {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if path == "/admin/account/password" || path == "/admin/api/auth/change-password" || path == "/admin/api/auth/me" || path == "/admin/api/auth/logout" {
			c.Next()
			return
		}

		if strings.HasPrefix(path, "/admin/api/") {
			c.JSON(http.StatusForbidden, gin.H{
				"error":                "首次登录需先修改密码",
				"must_change_password": true,
			})
			c.Abort()
			return
		}

		c.Redirect(http.StatusFound, "/admin/account/password")
		c.Abort()
	}
}

func handleUnauthorized(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/admin/api/") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		c.Abort()
		return
	}
	c.Redirect(http.StatusFound, "/admin/login")
	c.Abort()
}
