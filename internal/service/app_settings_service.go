package service

import (
	"context"
	"encoding/json"
	"log"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

const (
	settingKeyBilibili = "app.bilibili"
	settingKeyLLM      = "app.llm"
	settingKeyTask     = "app.task"
	settingKeyLog      = "app.log"
	settingKeyInteract = "app.interaction"
	// 注意：删除了 settingKeyLLMProviders，因为不再使用 KV 存储
)

// 数据结构定义 (BilibiliSettings, LLMSettings 等保持不变) ...
type BilibiliSettings struct {
	SESSData   string `json:"sess_data"`
	BiliJct    string `json:"bili_jct"`
	UserID     int64  `json:"user_id"`
	Cookie     string `json:"cookie,omitempty"`
	UserName   string `json:"user_name"`
	UserFace   string `json:"user_face"`
	IsLoggedIn bool   `json:"is_logged_in"`
}

type LLMSettings struct {
	DefaultProvider string `json:"default_provider"`
}

type LLMProviderSettings struct {
	Enabled     bool    `json:"enabled"`
	Provider    string  `json:"provider"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type TaskSettings struct {
	WorkerCount int `json:"worker_count"`
	QueueSize   int `json:"queue_size"`
}

type LogSettings struct {
	Level    string `json:"level"`
	Format   string `json:"format"`
	FilePath string `json:"file_path"`
}

type InteractionRuleSettings struct {
	EnableLike             bool   `json:"enable_like"`
	EnableCoin             bool   `json:"enable_coin"`
	EnableFavorite         bool   `json:"enable_favorite"`
	EnableFollowAutoReply  bool   `json:"enable_follow_auto_reply"`
	FollowAutoReplyContent string `json:"follow_auto_reply_content"`
	CoinPlayThreshold      int64  `json:"coin_play_threshold"`
	FavoriteMediaID        int64  `json:"favorite_media_id"`
	FanPageSize            int    `json:"fan_page_size"`
	VideoPageSize          int    `json:"video_page_size"`
	RequestIntervalSeconds int    `json:"request_interval_seconds"`
}

type AppSettings struct {
	Bilibili     BilibiliSettings               `json:"bilibili"`
	LLM          LLMSettings                    `json:"llm"`
	LLMProviders map[string]LLMProviderSettings `json:"llm_providers"`
	Task         TaskSettings                   `json:"task"`
	Log          LogSettings                    `json:"log"`
	Interaction  InteractionRuleSettings        `json:"interaction"`
}

type AppSettingsService struct {
	settingRepo  *repository.SettingRepository
	providerRepo *repository.LLMProviderRepository // 新增独立的 Provider 仓库
}

// 修改构造函数
func NewAppSettingsService(settingRepo *repository.SettingRepository, providerRepo *repository.LLMProviderRepository) *AppSettingsService {
	return &AppSettingsService{
		settingRepo:  settingRepo,
		providerRepo: providerRepo,
	}
}

func (s *AppSettingsService) Load(ctx context.Context) (*AppSettings, error) {
	app := &AppSettings{
		LLMProviders: make(map[string]LLMProviderSettings),
	}

	// 1. 加载旧的常规配置 (KV 表)
	if setting, _ := s.settingRepo.GetByKey(ctx, settingKeyBilibili); setting != nil {
		json.Unmarshal([]byte(setting.Value), &app.Bilibili)
	}
	if setting, _ := s.settingRepo.GetByKey(ctx, settingKeyLLM); setting != nil {
		json.Unmarshal([]byte(setting.Value), &app.LLM)
	}
	if setting, _ := s.settingRepo.GetByKey(ctx, settingKeyTask); setting != nil {
		json.Unmarshal([]byte(setting.Value), &app.Task)
	}
	if setting, _ := s.settingRepo.GetByKey(ctx, settingKeyLog); setting != nil {
		json.Unmarshal([]byte(setting.Value), &app.Log)
	}
	if setting, _ := s.settingRepo.GetByKey(ctx, settingKeyInteract); setting != nil {
		json.Unmarshal([]byte(setting.Value), &app.Interaction)
	}

	if app.Interaction.CoinPlayThreshold <= 0 {
		app.Interaction.CoinPlayThreshold = 10000
	}
	if app.Interaction.FanPageSize <= 0 {
		app.Interaction.FanPageSize = 20
	}
	if app.Interaction.VideoPageSize <= 0 {
		app.Interaction.VideoPageSize = 5
	}
	if app.Interaction.RequestIntervalSeconds <= 0 {
		app.Interaction.RequestIntervalSeconds = 3
	}
	if app.Interaction.FollowAutoReplyContent == "" {
		app.Interaction.FollowAutoReplyContent = "感谢关注！欢迎常来评论区交流～"
	}

	// 2. [核心改变]：从独立的 llm_providers 实体表中提取模型配置
	dbProviders, err := s.providerRepo.List(ctx)
	if err == nil {
		for _, p := range dbProviders {
			app.LLMProviders[p.Name] = LLMProviderSettings{
				Enabled:     p.Enabled,
				Provider:    p.Provider,
				APIKey:      p.APIKey,
				BaseURL:     p.BaseURL,
				Model:       p.Model,
				MaxTokens:   p.MaxTokens,
				Temperature: p.Temperature,
			}
		}
	}

	return app, nil
}

func (s *AppSettingsService) SaveApp(ctx context.Context, app *AppSettings) error {
	vBilibili, _ := EncodeJSON(app.Bilibili)
	vLLM, _ := EncodeJSON(app.LLM)
	vTask, _ := EncodeJSON(app.Task)
	vLog, _ := EncodeJSON(app.Log)
	vInteract, _ := EncodeJSON(app.Interaction)

	if err := s.settingRepo.Set(ctx, settingKeyBilibili, vBilibili); err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, settingKeyLLM, vLLM); err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, settingKeyTask, vTask); err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, settingKeyLog, vLog); err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, settingKeyInteract, vInteract); err != nil {
		return err
	}
	log.Printf("[settings.save] llm_default=%q worker=%d queue=%d log_level=%q log_format=%q interact={like:%v coin:%v favorite:%v follow_auto_reply:%v fan_page:%d video_page:%d interval_sec:%d follow_reply_len:%d}",
		app.LLM.DefaultProvider,
		app.Task.WorkerCount,
		app.Task.QueueSize,
		app.Log.Level,
		app.Log.Format,
		app.Interaction.EnableLike,
		app.Interaction.EnableCoin,
		app.Interaction.EnableFavorite,
		app.Interaction.EnableFollowAutoReply,
		app.Interaction.FanPageSize,
		app.Interaction.VideoPageSize,
		app.Interaction.RequestIntervalSeconds,
		len(app.Interaction.FollowAutoReplyContent),
	)
	// 不再保存 app.LLMProviders 到 KV 数据库中，完全由下方的独立接口负责
	return nil
}

func (s *AppSettingsService) SaveBilibili(ctx context.Context, bilibiliSettings BilibiliSettings) (*AppSettings, error) {
	current, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	current.Bilibili = bilibiliSettings
	if err := s.SaveApp(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

// --- 真正对接数据库表的 CRUD 方法 ---

func (s *AppSettingsService) AddOrUpdateLLMProvider(ctx context.Context, name string, settings LLMProviderSettings) (*AppSettings, error) {
	provider := &model.LLMProvider{
		Name:        name,
		Provider:    settings.Provider,
		APIKey:      settings.APIKey,
		BaseURL:     settings.BaseURL,
		Model:       settings.Model,
		MaxTokens:   settings.MaxTokens,
		Temperature: settings.Temperature,
		Enabled:     settings.Enabled,
	}

	// 将其保存到新的实体表中
	if err := s.providerRepo.Save(ctx, provider); err != nil {
		return nil, err
	}

	return s.Load(ctx) // 重新加载完整配置返回给前端
}

func (s *AppSettingsService) DeleteLLMProvider(ctx context.Context, name string) (*AppSettings, error) {
	// 从新实体表中删除
	if err := s.providerRepo.Delete(ctx, name); err != nil {
		return nil, err
	}
	return s.Load(ctx)
}

func EncodeJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func BuildBilibiliClient(settings BilibiliSettings) (*bilibili.Client, error) {
	if settings.Cookie != "" {
		return bilibili.NewClientFromCookieString(settings.Cookie)
	}
	if settings.SESSData == "" {
		return nil, nil
	}
	return bilibili.NewClient(&bilibili.Config{
		SESSData: settings.SESSData,
		BiliJct:  settings.BiliJct,
		UserID:   settings.UserID,
	})
}

func BuildLLMManager(settings *AppSettings) (*llm.Manager, error) {
	if settings == nil {
		return nil, nil
	}
	manager := llm.NewManager()
	for name, providerCfg := range settings.LLMProviders {
		if !providerCfg.Enabled {
			continue
		}
		if providerCfg.Provider != "ollama" && providerCfg.APIKey == "" {
			continue
		}
		cfg := &llm.Config{
			Provider:    llm.ProviderType(providerCfg.Provider),
			APIKey:      providerCfg.APIKey,
			BaseURL:     providerCfg.BaseURL,
			Model:       providerCfg.Model,
			MaxTokens:   providerCfg.MaxTokens,
			Temperature: providerCfg.Temperature,
		}
		if cfg.Provider == "" {
			cfg.Provider = llm.ProviderType(name)
		}
		if err := manager.CreateAndRegister(name, cfg); err != nil {
			return nil, err
		}
	}
	if len(manager.Names()) == 0 {
		return nil, nil
	}
	if settings.LLM.DefaultProvider != "" {
		manager.SetDefault(settings.LLM.DefaultProvider)
	}
	return manager, nil
}
