package monitoring

import (
	"runtime"
	"strings"
	"sync"
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

	// 热门引用来源
	TopReferers sync.Map `json:"-"` // 使用 sync.Map 足够了
}

type RequestLog struct {
	Time       int64   `json:"time"`
	Path       string  `json:"path"`
	Method     string  `json:"method"`
	StatusCode int     `json:"status_code"`
	Latency    float64 `json:"latency"`
	IP         string  `json:"ip"`
	Referer    string  `json:"referer"`
}

var (
	metrics   SystemMetrics
	startTime = time.Now()
)

func init() {
	// 定期清理引用来源
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			metrics.TopReferers = sync.Map{} // 直接重置
		}
	}()
}

func LogRequest(log RequestLog) {
	// 更新引用来源
	if log.Referer != "direct" {
		metrics.TopReferers.Store(log.Referer, 1)
	} else {
		metrics.TopReferers.Store("直接访问", 1)
	}

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
