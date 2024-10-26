package config

import (
	"math/rand"
	"time"
)

const (
	Port           = ":5003"
	RequestTimeout = 10 * time.Second
	EnvBaseURL     = "BASE_URL"
)

var (
	RNG *rand.Rand
)

func InitRNG(r *rand.Rand) {
	RNG = r
}
