package stats

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type EndpointStats struct {
	TotalCalls    int64  `json:"total_calls"`
	TodayCalls    int64  `json:"today_calls"`
	LastResetDate string `json:"last_reset_date"`
}

type StatsManager struct {
	Stats    map[string]*EndpointStats `json:"stats"`
	mu       sync.RWMutex
	filepath string
}

func NewStatsManager(filepath string) *StatsManager {
	sm := &StatsManager{
		Stats:    make(map[string]*EndpointStats),
		filepath: filepath,
	}
	sm.LoadStats()
	go sm.startDailyReset()
	return sm
}

func (sm *StatsManager) IncrementCalls(endpoint string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.Stats[endpoint]; !exists {
		sm.Stats[endpoint] = &EndpointStats{
			LastResetDate: time.Now().Format("2006-01-02"),
		}
	}

	sm.Stats[endpoint].TotalCalls++
	sm.Stats[endpoint].TodayCalls++

	// 异步保存统计数据
	go sm.SaveStats()
}

func (sm *StatsManager) GetStats() map[string]*EndpointStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.Stats
}

func (sm *StatsManager) SaveStats() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.filepath, data, 0644)
}

func (sm *StatsManager) LoadStats() error {
	data, err := os.ReadFile(sm.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, sm)
}

func (sm *StatsManager) startDailyReset() {
	for {
		now := time.Now()
		next := now.Add(24 * time.Hour)
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		duration := next.Sub(now)

		time.Sleep(duration)

		sm.mu.Lock()
		for _, stats := range sm.Stats {
			stats.TodayCalls = 0
			stats.LastResetDate = time.Now().Format("2006-01-02")
		}
		sm.mu.Unlock()

		sm.SaveStats()
	}
}
