package monitoring

import (
	"runtime"
	"sync"
	"time"
)

type SystemMetrics struct {
	// 基础指标
	Uptime    time.Duration `json:"uptime"`
	StartTime time.Time     `json:"start_time"`

	// 系统指标
	NumCPU       int `json:"num_cpu"`
	NumGoroutine int `json:"num_goroutine"`
	MemoryStats  struct {
		Alloc      uint64 `json:"alloc"`
		TotalAlloc uint64 `json:"total_alloc"`
		Sys        uint64 `json:"sys"`
		HeapAlloc  uint64 `json:"heap_alloc"`
		HeapSys    uint64 `json:"heap_sys"`
	} `json:"memory_stats"`

	// 性能指标
	RequestCount   int64   `json:"request_count"`
	AverageLatency float64 `json:"average_latency"`

	// 流量统计
	TotalBytesIn  int64 `json:"total_bytes_in"`
	TotalBytesOut int64 `json:"total_bytes_out"`

	// 状态码统计
	StatusCodes map[int]int64 `json:"status_codes"`

	// 路径延迟统计
	PathLatencies map[string]float64 `json:"path_latencies"`

	// 最近请求
	RecentRequests []RequestLog `json:"recent_requests"`

	// 热门引用来源
	TopReferers map[string]int64 `json:"top_referers"`
}

type RequestLog struct {
	Time       time.Time `json:"time"`
	Path       string    `json:"path"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	Latency    float64   `json:"latency"`
	IP         string    `json:"ip"`
	Referer    string    `json:"referer"`
}

var (
	metrics   SystemMetrics
	mu        sync.RWMutex
	startTime = time.Now()
)

func init() {
	metrics.StatusCodes = make(map[int]int64)
	metrics.PathLatencies = make(map[string]float64)
	metrics.TopReferers = make(map[string]int64)
	metrics.RecentRequests = make([]RequestLog, 0, 100)
}

func CollectMetrics() *SystemMetrics {
	mu.Lock()
	defer mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics.Uptime = time.Since(startTime)
	metrics.StartTime = startTime
	metrics.NumCPU = runtime.NumCPU()
	metrics.NumGoroutine = runtime.NumGoroutine()

	metrics.MemoryStats.Alloc = m.Alloc
	metrics.MemoryStats.TotalAlloc = m.TotalAlloc
	metrics.MemoryStats.Sys = m.Sys
	metrics.MemoryStats.HeapAlloc = m.HeapAlloc
	metrics.MemoryStats.HeapSys = m.HeapSys

	return &metrics
}

func LogRequest(log RequestLog) {
	mu.Lock()
	defer mu.Unlock()

	metrics.RequestCount++
	metrics.StatusCodes[log.StatusCode]++
	metrics.TopReferers[log.Referer]++

	// 更新路径延迟
	if existing, ok := metrics.PathLatencies[log.Path]; ok {
		metrics.PathLatencies[log.Path] = (existing + log.Latency) / 2
	} else {
		metrics.PathLatencies[log.Path] = log.Latency
	}

	// 保存最近请求记录
	metrics.RecentRequests = append(metrics.RecentRequests, log)
	if len(metrics.RecentRequests) > 100 {
		metrics.RecentRequests = metrics.RecentRequests[1:]
	}
}
