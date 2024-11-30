package monitoring

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	RequestCount   atomic.Int64 `json:"request_count"`
	AverageLatency float64      `json:"average_latency"`

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

	// 添加性能监控指标
	GCStats struct {
		NumGC      uint32  `json:"num_gc"`
		PauseTotal float64 `json:"pause_total"`
		PauseAvg   float64 `json:"pause_avg"`
	} `json:"gc_stats"`

	CPUUsage    float64 `json:"cpu_usage"`
	ThreadCount int     `json:"thread_count"`
}

type RequestLog struct {
	Time       int64   `json:"time"`   // 使用 Unix 时间戳
	Path       string  `json:"path"`   // 考虑使用字符串池
	Method     string  `json:"method"` // 使用常量池
	StatusCode int     `json:"status_code"`
	Latency    float64 `json:"latency"` // 改回 float64，保持一致性
	IP         string  `json:"ip"`
	Referer    string  `json:"referer"`
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

func formatLatency(microseconds float64) string {
	if microseconds < 1000 {
		return fmt.Sprintf("%.2fµs", microseconds)
	}
	if microseconds < 1000000 {
		return fmt.Sprintf("%.2fms", microseconds/1000)
	}
	return fmt.Sprintf("%.2fs", microseconds/1000000)
}

func LogRequest(log RequestLog) {
	metrics.RequestCount.Add(1)

	// 使用分段锁减少锁竞争
	bucket := getBucket(log.Path)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	metrics.StatusCodes[log.StatusCode]++

	// 处理 referer，只保留域名
	if log.Referer != "direct" {
		if parsedURL, err := url.Parse(log.Referer); err == nil {
			metrics.TopReferers[parsedURL.Host]++
		}
	} else {
		metrics.TopReferers["direct"]++
	}

	// 只记录 API 请求
	if strings.HasPrefix(log.Path, "/pic/") || strings.HasPrefix(log.Path, "/video/") {
		// 更新路径延迟
		if existing, ok := metrics.PathLatencies[log.Path]; ok {
			metrics.PathLatencies[log.Path] = (existing + log.Latency) / 2
		} else {
			metrics.PathLatencies[log.Path] = log.Latency
		}

		// 保存最近请求记录，插入到开头
		metrics.RecentRequests = append([]RequestLog{log}, metrics.RecentRequests...)
		if len(metrics.RecentRequests) > 100 {
			metrics.RecentRequests = metrics.RecentRequests[:100]
		}
	}
}

// 添加字符串池
var stringPool = sync.Pool{
	New: func() interface{} {
		return new(string)
	},
}

// 添加分段锁结构
type bucket struct {
	mu sync.Mutex
}

var buckets = make([]bucket, 32)

func getBucket(path string) *bucket {
	hash := uint32(0)
	for i := 0; i < len(path); i++ {
		hash = hash*31 + uint32(path[i])
	}
	return &buckets[hash%32]
}
