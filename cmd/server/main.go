package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"slices"

	"bilibili-up-admin/config"
	"bilibili-up-admin/internal/handler"
	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"
	"bilibili-up-admin/internal/service"
	"bilibili-up-admin/pkg/bilibili"
	"bilibili-up-admin/pkg/llm"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	if err := config.Load("config/config.yaml"); err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	zapLogger, err := initLogger()
	if err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	defer zapLogger.Sync()

	db, err := initDatabase()
	if err != nil {
		zapLogger.Fatal("init database failed", zap.Error(err))
	}

	biliClient, err := initBilibiliClient()
	if err != nil {
		zapLogger.Warn("init bilibili client failed", zap.Error(err))
	}

	llmManager, err := initLLMManager()
	if err != nil {
		zapLogger.Warn("init llm manager failed", zap.Error(err))
	}

	repos := initRepositories(db)
	services := initServices(biliClient, llmManager, repos)
	handlers := initHandlers(services)
	router := initRouter(handlers, config.GlobalConfig.Server.Mode)

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	zapLogger.Info("server starting", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		zapLogger.Fatal("server start failed", zap.Error(err))
	}
}

func initLogger() (*zap.Logger, error) {
	cfg := config.GlobalConfig.Log

	var zapConfig zap.Config
	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	switch cfg.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	if cfg.FilePath != "" {
		if err := ensureDir(filepath.Dir(cfg.FilePath)); err != nil {
			return nil, fmt.Errorf("create log dir failed: %w", err)
		}
		zapConfig.OutputPaths = []string{"stdout", cfg.FilePath}
		zapConfig.ErrorOutputPaths = []string{"stderr", cfg.FilePath}
		if cfg.Format == "console" {
			zapConfig.Encoding = "console"
			zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	}

	return zapConfig.Build()
}

func initDatabase() (*gorm.DB, error) {
	dbConfig := config.GlobalConfig.Database

	gormConfig := &gorm.Config{}
	if config.GlobalConfig.Server.Mode == "debug" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	dsn := dbConfig.DSN()
	if dbConfig.Driver == "" || dbConfig.Driver == "sqlite" {
		dir := filepath.Dir(dsn)
		if err := ensureDir(dir); err != nil {
			return nil, fmt.Errorf("create sqlite dir failed: %w", err)
		}
	}

	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("open database failed: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Comment{},
		&model.Message{},
		&model.Interaction{},
		&model.TagRanking{},
		&model.LLMChatLog{},
		&model.Setting{},
		&model.Task{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	return db, nil
}

func ensureDir(dir string) error {
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func loadHTMLTemplates(root string) (*template.Template, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no html templates found under %s", root)
	}
	slices.Sort(files)
	return template.ParseFiles(files...)
}

func initBilibiliClient() (*bilibili.Client, error) {
	cfg := config.GlobalConfig.Bilibili
	if cfg.SESSData == "" || cfg.SESSData == "your_sess_data_here" {
		return nil, fmt.Errorf("missing bilibili sess_data")
	}

	return bilibili.NewClient(&bilibili.Config{
		SESSData: cfg.SESSData,
		BiliJct:  cfg.BiliJct,
		UserID:   cfg.UserID,
	})
}

func initLLMManager() (*llm.Manager, error) {
	manager := llm.NewManager()

	cfg := config.GlobalConfig.LLM
	if shouldRegisterLLM(cfg) {
		providerCfg := &llm.Config{
			Provider:    llm.ProviderType(cfg.Provider),
			APIKey:      cfg.APIKey,
			BaseURL:     cfg.BaseURL,
			Model:       cfg.Model,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		}

		if err := manager.CreateAndRegister("default", providerCfg); err != nil {
			return nil, err
		}
		manager.SetDefault("default")
	}

	for name, providerCfg := range config.GlobalConfig.LLMProviders {
		if !shouldRegisterLLM(providerCfg) {
			continue
		}

		cfg := &llm.Config{
			Provider:    llm.ProviderType(name),
			APIKey:      providerCfg.APIKey,
			BaseURL:     providerCfg.BaseURL,
			Model:       providerCfg.Model,
			MaxTokens:   providerCfg.MaxTokens,
			Temperature: providerCfg.Temperature,
		}
		if err := manager.CreateAndRegister(name, cfg); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func shouldRegisterLLM(cfg config.LLMConfig) bool {
	if cfg.Provider == "ollama" {
		return true
	}
	return cfg.APIKey != "" && cfg.APIKey != "your_api_key_here"
}

type Repositories struct {
	Comment     *repository.CommentRepository
	Message     *repository.MessageRepository
	Interaction *repository.InteractionRepository
	TagRanking  *repository.TagRankingRepository
	LLMChatLog  *repository.LLMChatLogRepository
}

func initRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Comment:     repository.NewCommentRepository(db),
		Message:     repository.NewMessageRepository(db),
		Interaction: repository.NewInteractionRepository(db),
		TagRanking:  repository.NewTagRankingRepository(db),
		LLMChatLog:  repository.NewLLMChatLogRepository(db),
	}
}

type Services struct {
	Comment     *service.CommentService
	Message     *service.MessageService
	Interaction *service.InteractionService
	Trend       *service.TrendService
	LLM         *service.LLMService
}

func initServices(biliClient *bilibili.Client, llmManager *llm.Manager, repos *Repositories) *Services {
	var llmProvider llm.Provider
	if llmManager != nil {
		llmProvider, _ = llmManager.Default()
	}

	return &Services{
		Comment:     service.NewCommentService(biliClient, llmProvider, repos.Comment, repos.LLMChatLog),
		Message:     service.NewMessageService(biliClient, llmProvider, repos.Message, repos.LLMChatLog),
		Interaction: service.NewInteractionService(biliClient, repos.Interaction),
		Trend:       service.NewTrendService(biliClient, repos.TagRanking),
		LLM:         service.NewLLMService(llmManager, repos.LLMChatLog),
	}
}

type Handlers struct {
	Page        *handler.PageHandler
	Comment     *handler.CommentHandler
	Message     *handler.MessageHandler
	Interaction *handler.InteractionHandler
	Trend       *handler.TrendHandler
	LLM         *handler.LLMHandler
}

func initHandlers(services *Services) *Handlers {
	return &Handlers{
		Page:        handler.NewPageHandler(),
		Comment:     handler.NewCommentHandler(services.Comment),
		Message:     handler.NewMessageHandler(services.Message),
		Interaction: handler.NewInteractionHandler(services.Interaction),
		Trend:       handler.NewTrendHandler(services.Trend),
		LLM:         handler.NewLLMHandler(services.LLM),
	}
}

func initRouter(h *Handlers, mode string) *gin.Engine {
	gin.SetMode(mode)
	router := gin.Default()

	if err := router.SetTrustedProxies(config.GlobalConfig.Server.TrustedProxies); err != nil {
		panic(fmt.Errorf("set trusted proxies failed: %w", err))
	}

	tmpl := template.Must(loadHTMLTemplates("web/templates"))
	router.SetHTMLTemplate(tmpl)
	router.Static("/static", "web/static")
	router.Use(corsMiddleware())

	router.GET("/", h.Page.Index)
	router.GET("/comments", h.Page.Comments)
	router.GET("/messages", h.Page.Messages)
	router.GET("/interaction", h.Page.Interaction)
	router.GET("/trends", h.Page.Trends)

	api := router.Group("/api")
	{
		api.GET("/comments", h.Comment.List)
		api.POST("/comments/sync", h.Comment.Sync)
		api.POST("/comments/:id/ai-reply", h.Comment.AIReply)
		api.POST("/comments/:id/reply", h.Comment.ManualReply)
		api.POST("/comments/:id/ignore", h.Comment.Ignore)
		api.POST("/comments/batch-ai-reply", h.Comment.BatchAIReply)

		api.GET("/messages", h.Message.List)
		api.POST("/messages/sync", h.Message.Sync)
		api.GET("/messages/unread", h.Message.UnreadCount)
		api.POST("/messages/:id/ai-reply", h.Message.AIReply)
		api.POST("/messages/:id/reply", h.Message.ManualReply)
		api.POST("/messages/:id/ignore", h.Message.Ignore)

		api.GET("/interactions", h.Interaction.List)
		api.GET("/interactions/stats", h.Interaction.Stats)
		api.POST("/videos/:id/like", h.Interaction.Like)
		api.POST("/videos/:id/coin", h.Interaction.Coin)
		api.POST("/videos/:id/triple", h.Interaction.Triple)
		api.POST("/videos/batch-interact", h.Interaction.BatchInteract)
		api.POST("/fans/interact", h.Interaction.InteractFans)

		api.GET("/trends/tags", h.Trend.TrendingTags)
		api.GET("/trends/tags/:name", h.Trend.TagDetail)
		api.GET("/trends/videos", h.Trend.VideoRanking)
		api.GET("/trends/historical", h.Trend.HistoricalRankings)
		api.GET("/trends/latest", h.Trend.LatestRankings)
		api.GET("/trends/search", h.Trend.SearchTag)
		api.POST("/trends/sync", h.Trend.Sync)
		api.GET("/trends/stats", h.Trend.Stats)

		api.POST("/llm/chat", h.LLM.Chat)
		api.GET("/llm/providers", h.LLM.Providers)
		api.POST("/llm/default", h.LLM.SetDefault)
		api.GET("/llm/test/:provider", h.LLM.Test)
		api.GET("/llm/stats", h.LLM.Stats)

		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

type ContextKey string

const (
	ContextKeyDB      ContextKey = "db"
	ContextKeyBili    ContextKey = "bili"
	ContextKeyLLM     ContextKey = "llm"
	ContextKeyContext ContextKey = "context"
)

func GetContext(c *gin.Context) context.Context {
	ctx, exists := c.Get(string(ContextKeyContext))
	if !exists {
		return c.Request.Context()
	}
	return ctx.(context.Context)
}
