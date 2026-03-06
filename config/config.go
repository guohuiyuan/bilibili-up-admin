package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

//go:embed config.yaml
var embeddedConfigYAML []byte

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
	v := viper.New()
	v.SetEnvPrefix("BILI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := loadConfigSource(v, configPath); err != nil {
		return err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("unmarshal config failed: %w", err)
	}

	GlobalConfig = &cfg
	normalizePaths(GlobalConfig)
	return nil
}

func loadConfigSource(v *viper.Viper, configPath string) error {
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("read config failed: %w", err)
		}
	}

	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(embeddedConfigYAML)); err != nil {
		return fmt.Errorf("read embedded config failed: %w", err)
	}

	return nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.trusted_proxies", []string{"127.0.0.1", "::1"})
	v.SetDefault("data_dir", "data")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.path", "bilibili-up-admin.db")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.charset", "utf8mb4")
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
