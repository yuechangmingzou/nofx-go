package config

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

// PerformanceOptimizer 性能优化器
type PerformanceOptimizer struct {
	redis *RedisAdapter
}

var globalOptimizer *PerformanceOptimizer

// GetOptimizer 获取优化器实例
func GetOptimizer() *PerformanceOptimizer {
	if globalOptimizer == nil {
		globalOptimizer = &PerformanceOptimizer{
			redis: NewRedisAdapter(),
		}
	}
	return globalOptimizer
}

// GetRedisAdapter 获取Redis适配器
func (o *PerformanceOptimizer) GetRedisAdapter() (*RedisAdapter, bool) {
	return o.redis, o.redis != nil
}

// OptimizeConfig 根据性能指标优化配置
func (o *PerformanceOptimizer) OptimizeConfig(ctx context.Context) error {
	logger := zap.S().Named("optimizer")
	cfg := Get()

	// 从Redis读取性能指标
	key := GetRedisKey("metrics:performance")
	raw, err := o.redis.Get(ctx, key).Result()
	if err != nil {
		logger.Debugw("未找到性能指标，跳过优化", "error", err)
		return nil
	}

	var metrics map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &metrics); err != nil {
		logger.Warnw("解析性能指标失败", "error", err)
		return err
	}

	recommendations := make(map[string]interface{})

	// 分析HTTP指标
	if httpData, ok := metrics["http"].(map[string]interface{}); ok {
		avgLatencyMs, _ := httpData["avg_latency_ms"].(float64)
		errorRate := 0.0
		if total, _ := httpData["requests_total"].(float64); total > 0 {
			errors, _ := httpData["requests_error"].(float64)
			errorRate = errors / total
		}

		// 如果平均延迟高，建议增加缓存时间
		if avgLatencyMs > 200 {
			currentTTL := float64(cfg.WebStatusCacheTTLSec)
			if currentTTL < 30 {
				recommendations["web_status_cache_ttl_sec"] = 30.0
				logger.Infow("建议增加状态缓存时间",
					"当前", currentTTL,
					"建议", 30.0,
					"原因", "平均延迟较高",
				)
			}
		}

		// 如果错误率高，建议增加超时时间
		if errorRate > 0.05 {
			currentTimeout := cfg.BinanceHTTPTimeoutSec
			if currentTimeout < 30 {
				recommendations["binance_http_timeout_sec"] = 30.0
				logger.Infow("建议增加HTTP超时时间",
					"当前", currentTimeout,
					"建议", 30.0,
					"原因", "错误率较高",
				)
			}
		}
	}

	// 分析系统指标
	if sysData, ok := metrics["system"].(map[string]interface{}); ok {
		goroutines, _ := sysData["goroutines"].(float64)
		memoryAlloc, _ := sysData["memory_alloc"].(float64)

		// 如果goroutine过多，建议减少并发
		if goroutines > 1000 {
			currentConcurrency := cfg.ScanConcurrency
			if currentConcurrency > 5 {
				recommendations["scan_concurrency"] = 5
				logger.Infow("建议减少扫描并发数",
					"当前", currentConcurrency,
					"建议", 5,
					"原因", "goroutine数量过多",
				)
			}
		}

		// 如果内存使用高，建议减少缓存
		if memoryAlloc > 500*1024*1024 { // 500MB
			currentCacheTTL := float64(cfg.MarketSnapshotTTLSec)
			if currentCacheTTL > 60 {
				recommendations["market_snapshot_ttl_sec"] = 60
				logger.Infow("建议减少市场快照缓存时间",
					"当前", currentCacheTTL,
					"建议", 60,
					"原因", "内存使用较高",
				)
			}
		}
	}

	// 分析业务指标
	if bizData, ok := metrics["business"].(map[string]interface{}); ok {
		aiLatencyMs, _ := bizData["ai_avg_latency_ms"].(float64)
		aiErrorRate := 0.0
		if aiTotal, _ := bizData["ai_requests_total"].(float64); aiTotal > 0 {
			aiErrors, _ := bizData["ai_requests_failed"].(float64)
			aiErrorRate = aiErrors / aiTotal
		}

		// 如果AI延迟高，建议减少批次大小
		if aiLatencyMs > 5000 {
			currentBatchSize := cfg.AIBatchSize
			if currentBatchSize > 1 {
				recommendations["ai_batch_size"] = 1
				logger.Infow("建议减少AI批次大小",
					"当前", currentBatchSize,
					"建议", 1,
					"原因", "AI延迟较高",
				)
			}
		}

		// 如果AI错误率高，建议增加重试或超时
		if aiErrorRate > 0.1 {
			logger.Warnw("AI错误率较高",
				"error_rate", aiErrorRate,
				"建议", "检查AI服务可用性或增加超时时间",
			)
		}
	}

	// 保存优化建议
	if len(recommendations) > 0 {
		recommendationsKey := GetRedisKey("config:recommendations")
		recommendationsData := map[string]interface{}{
			"recommendations": recommendations,
			"timestamp":      time.Now().Unix(),
		}
		recommendationsJSON, _ := json.Marshal(recommendationsData)
		_ = o.redis.Set(ctx, recommendationsKey, recommendationsJSON, 24*time.Hour)
	}

	return nil
}

// StartOptimizer 启动优化器
func StartOptimizer(ctx context.Context) {
	logger := zap.S().Named("optimizer")
	optimizer := GetOptimizer()

	ticker := time.NewTicker(5 * time.Minute) // 每5分钟优化一次
	defer ticker.Stop()

	logger.Info("配置优化器启动")

	for {
		select {
		case <-ctx.Done():
			logger.Info("配置优化器停止")
			return
		case <-ticker.C:
			if err := optimizer.OptimizeConfig(ctx); err != nil {
				logger.Warnw("配置优化失败", "error", err)
			}
		}
	}
}

