package config

import (
	"encoding/json"
	"math/rand"
	"os"
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
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}

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
