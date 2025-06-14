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

// EndpointStatsResponse 用于API响应的结构体，使用PascalCase
type EndpointStatsResponse struct {
	TotalCalls    int64  `json:"TotalCalls"`
	TodayCalls    int64  `json:"TodayCalls"`
	LastResetDate string `json:"LastResetDate"`
}

type StatsManager struct {
	Stats           map[string]*EndpointStats `json:"stats"`
	mu              sync.RWMutex
	filepath        string
	isDirty         bool
	lastSaveTime    time.Time
	saveInterval    time.Duration
	minSaveInterval time.Duration
	shutdown        chan struct{}
	wg              sync.WaitGroup // 添加 WaitGroup 用于优雅关闭
}

func NewStatsManager(filepath string) *StatsManager {
	sm := &StatsManager{
		Stats:           make(map[string]*EndpointStats),
		filepath:        filepath,
		saveInterval:    3 * time.Second,
		minSaveInterval: 1 * time.Second,
		lastSaveTime:    time.Now(),
		shutdown:        make(chan struct{}),
	}

	sm.LoadStats()

	sm.wg.Add(2) // 为两个goroutine添加计数
	go sm.startDailyReset()
	go sm.periodicSave()

	return sm
}

func (sm *StatsManager) periodicSave() {
	defer sm.wg.Done()
	ticker := time.NewTicker(sm.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.mu.Lock()
			if sm.isDirty && time.Since(sm.lastSaveTime) >= sm.minSaveInterval {
				sm.saveStatsLocked()
				sm.isDirty = false
				sm.lastSaveTime = time.Now()
			}
			sm.mu.Unlock()

		case <-sm.shutdown:
			sm.mu.Lock()
			if sm.isDirty {
				sm.saveStatsLocked()
				sm.lastSaveTime = time.Now()
			}
			sm.mu.Unlock()
			return
		}
	}
}

func (sm *StatsManager) startDailyReset() {
	defer sm.wg.Done()
	for {
		now := time.Now()
		next := now.Add(24 * time.Hour)
		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
		duration := next.Sub(now)

		select {
		case <-time.After(duration):
			sm.mu.Lock()
			for _, stats := range sm.Stats {
				stats.TodayCalls = 0
				stats.LastResetDate = time.Now().Format("2006-01-02")
			}
			sm.isDirty = true
			sm.mu.Unlock()
		case <-sm.shutdown:
			return
		}
	}
}

// 优雅关闭
func (sm *StatsManager) Shutdown() {
	close(sm.shutdown) // 通知所有goroutine关闭
	sm.wg.Wait()       // 等待所有goroutine完成

	// 最后一次保存
	sm.mu.Lock()
	if sm.isDirty {
		sm.saveStatsLocked()
	}
	sm.mu.Unlock()
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
	sm.isDirty = true
}

func (sm *StatsManager) saveStatsLocked() error {
	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.filepath, data, 0644)
}

func (sm *StatsManager) ForceSave() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	err := sm.saveStatsLocked()
	if err == nil {
		sm.isDirty = false
		sm.lastSaveTime = time.Now()
	}
	return err
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

func (sm *StatsManager) GetStats() map[string]*EndpointStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	statsCopy := make(map[string]*EndpointStats)
	for k, v := range sm.Stats {
		statsCopy[k] = &EndpointStats{
			TotalCalls:    v.TotalCalls,
			TodayCalls:    v.TodayCalls,
			LastResetDate: v.LastResetDate,
		}
	}

	return statsCopy
}

func (sm *StatsManager) GetStatsForAPI() map[string]*EndpointStatsResponse {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	statsCopy := make(map[string]*EndpointStatsResponse)
	for k, v := range sm.Stats {
		statsCopy[k] = &EndpointStatsResponse{
			TotalCalls:    v.TotalCalls,
			TodayCalls:    v.TodayCalls,
			LastResetDate: v.LastResetDate,
		}
	}

	return statsCopy
}

func (sm *StatsManager) LastSaveTime() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lastSaveTime
}
