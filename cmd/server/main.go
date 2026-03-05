package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http" // 新增导入
	"os"
	"path/filepath"
	"slices"
	"time"

	"bilibili-up-admin/config"
	"bilibili-up-admin/internal/handler"
	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/polling"
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
	settingsSvc := service.NewAppSettingsService(repos.Setting, repos.LLMProvider)
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
	pollingManager := initPolling(runtimeStore, services)
	handlers := initHandlers(services, settingsSvc, runtimeStore, pollingManager)
	router := initRouter(handlers, config.GlobalConfig.Server.Mode)

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	localURL := fmt.Sprintf("http://localhost%s/admin/", addr)

	// 启动时在控制台打印运行网址
	fmt.Printf("\n========================================================\n")
	fmt.Printf("🚀 服务启动成功! 请在浏览器中访问: %s\n", localURL)
	fmt.Printf("========================================================\n\n")

	zapLogger.Info("server starting", zap.String("addr", addr), zap.String("url", localURL))
	pollingManager.SetLogger(func(format string, args ...any) {
		zapLogger.Sugar().Infof(format, args...)
	})
	if err := pollingManager.Start(context.Background()); err != nil {
		zapLogger.Fatal("start polling manager failed", zap.Error(err))
	}
	defer pollingManager.Stop(context.Background())

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

	if err := configureSQLiteConcurrency(db, dbConfig.Driver); err != nil {
		return nil, err
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
		&model.LLMProvider{}, // 新增这一行
	); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	return db, nil
}

func configureSQLiteConcurrency(db *gorm.DB, driver string) error {
	if driver != "" && driver != "sqlite" {
		return nil
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA wal_autocheckpoint=1000;",
		"PRAGMA temp_store=MEMORY;",
	}
	for _, stmt := range pragmas {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply sqlite pragma failed (%s): %w", stmt, err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql db failed: %w", err)
	}
	sqlDB.SetMaxOpenConns(64)
	sqlDB.SetMaxIdleConns(32)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return nil
}

func initPolling(runtime *appruntime.Store, services *Services) *polling.Manager {
	mgr := polling.NewManager()

	checkReady := func(_ context.Context) error {
		if runtime == nil || runtime.BilibiliClient() == nil {
			return polling.ErrSkipTask
		}
		return nil
	}

	postHandle := func(_ context.Context, runErr error) error {
		return runErr
	}

	_ = mgr.Register(polling.Task{
		Name:       "trend-taginfo-sync",
		Interval:   15 * time.Minute,
		Timeout:    180 * time.Second,
		RunOnStart: true,
		PreHandle:  checkReady,
		Handle: func(ctx context.Context) error {
			_, err := services.Trend.SyncTagInfoHotValues(ctx, 50)
			return err
		},
		PostHandle: postHandle,
	})

	_ = mgr.Register(polling.Task{
		Name:       "video-comments-sync",
		Interval:   5 * time.Minute,
		Timeout:    90 * time.Second,
		RunOnStart: true,
		PreHandle:  checkReady,
		Handle: func(ctx context.Context) error {
			_, err := services.Comment.SyncRecentVideoComments(ctx, 3, 1, 20)
			return err
		},
		PostHandle: postHandle,
	})

	_ = mgr.Register(polling.Task{
		Name:       "private-messages-sync",
		Interval:   1 * time.Minute,
		Timeout:    90 * time.Second,
		RunOnStart: true,
		PreHandle:  checkReady,
		Handle: func(ctx context.Context) error {
			_, err := services.Message.SyncMessages(ctx, 1, 20)
			return err
		},
		PostHandle: postHandle,
	})

	_ = mgr.Register(polling.Task{
		Name:       "fans-weekly-interact",
		Interval:   10 * time.Minute,
		Timeout:    180 * time.Second,
		RunOnStart: false,
		PreHandle:  checkReady,
		Handle: func(ctx context.Context) error {
			cfg, err := services.Settings.Load(ctx)
			if err != nil {
				return err
			}
			_, err = services.Interaction.AutoInteractRecentFanVideos(ctx, cfg.Interaction, 20)
			return err
		},
		PostHandle: postHandle,
	})

	return mgr
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
	LLMProvider *repository.LLMProviderRepository // 新增这一行
}

func initRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Comment:     repository.NewCommentRepository(db),
		Message:     repository.NewMessageRepository(db),
		Interaction: repository.NewInteractionRepository(db),
		TagRanking:  repository.NewTagRankingRepository(db),
		LLMChatLog:  repository.NewLLMChatLogRepository(db),
		Setting:     repository.NewSettingRepository(db),
		LLMProvider: repository.NewLLMProviderRepository(db), // 新增这一行
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
	Page          *handler.PageHandler
	Comment       *handler.CommentHandler
	Message       *handler.MessageHandler
	Interaction   *handler.InteractionHandler
	Trend         *handler.TrendHandler
	LLM           *handler.LLMHandler
	Settings      *handler.SettingsHandler
	Observability *handler.ObservabilityHandler
}

func initHandlers(services *Services, settings *service.AppSettingsService, runtime *appruntime.Store, pollingManager *polling.Manager) *Handlers {
	return &Handlers{
		Page:          handler.NewPageHandler(),
		Comment:       handler.NewCommentHandler(services.Comment),
		Message:       handler.NewMessageHandler(services.Message),
		Interaction:   handler.NewInteractionHandler(services.Interaction, settings),
		Trend:         handler.NewTrendHandler(services.Trend),
		LLM:           handler.NewLLMHandler(services.LLM),
		Settings:      handler.NewSettingsHandler(settings, runtime),
		Observability: handler.NewObservabilityHandler(pollingManager),
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
	router.Use(corsMiddleware())

	// 根目录重定向至统一前缀
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/")
	})

	// 统一前缀路由组
	admin := router.Group("/admin")
	{
		admin.Static("/static", "web/static")

		admin.GET("/", h.Page.Index)
		admin.GET("/comments", h.Page.Comments)
		admin.GET("/messages", h.Page.Messages)
		admin.GET("/interaction", h.Page.Interaction)
		admin.GET("/trends", h.Page.Trends)
		admin.GET("/settings", h.Page.Settings)
		admin.GET("/settings/bilibili", h.Page.Bilibili)

		api := admin.Group("/api")
		{
			api.GET("/observability/polling", h.Observability.PollingStats)

			api.GET("/comments", h.Comment.List)
			api.POST("/comments/sync", h.Comment.Sync)
			api.GET("/comments/my-videos", h.Comment.GetMyVideos)
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
			api.GET("/fans/list", h.Interaction.FansList)
			api.GET("/fans/:id/videos", h.Interaction.FanVideos)
			api.GET("/fans/videos", h.Interaction.FansVideos)
			api.GET("/videos/:id/engagement", h.Interaction.SyncVideoEngagement)
			api.POST("/videos/:id/like", h.Interaction.Like)
			api.POST("/videos/:id/coin", h.Interaction.Coin)
			api.POST("/videos/:id/favorite", h.Interaction.Favorite)
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

			// 新增的 大模型 CRUD 独立路由
			api.GET("/settings/llm/channels", h.Settings.GetLLMChannels)
			api.GET("/settings/llm/providers", h.Settings.GetLLMProviders)
			api.POST("/settings/llm/providers", h.Settings.AddLLMProvider)
			api.PUT("/settings/llm/providers/:name", h.Settings.UpdateLLMProvider)
			api.DELETE("/settings/llm/providers/:name", h.Settings.DeleteLLMProvider)

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
