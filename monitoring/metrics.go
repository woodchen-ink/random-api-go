package monitoring

import (
	"runtime"
	"strings"
	"time"
)

type SystemMetrics struct {
	// 基础指标
	Uptime    time.Duration `json:"uptime"`
	StartTime time.Time     `json:"start_time"`

	// 系统指标
	NumCPU         int     `json:"num_cpu"`
	NumGoroutine   int     `json:"num_goroutine"`
	AverageLatency float64 `json:"average_latency"`
	MemoryStats    struct {
		HeapAlloc uint64 `json:"heap_alloc"`
		HeapSys   uint64 `json:"heap_sys"`
	} `json:"memory_stats"`
}

type RequestLog struct {
	Time       int64   `json:"time"`
	Path       string  `json:"path"`
	Method     string  `json:"method"`
	StatusCode int     `json:"status_code"`
	Latency    float64 `json:"latency"`
	IP         string  `json:"ip"`
	Referer    string  `json:"-"`
}

var (
	metrics   SystemMetrics
	startTime = time.Now()
)

func LogRequest(log RequestLog) {
	// 更新平均延迟 (只关心 API 请求)
	if strings.HasPrefix(log.Path, "/pic/") || strings.HasPrefix(log.Path, "/video/") {
		metrics.AverageLatency = (metrics.AverageLatency + log.Latency) / 2
	}
}

func CollectMetrics() *SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics.Uptime = time.Since(startTime)
	metrics.StartTime = startTime
	metrics.NumCPU = runtime.NumCPU()
	metrics.NumGoroutine = runtime.NumGoroutine()
	metrics.MemoryStats.HeapAlloc = m.HeapAlloc
	metrics.MemoryStats.HeapSys = m.HeapSys

	return &metrics
}
