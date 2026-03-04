package handler

import (
	"context"
	"net/http"

	appruntime "bilibili-up-admin/internal/runtime"
	"bilibili-up-admin/internal/service"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settings *service.AppSettingsService
	runtime  *appruntime.Store
}

func NewSettingsHandler(settings *service.AppSettingsService, runtime *appruntime.Store) *SettingsHandler {
	return &SettingsHandler{settings: settings, runtime: runtime}
}

func (h *SettingsHandler) GetApp(c *gin.Context) {
	settings, err := h.settings.Load(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) SaveApp(c *gin.Context) {
	var req struct {
		LLM  service.LLMSettings  `json:"llm"`
		Task service.TaskSettings `json:"task"`
		Log  service.LogSettings  `json:"log"`
		// 注意：不再接受 llm_providers，因为现在通过独立的 API 管理
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 分别保存各个设置项
	current, err := h.settings.Load(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新设置
	current.LLM = req.LLM
	current.Task = req.Task
	current.Log = req.Log

	// 保存到数据库
	if err := h.settings.SaveApp(c.Request.Context(), current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := applyRuntimeSettings(c.Request.Context(), h.settings, h.runtime, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, current)
}

func (h *SettingsHandler) GetBilibili(c *gin.Context) {
	settings, err := h.settings.Load(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings.Bilibili)
}

func (h *SettingsHandler) SaveBilibiliCookie(c *gin.Context) {
	var req struct {
		Cookie string `json:"cookie" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, err := bilibili.NewClientFromCookieString(req.Cookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := client.GetUserInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := client.GetConfig()
	current, err := h.settings.SaveBilibili(c.Request.Context(), service.BilibiliSettings{
		SESSData:   cfg.SESSData,
		BiliJct:    cfg.BiliJct,
		UserID:     cfg.UserID,
		Cookie:     req.Cookie,
		UserName:   user.Name,
		UserFace:   user.Face,
		IsLoggedIn: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := applyRuntimeSettings(c.Request.Context(), h.settings, h.runtime, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, current.Bilibili)
}

func (h *SettingsHandler) GenerateBilibiliQRCode(c *gin.Context) {
	qr, err := bilibili.GenerateLoginQRCode(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, qr)
}

func (h *SettingsHandler) PollBilibiliQRCode(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing qrcode key"})
		return
	}
	state, err := bilibili.PollLoginQRCode(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !state.Success || state.Client == nil {
		c.JSON(http.StatusOK, state)
		return
	}
	cfg := state.Client.GetConfig()
	current, err := h.settings.SaveBilibili(c.Request.Context(), service.BilibiliSettings{
		SESSData:   cfg.SESSData,
		BiliJct:    cfg.BiliJct,
		UserID:     cfg.UserID,
		UserName:   state.User.Name,
		UserFace:   state.User.Face,
		IsLoggedIn: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := applyRuntimeSettings(c.Request.Context(), h.settings, h.runtime, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    state.User,
	})
}

func applyRuntimeSettings(ctx context.Context, settingsSvc *service.AppSettingsService, store *appruntime.Store, settings *service.AppSettings) error {
	if settings == nil {
		var err error
		settings, err = settingsSvc.Load(ctx)
		if err != nil {
			return err
		}
	}
	biliClient, err := service.BuildBilibiliClient(settings.Bilibili)
	if err != nil {
		return err
	}
	llmManager, err := service.BuildLLMManager(settings)
	if err != nil {
		return err
	}
	store.Apply(biliClient, llmManager)
	return nil
}

func (h *SettingsHandler) GetLLMChannels(c *gin.Context) {
	c.JSON(http.StatusOK, llm.SupportedProviders())
}

func (h *SettingsHandler) GetLLMProviders(c *gin.Context) {
	settings, err := h.settings.Load(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings.LLMProviders)
}

func (h *SettingsHandler) AddLLMProvider(c *gin.Context) {
	var req struct {
		Name     string                      `json:"name" binding:"required"`
		Settings service.LLMProviderSettings `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	current, err := h.settings.AddOrUpdateLLMProvider(c.Request.Context(), req.Name, req.Settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := applyRuntimeSettings(c.Request.Context(), h.settings, h.runtime, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SettingsHandler) UpdateLLMProvider(c *gin.Context) {
	name := c.Param("name")
	var settings service.LLMProviderSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	current, err := h.settings.AddOrUpdateLLMProvider(c.Request.Context(), name, settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := applyRuntimeSettings(c.Request.Context(), h.settings, h.runtime, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SettingsHandler) DeleteLLMProvider(c *gin.Context) {
	name := c.Param("name")
	current, err := h.settings.DeleteLLMProvider(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := applyRuntimeSettings(c.Request.Context(), h.settings, h.runtime, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
