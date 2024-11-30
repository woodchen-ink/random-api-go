package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"random-api-go/config"
	"random-api-go/handlers"
	"random-api-go/logging"
	"random-api-go/router"
	"random-api-go/services"
	"random-api-go/stats"
	"syscall"
	"time"
)

type App struct {
	server *http.Server
	router *router.Router
	Stats  *stats.StatsManager
}

func NewApp() *App {
	return &App{
		router: router.New(),
	}
}

func (a *App) Initialize() error {
	// 先加载配置
	if err := config.Load("/root/data/config.json"); err != nil {
		return err
	}

	// 初始化随机数生成器
	source := rand.NewSource(time.Now().UnixNano())
	config.InitRNG(rand.New(source))

	// 然后创建必要的目录
	if err := os.MkdirAll(config.Get().Storage.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// 初始化日志
	logging.SetupLogging()

	// 初始化统计管理器
	a.Stats = stats.NewStatsManager(config.Get().Storage.StatsFile)

	// 初始化服务
	if err := services.InitializeCSVService(); err != nil {
		return err
	}

	// 创建 handlers
	handlers := &handlers.Handlers{
		Stats: a.Stats,
	}

	// 设置路由
	a.router.Setup(handlers)

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
