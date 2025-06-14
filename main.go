package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"random-api-go/config"
	"random-api-go/database"
	"random-api-go/handlers"
	"random-api-go/logging"
	"random-api-go/router"
	"random-api-go/services"
	"random-api-go/stats"
	"syscall"
	"time"
)

type App struct {
	server        *http.Server
	router        *router.Router
	Stats         *stats.StatsManager
	adminHandler  *handlers.AdminHandler
	staticHandler *handlers.StaticHandler
}

func NewApp() *App {
	return &App{
		router: router.New(),
	}
}

func (a *App) Initialize() error {
	// 先加载配置
	if err := config.Load(); err != nil {
		return err
	}

	// 初始化随机数生成器
	source := rand.NewSource(time.Now().UnixNano())
	config.InitRNG(rand.New(source))

	// 然后创建必要的目录
	if err := os.MkdirAll(config.Get().Storage.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// 初始化数据库
	if err := database.Initialize(config.Get().Storage.DataDir); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// 初始化日志
	logging.SetupLogging()

	// 初始化统计管理器
	statsFile := config.Get().Storage.DataDir + "/stats.json"
	a.Stats = stats.NewStatsManager(statsFile)

	// 初始化端点服务
	services.GetEndpointService()

	// 创建管理后台处理器
	a.adminHandler = handlers.NewAdminHandler()

	// 创建静态文件处理器
	staticDir := "./web/out"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("Warning: Static directory %s does not exist, static file serving will be disabled", staticDir)
	} else {
		absStaticDir, err := filepath.Abs(staticDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for static directory: %w", err)
		}
		a.staticHandler = handlers.NewStaticHandler(absStaticDir)
		log.Printf("Static file serving enabled from: %s", absStaticDir)
	}

	// 创建 handlers
	handlers := &handlers.Handlers{
		Stats: a.Stats,
	}

	// 设置路由
	a.router.Setup(handlers)
	a.router.SetupAdminRoutes(a.adminHandler)

	// 设置静态文件路由（如果静态文件处理器存在）
	if a.staticHandler != nil {
		a.router.SetupStaticRoutes(a.staticHandler)
	}

	// 创建 HTTP 服务器
	cfg := config.Get().Server
	a.server = &http.Server{
		Addr:           cfg.Port,
		Handler:        a.router,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	return nil
}

func (a *App) Run() error {
	// 启动服务器
	go func() {
		log.Printf("Server starting on %s...\n", a.server.Addr)
		if a.staticHandler != nil {
			log.Printf("Frontend available at: http://localhost%s", a.server.Addr)
			log.Printf("Admin panel available at: http://localhost%s/admin", a.server.Addr)
		}
		if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 优雅关闭
	return a.gracefulShutdown()
}

func (a *App) gracefulShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a.Stats.Shutdown()

	// 关闭数据库连接
	if err := database.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server shutdown completed")
	return nil
}

func main() {
	app := NewApp()
	if err := app.Initialize(); err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
