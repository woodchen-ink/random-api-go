package model

import (
	"random-api-go/config"
	"sync"
)

type URLSelector struct {
	URLs []string
	mu   sync.Mutex
}

func NewURLSelector(urls []string) *URLSelector {
	return &URLSelector{
		URLs: urls,
	}
}

func (us *URLSelector) GetRandomURL() string {
	us.mu.Lock()
	defer us.mu.Unlock()

	if len(us.URLs) == 0 {
		return ""
	}
	return us.URLs[config.RNG.Intn(len(us.URLs))]
}
