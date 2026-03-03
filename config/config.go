package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig         `mapstructure:"server"`
	Database     DatabaseConfig       `mapstructure:"database"`
	Redis        RedisConfig          `mapstructure:"redis"`
	Bilibili     BilibiliConfig       `mapstructure:"bilibili"`
	LLM          LLMConfig            `mapstructure:"llm"`
	LLMProviders map[string]LLMConfig `mapstructure:"llm_providers"`
	Task         TaskConfig           `mapstructure:"task"`
	Log          LogConfig            `mapstructure:"log"`
	DataDir      string               `mapstructure:"data_dir"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Path     string `mapstructure:"path"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

func (c *DatabaseConfig) DSN() string {
	if c.Driver == "sqlite" || c.Driver == "" {
		if c.Path == "" {
			return "bilibili-up-admin.db"
		}
		return c.Path
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.DBName, c.Charset)
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type BilibiliConfig struct {
	SESSData string `mapstructure:"sess_data"`
	BiliJct  string `mapstructure:"bili_jct"`
	UserID   int64  `mapstructure:"user_id"`
}

type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	APIKey      string  `mapstructure:"api_key"`
	BaseURL     string  `mapstructure:"base_url"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

type TaskConfig struct {
	WorkerCount int `mapstructure:"worker_count"`
	QueueSize   int `mapstructure:"queue_size"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	FilePath string `mapstructure:"file_path"`
}

var GlobalConfig *Config

func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("BILI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("unmarshal config failed: %w", err)
	}

	GlobalConfig = &cfg
	normalizePaths(GlobalConfig)
	return nil
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("data_dir", "data")
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.path", "bilibili-up-admin.db")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.charset", "utf8mb4")
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.max_tokens", 1000)
	viper.SetDefault("llm.temperature", 0.7)
	viper.SetDefault("task.worker_count", 5)
	viper.SetDefault("task.queue_size", 100)
	viper.SetDefault("log.level", "debug")
	viper.SetDefault("log.format", "console")
	viper.SetDefault("log.file_path", "logs/bilibili-up-admin.log")
}

func GetLLMConfig(provider string) *LLMConfig {
	if provider == "" {
		provider = GlobalConfig.LLM.Provider
	}

	if cfg, ok := GlobalConfig.LLMProviders[provider]; ok {
		cfg.Provider = provider
		return &cfg
	}

	cfg := GlobalConfig.LLM
	if cfg.Provider == "" {
		cfg.Provider = provider
	}
	return &cfg
}

func normalizePaths(cfg *Config) {
	if cfg == nil {
		return
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}

	if cfg.Database.Driver == "" || cfg.Database.Driver == "sqlite" {
		cfg.Database.Path = ensureUnderDataDir(cfg.Database.Path, cfg.DataDir)
	}
	cfg.Log.FilePath = ensureUnderDataDir(cfg.Log.FilePath, cfg.DataDir)
}

func ensureUnderDataDir(path, dataDir string) string {
	if path == "" {
		return ""
	}
	if isAbs(path) || strings.HasPrefix(path, dataDir+"/") || strings.HasPrefix(path, dataDir+"\\") {
		return path
	}
	return dataDir + "/" + path
}

func isAbs(path string) bool {
	if len(path) > 1 && path[1] == ':' {
		return true
	}
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\")
}
