package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

const (
	EnvBaseURL     = "BASE_URL"
	DefaultPort    = ":5003"
	RequestTimeout = 10 * time.Second
)

type Config struct {
	Server struct {
		Port           string        `json:"port"`
		ReadTimeout    time.Duration `json:"read_timeout"`
		WriteTimeout   time.Duration `json:"write_timeout"`
		MaxHeaderBytes int           `json:"max_header_bytes"`
	} `json:"server"`

	Storage struct {
		DataDir   string `json:"data_dir"`
		StatsFile string `json:"stats_file"`
		LogFile   string `json:"log_file"`
	} `json:"storage"`

	API struct {
		BaseURL        string        `json:"base_url"`
		RequestTimeout time.Duration `json:"request_timeout"`
	} `json:"api"`
}

var (
	cfg Config
	RNG *rand.Rand
)

func Load(configFile string) error {
	// 尝试创建配置目录
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 创建默认配置
		defaultConfig := Config{
			Server: struct {
				Port           string        `json:"port"`
				ReadTimeout    time.Duration `json:"read_timeout"`
				WriteTimeout   time.Duration `json:"write_timeout"`
				MaxHeaderBytes int           `json:"max_header_bytes"`
			}{
				Port:           ":5003",
				ReadTimeout:    30 * time.Second,
				WriteTimeout:   30 * time.Second,
				MaxHeaderBytes: 1 << 20,
			},
			Storage: struct {
				DataDir   string `json:"data_dir"`
				StatsFile string `json:"stats_file"`
				LogFile   string `json:"log_file"`
			}{
				DataDir:   "/root/data",
				StatsFile: "/root/data/stats.json",
				LogFile:   "/root/data/logs/server.log",
			},
			API: struct {
				BaseURL        string        `json:"base_url"`
				RequestTimeout time.Duration `json:"request_timeout"`
			}{
				BaseURL:        "",
				RequestTimeout: 10 * time.Second,
			},
		}

		// 将默认配置写入文件
		data, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %w", err)
		}

		if err := os.WriteFile(configFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}

		cfg = defaultConfig
		return nil
	}

	// 读取现有配置文件
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}

	// 如果环境变量设置了 BASE_URL，则覆盖配置文件中的设置
	if envBaseURL := os.Getenv(EnvBaseURL); envBaseURL != "" {
		cfg.API.BaseURL = envBaseURL
	}

	return nil
}

func Get() *Config {
	return &cfg
}

func InitRNG(r *rand.Rand) {
	RNG = r
}
