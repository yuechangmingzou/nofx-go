package metrics

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/utils"
)

// Metrics 性能指标
type Metrics struct {
	mu sync.RWMutex

	// HTTP请求指标
	HTTPRequestsTotal    int64
	HTTPRequestsSuccess  int64
	HTTPRequestsError    int64
	HTTPRequestLatency   []time.Duration // 最近100个请求的延迟
	HTTPRequestsByPath   map[string]int64
	HTTPRequestsByStatus map[int]int64

	// WebSocket指标
	WebSocketConnections    int64
	WebSocketMessagesSent   int64
	WebSocketMessagesFailed int64

	// 系统指标
	GoroutineCount int
	MemoryAlloc    uint64
	MemorySys      uint64
	NumGC          uint32

	// 业务指标
	SignalsProcessed    int64
	SignalsSuccess      int64
	SignalsFailed       int64
	OrdersPlaced        int64
	OrdersFailed        int64
	AIRequestsTotal     int64
	AIRequestsSuccess   int64
	AIRequestsFailed    int64
	AILatency           []time.Duration

	// 时间戳
	LastUpdate time.Time
}

var globalMetrics = &Metrics{
	HTTPRequestsByPath:   make(map[string]int64),
	HTTPRequestsByStatus: make(map[int]int64),
	HTTPRequestLatency:   make([]time.Duration, 0, 100),
	AILatency:            make([]time.Duration, 0, 100),
}

// GetMetrics 获取当前指标
func GetMetrics() *Metrics {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	// 更新系统指标
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	globalMetrics.GoroutineCount = runtime.NumGoroutine()
	globalMetrics.MemoryAlloc = m.Alloc
	globalMetrics.MemorySys = m.Sys
	globalMetrics.NumGC = m.NumGC
	globalMetrics.LastUpdate = time.Now()

	// 返回副本
	metrics := *globalMetrics
	metrics.HTTPRequestsByPath = make(map[string]int64)
	metrics.HTTPRequestsByStatus = make(map[int]int64)
	for k, v := range globalMetrics.HTTPRequestsByPath {
		metrics.HTTPRequestsByPath[k] = v
	}
	for k, v := range globalMetrics.HTTPRequestsByStatus {
		metrics.HTTPRequestsByStatus[k] = v
	}

	return &metrics
}

// RecordHTTPRequest 记录HTTP请求
func RecordHTTPRequest(path string, status int, latency time.Duration) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.HTTPRequestsTotal++
	if status >= 200 && status < 400 {
		globalMetrics.HTTPRequestsSuccess++
	} else {
		globalMetrics.HTTPRequestsError++
	}

	globalMetrics.HTTPRequestsByPath[path]++
	globalMetrics.HTTPRequestsByStatus[status]++

	// 保留最近100个延迟记录
	if len(globalMetrics.HTTPRequestLatency) >= 100 {
		globalMetrics.HTTPRequestLatency = globalMetrics.HTTPRequestLatency[1:]
	}
	globalMetrics.HTTPRequestLatency = append(globalMetrics.HTTPRequestLatency, latency)
}

// RecordWebSocketMessage 记录WebSocket消息
func RecordWebSocketMessage(success bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	if success {
		globalMetrics.WebSocketMessagesSent++
	} else {
		globalMetrics.WebSocketMessagesFailed++
	}
}

// RecordWebSocketConnection 记录WebSocket连接
func RecordWebSocketConnection(connected bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	if connected {
		globalMetrics.WebSocketConnections++
	}
}

// RecordSignal 记录信号处理
func RecordSignal(success bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.SignalsProcessed++
	if success {
		globalMetrics.SignalsSuccess++
	} else {
		globalMetrics.SignalsFailed++
	}
}

// RecordOrder 记录订单
func RecordOrder(success bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	if success {
		globalMetrics.OrdersPlaced++
	} else {
		globalMetrics.OrdersFailed++
	}
}

// RecordAIRequest 记录AI请求
func RecordAIRequest(success bool, latency time.Duration) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.AIRequestsTotal++
	if success {
		globalMetrics.AIRequestsSuccess++
	} else {
		globalMetrics.AIRequestsFailed++
	}

	// 保留最近100个延迟记录
	if len(globalMetrics.AILatency) >= 100 {
		globalMetrics.AILatency = globalMetrics.AILatency[1:]
	}
	globalMetrics.AILatency = append(globalMetrics.AILatency, latency)
}

// SaveToRedis 保存指标到Redis
func SaveToRedis(ctx context.Context) error {
	metrics := GetMetrics()
	redis := utils.GetRedisClient()
	logger := utils.GetLogger("metrics")

	// 计算统计信息
	avgHTTPLatency := calculateAvgLatency(metrics.HTTPRequestLatency)
	avgAILatency := calculateAvgLatency(metrics.AILatency)

	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"http": map[string]interface{}{
			"requests_total":   metrics.HTTPRequestsTotal,
			"requests_success": metrics.HTTPRequestsSuccess,
			"requests_error":   metrics.HTTPRequestsError,
			"avg_latency_ms":   avgHTTPLatency.Milliseconds(),
			"by_path":          metrics.HTTPRequestsByPath,
			"by_status":        metrics.HTTPRequestsByStatus,
		},
		"websocket": map[string]interface{}{
			"connections":      metrics.WebSocketConnections,
			"messages_sent":    metrics.WebSocketMessagesSent,
			"messages_failed":  metrics.WebSocketMessagesFailed,
		},
		"system": map[string]interface{}{
			"goroutines":   metrics.GoroutineCount,
			"memory_alloc": metrics.MemoryAlloc,
			"memory_sys":   metrics.MemorySys,
			"num_gc":       metrics.NumGC,
		},
		"business": map[string]interface{}{
			"signals_processed": metrics.SignalsProcessed,
			"signals_success":   metrics.SignalsSuccess,
			"signals_failed":    metrics.SignalsFailed,
			"orders_placed":     metrics.OrdersPlaced,
			"orders_failed":     metrics.OrdersFailed,
			"ai_requests_total": metrics.AIRequestsTotal,
			"ai_requests_success": metrics.AIRequestsSuccess,
			"ai_requests_failed":  metrics.AIRequestsFailed,
			"ai_avg_latency_ms":   avgAILatency.Milliseconds(),
		},
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := config.GetRedisKey("metrics:performance")
	ttl := time.Duration(config.Get().MetricsGlobalTTLSec) * time.Second
	if ttl <= 0 {
		ttl = 300 * time.Second // 默认5分钟
	}

	if err := redis.Set(ctx, key, dataJSON, ttl).Err(); err != nil {
		logger.Warnw("保存性能指标失败", "error", err)
		return err
	}

	return nil
}

// calculateAvgLatency 计算平均延迟
func calculateAvgLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}

	return sum / time.Duration(len(latencies))
}

// StartCollector 启动指标收集器
func StartCollector(ctx context.Context) {
	logger := utils.GetLogger("metrics")
	cfg := config.Get()

	interval := time.Duration(cfg.MetricsGlobalRefreshSec) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second // 默认60秒
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("性能指标收集器启动", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("性能指标收集器停止")
			return
		case <-ticker.C:
			if err := SaveToRedis(ctx); err != nil {
				logger.Warnw("保存指标失败", "error", err)
			}
		}
	}
}

