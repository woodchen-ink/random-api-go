package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"random-api-go/config"
	"random-api-go/handlers"
	"random-api-go/logging"
	"random-api-go/stats"

	"syscall"
	"time"
)

func init() {
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}
}

func main() {
	source := rand.NewSource(time.Now().UnixNano())
	config.InitRNG(rand.New(source))

	logging.SetupLogging()
	statsManager := stats.NewStatsManager("data/stats.json")

	// 设置优雅关闭
	setupGracefulShutdown(statsManager)

	// 初始化handlers
	if err := handlers.InitializeHandlers(statsManager); err != nil {
		log.Fatal("Failed to initialize handlers:", err)
	}

	// 设置路由
	setupRoutes()

	log.Printf("Server starting on %s...\n", config.Port)
	if err := http.ListenAndServe(config.Port, nil); err != nil {
		log.Fatal(err)
	}
}

func setupGracefulShutdown(statsManager *stats.StatsManager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Server is shutting down...")
		statsManager.Shutdown()
		log.Println("Stats manager shutdown completed")
		os.Exit(0)
	}()
}

func setupRoutes() {
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)
	http.HandleFunc("/pic/", handlers.HandleAPIRequest)
	http.HandleFunc("/video/", handlers.HandleAPIRequest)
	http.HandleFunc("/stats", handlers.HandleStats)
}
