package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	DataDir  string         `mapstructure:"data_dir"`
}

type ServerConfig struct {
	Port           int      `mapstructure:"port"`
	Mode           string   `mapstructure:"mode"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
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
	viper.SetDefault("server.trusted_proxies", []string{"127.0.0.1", "::1"})
	viper.SetDefault("data_dir", "data")
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.path", "bilibili-up-admin.db")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.charset", "utf8mb4")
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
