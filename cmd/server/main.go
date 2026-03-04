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
	appruntime "bilibili-up-admin/internal/runtime"
	"bilibili-up-admin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
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

	db, err := initDatabase()
	if err != nil {
		log.Fatalf("init database failed: %v", err)
	}

	repos := initRepositories(db)
	settingsSvc := service.NewAppSettingsService(repos.Setting)
	runtimeStore := appruntime.NewStore()

	appSettings, err := settingsSvc.Load(context.Background())
	if err != nil {
		log.Fatalf("load app settings failed: %v", err)
	}
	biliClient, err := service.BuildBilibiliClient(appSettings.Bilibili)
	if err != nil {
		log.Printf("init bilibili client failed: %v", err)
	}
	llmManager, err := service.BuildLLMManager(appSettings)
	if err != nil {
		log.Printf("init llm manager failed: %v", err)
	}
	runtimeStore.Apply(biliClient, llmManager)

	zapLogger, err := initLogger(appSettings.Log, config.GlobalConfig.DataDir)
	if err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	defer zapLogger.Sync()

	services := initServices(runtimeStore, settingsSvc, repos)
	handlers := initHandlers(services, settingsSvc, runtimeStore)
	router := initRouter(handlers, config.GlobalConfig.Server.Mode)

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	zapLogger.Info("server starting", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		zapLogger.Fatal("server start failed", zap.Error(err))
	}
}

func initLogger(cfg service.LogSettings, dataDir string) (*zap.Logger, error) {
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
		cfg.FilePath = ensureUnderDataDir(cfg.FilePath, dataDir)
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

func ensureUnderDataDir(path, dataDir string) string {
	if path == "" || dataDir == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dataDir, path)
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

type templateRenderer struct {
	templates map[string]*template.Template
}

func (r templateRenderer) Instance(name string, data any) render.Render {
	tmpl, ok := r.templates[name]
	if !ok {
		return render.HTML{
			Template: template.Must(template.New(name).Parse("template not found")),
			Name:     name,
			Data:     data,
		}
	}
	return render.HTML{
		Template: tmpl,
		Name:     name,
		Data:     data,
	}
}

func buildHTMLRenderer(root string) (render.HTMLRender, error) {
	basePath := filepath.Join(root, "layout", "base.html")
	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("read base template failed: %w", err)
	}

	renderer := templateRenderer{templates: make(map[string]*template.Template)}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".html" || path == basePath {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		pageContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl := template.New(relPath)
		if _, err := tmpl.Parse(`{{ template "layout/base.html" . }}`); err != nil {
			return err
		}
		if _, err := tmpl.New("layout/base.html").Parse(string(baseContent)); err != nil {
			return err
		}
		if _, err := tmpl.Parse(string(pageContent)); err != nil {
			return err
		}

		renderer.templates[relPath] = tmpl
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(renderer.templates) == 0 {
		return nil, fmt.Errorf("no page templates found under %s", root)
	}
	return renderer, nil
}

type Repositories struct {
	Comment     *repository.CommentRepository
	Message     *repository.MessageRepository
	Interaction *repository.InteractionRepository
	TagRanking  *repository.TagRankingRepository
	LLMChatLog  *repository.LLMChatLogRepository
	Setting     *repository.SettingRepository
}

func initRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Comment:     repository.NewCommentRepository(db),
		Message:     repository.NewMessageRepository(db),
		Interaction: repository.NewInteractionRepository(db),
		TagRanking:  repository.NewTagRankingRepository(db),
		LLMChatLog:  repository.NewLLMChatLogRepository(db),
		Setting:     repository.NewSettingRepository(db),
	}
}

type Services struct {
	Comment     *service.CommentService
	Message     *service.MessageService
	Interaction *service.InteractionService
	Trend       *service.TrendService
	LLM         *service.LLMService
	Settings    *service.AppSettingsService
}

func initServices(runtime *appruntime.Store, settings *service.AppSettingsService, repos *Repositories) *Services {
	return &Services{
		Comment:     service.NewCommentService(runtime, repos.Comment, repos.LLMChatLog),
		Message:     service.NewMessageService(runtime, repos.Message, repos.LLMChatLog),
		Interaction: service.NewInteractionService(runtime, repos.Interaction),
		Trend:       service.NewTrendService(runtime, repos.TagRanking),
		LLM:         service.NewLLMService(runtime, repos.LLMChatLog),
		Settings:    settings,
	}
}

type Handlers struct {
	Page        *handler.PageHandler
	Comment     *handler.CommentHandler
	Message     *handler.MessageHandler
	Interaction *handler.InteractionHandler
	Trend       *handler.TrendHandler
	LLM         *handler.LLMHandler
	Settings    *handler.SettingsHandler
}

func initHandlers(services *Services, settings *service.AppSettingsService, runtime *appruntime.Store) *Handlers {
	return &Handlers{
		Page:        handler.NewPageHandler(),
		Comment:     handler.NewCommentHandler(services.Comment),
		Message:     handler.NewMessageHandler(services.Message),
		Interaction: handler.NewInteractionHandler(services.Interaction),
		Trend:       handler.NewTrendHandler(services.Trend),
		LLM:         handler.NewLLMHandler(services.LLM),
		Settings:    handler.NewSettingsHandler(settings, runtime),
	}
}

func initRouter(h *Handlers, mode string) *gin.Engine {
	gin.SetMode(mode)
	router := gin.Default()

	if err := router.SetTrustedProxies(config.GlobalConfig.Server.TrustedProxies); err != nil {
		panic(fmt.Errorf("set trusted proxies failed: %w", err))
	}

	htmlRenderer, err := buildHTMLRenderer("web/templates")
	if err != nil {
		panic(fmt.Errorf("build html renderer failed: %w", err))
	}
	router.HTMLRender = htmlRenderer
	router.Static("/static", "web/static")
	router.Use(corsMiddleware())

	router.GET("/", h.Page.Index)
	router.GET("/comments", h.Page.Comments)
	router.GET("/messages", h.Page.Messages)
	router.GET("/interaction", h.Page.Interaction)
	router.GET("/trends", h.Page.Trends)
	router.GET("/settings", h.Page.Settings)
	router.GET("/settings/bilibili", h.Page.Bilibili)

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

		api.GET("/settings/app", h.Settings.GetApp)
		api.PUT("/settings/app", h.Settings.SaveApp)
		api.GET("/settings/bilibili", h.Settings.GetBilibili)
		api.PUT("/settings/bilibili/cookie", h.Settings.SaveBilibiliCookie)
		api.POST("/settings/bilibili/qrcode", h.Settings.GenerateBilibiliQRCode)
		api.GET("/settings/bilibili/qrcode/poll", h.Settings.PollBilibiliQRCode)

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
