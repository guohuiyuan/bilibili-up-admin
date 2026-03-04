package service

import (
	"context"
	"encoding/json"
	"fmt"

	"bilibili-up-admin/internal/repository"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"
)

const (
	settingKeyBilibili     = "app.bilibili"
	settingKeyLLM          = "app.llm"
	settingKeyLLMProviders = "app.llm_providers"
	settingKeyTask         = "app.task"
	settingKeyLog          = "app.log"
)

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

type AppSettings struct {
	Bilibili     BilibiliSettings               `json:"bilibili"`
	LLM          LLMSettings                    `json:"llm"`
	LLMProviders map[string]LLMProviderSettings `json:"llm_providers"`
	Task         TaskSettings                   `json:"task"`
	Log          LogSettings                    `json:"log"`
}

type AppSettingsService struct {
	repo *repository.SettingRepository
}

func NewAppSettingsService(repo *repository.SettingRepository) *AppSettingsService {
	return &AppSettingsService{repo: repo}
}

func DefaultAppSettings() *AppSettings {
	return &AppSettings{
		LLM: LLMSettings{
			DefaultProvider: "openai",
		},
		LLMProviders: map[string]LLMProviderSettings{
			"openai":   {Enabled: false, Provider: "openai", Model: "gpt-4o-mini", MaxTokens: 1000, Temperature: 0.7},
			"claude":   {Enabled: false, Provider: "claude", Model: "claude-3-5-sonnet-latest", MaxTokens: 1000, Temperature: 0.7},
			"gemini":   {Enabled: false, Provider: "gemini", Model: "gemini-2.5-flash", MaxTokens: 1000, Temperature: 0.7},
			"deepseek": {Enabled: false, Provider: "deepseek", Model: "deepseek-chat", MaxTokens: 1000, Temperature: 0.7},
			"qwen":     {Enabled: false, Provider: "qwen", Model: "qwen-plus", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", MaxTokens: 1000, Temperature: 0.7},
			"moonshot": {Enabled: false, Provider: "moonshot", Model: "moonshot-v1-8k", MaxTokens: 1000, Temperature: 0.7},
			"zhipu":    {Enabled: false, Provider: "zhipu", Model: "glm-4-flash", MaxTokens: 1000, Temperature: 0.7},
			"ollama":   {Enabled: false, Provider: "ollama", Model: "llama3.1", BaseURL: "http://localhost:11434", MaxTokens: 1000, Temperature: 0.7},
		},
		Task: TaskSettings{
			WorkerCount: 5,
			QueueSize:   100,
		},
		Log: LogSettings{
			Level:    "debug",
			Format:   "console",
			FilePath: "logs/bilibili-up-admin.log",
		},
	}
}

func CloneAppSettings(src *AppSettings) *AppSettings {
	if src == nil {
		return DefaultAppSettings()
	}
	clone := *src
	if src.LLMProviders != nil {
		clone.LLMProviders = make(map[string]LLMProviderSettings, len(src.LLMProviders))
		for k, v := range src.LLMProviders {
			clone.LLMProviders[k] = v
		}
	}
	return &clone
}

func (s *AppSettingsService) Load(ctx context.Context) (*AppSettings, error) {
	settings := DefaultAppSettings()
	if err := s.repo.GetJSON(ctx, settingKeyBilibili, &settings.Bilibili); err != nil {
		return nil, err
	}
	if err := s.repo.GetJSON(ctx, settingKeyLLM, &settings.LLM); err != nil {
		return nil, err
	}
	if err := s.repo.GetJSON(ctx, settingKeyLLMProviders, &settings.LLMProviders); err != nil {
		return nil, err
	}
	if err := s.repo.GetJSON(ctx, settingKeyTask, &settings.Task); err != nil {
		return nil, err
	}
	if err := s.repo.GetJSON(ctx, settingKeyLog, &settings.Log); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *AppSettingsService) SaveApp(ctx context.Context, settings *AppSettings) error {
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}
	if settings.LLMProviders == nil {
		settings.LLMProviders = map[string]LLMProviderSettings{}
	}
	if err := s.repo.SetJSON(ctx, settingKeyBilibili, settings.Bilibili); err != nil {
		return err
	}
	if err := s.repo.SetJSON(ctx, settingKeyLLM, settings.LLM); err != nil {
		return err
	}
	if err := s.repo.SetJSON(ctx, settingKeyLLMProviders, settings.LLMProviders); err != nil {
		return err
	}
	if err := s.repo.SetJSON(ctx, settingKeyTask, settings.Task); err != nil {
		return err
	}
	if err := s.repo.SetJSON(ctx, settingKeyLog, settings.Log); err != nil {
		return err
	}
	return nil
}

func (s *AppSettingsService) SaveGeneral(ctx context.Context, llmSettings LLMSettings, providers map[string]LLMProviderSettings, task TaskSettings, log LogSettings) (*AppSettings, error) {
	current, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	current.LLM = llmSettings
	current.LLMProviders = providers
	current.Task = task
	current.Log = log
	if err := s.SaveApp(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
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

func BuildBilibiliClient(settings BilibiliSettings) (*bilibili.Client, error) {
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

func EncodeJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *AppSettingsService) AddOrUpdateLLMProvider(ctx context.Context, name string, provider LLMProviderSettings) (*AppSettings, error) {
	current, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	if current.LLMProviders == nil {
		current.LLMProviders = make(map[string]LLMProviderSettings)
	}
	current.LLMProviders[name] = provider
	if err := s.SaveApp(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *AppSettingsService) DeleteLLMProvider(ctx context.Context, name string) (*AppSettings, error) {
	current, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	if current.LLMProviders != nil {
		delete(current.LLMProviders, name)
		if err := s.SaveApp(ctx, current); err != nil {
			return nil, err
		}
	}
	return current, nil
}
