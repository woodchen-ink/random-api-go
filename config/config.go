package config

import (
	"bufio"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server struct {
		Port           string
		ReadTimeout    time.Duration
		WriteTimeout   time.Duration
		MaxHeaderBytes int
	}

	Storage struct {
		DataDir string
	}

	OAuth struct {
		ClientID     string
		ClientSecret string
	}

	App struct {
		BaseURL string
	}
}

var (
	cfg Config
	RNG *rand.Rand
)

// loadEnvFile 加载.env文件
func loadEnvFile() error {
	file, err := os.Open(".env")
	if err != nil {
		return err // .env文件不存在，这是正常的
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 移除引号
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		// 只有当环境变量不存在时才设置
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// Load 从环境变量加载配置
func Load() error {
	// 首先尝试加载.env文件
	loadEnvFile() // 忽略错误，因为.env文件是可选的

	// 服务器配置
	cfg.Server.Port = getEnv("PORT", ":5003")
	cfg.Server.ReadTimeout = getDurationEnv("READ_TIMEOUT", 30*time.Second)
	cfg.Server.WriteTimeout = getDurationEnv("WRITE_TIMEOUT", 30*time.Second)
	cfg.Server.MaxHeaderBytes = getIntEnv("MAX_HEADER_BYTES", 1<<20)

	// 存储配置
	cfg.Storage.DataDir = getEnv("DATA_DIR", "./data")

	// OAuth配置
	cfg.OAuth.ClientID = getEnv("OAUTH_CLIENT_ID", "")
	cfg.OAuth.ClientSecret = getEnv("OAUTH_CLIENT_SECRET", "")

	// 应用配置
	cfg.App.BaseURL = getEnv("BASE_URL", "http://localhost:5003")

	return nil
}

func Get() *Config {
	return &cfg
}

func InitRNG(r *rand.Rand) {
	RNG = r
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv 获取整数类型的环境变量
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv 获取时间间隔类型的环境变量
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
