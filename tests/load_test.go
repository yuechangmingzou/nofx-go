package tests

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// LoadTestConfig 负载测试配置
type LoadTestConfig struct {
	BaseURL       string
	Concurrency   int
	TotalRequests int
	Duration      time.Duration
	Username      string
	Password      string
}

// LoadTestResult 负载测试结果
type LoadTestResult struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	RequestsPerSec  float64
	Errors          []string
}

// RunLoadTest 运行负载测试
func RunLoadTest(config LoadTestConfig) (*LoadTestResult, error) {
	var (
		totalRequests   int64
		successRequests int64
		failedRequests  int64
		totalLatency    int64
		minLatency      = time.Duration(1 << 30) // 最大值
		maxLatency      time.Duration
		latencies       []time.Duration
		errors          []string
		mu              sync.Mutex
	)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 测试端点列表
	endpoints := []string{
		"/api/status",
		"/api/market-data",
		"/api/balance",
		"/api/positions",
		"/api/scanned-symbols",
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, config.Concurrency)

	// 启动worker
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				sem <- struct{}{}
				atomic.AddInt64(&totalRequests, 1)

				// 随机选择一个端点
				endpoint := endpoints[totalRequests%int64(len(endpoints))]

				reqStart := time.Now()
				req, err := http.NewRequest("GET", config.BaseURL+endpoint, nil)
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Sprintf("创建请求失败: %v", err))
					mu.Unlock()
					atomic.AddInt64(&failedRequests, 1)
					<-sem
					continue
				}

				req.SetBasicAuth(config.Username, config.Password)

				resp, err := client.Do(req)
				latency := time.Since(reqStart)

				mu.Lock()
				if latency < minLatency {
					minLatency = latency
				}
				if latency > maxLatency {
					maxLatency = latency
				}
				latencies = append(latencies, latency)
				totalLatency += int64(latency)
				mu.Unlock()

				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Sprintf("请求失败: %v", err))
					mu.Unlock()
					atomic.AddInt64(&failedRequests, 1)
				} else {
					resp.Body.Close()
					if resp.StatusCode >= 200 && resp.StatusCode < 400 {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
						mu.Lock()
						errors = append(errors, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, endpoint))
						mu.Unlock()
					}
				}

				<-sem

				// 如果达到总请求数，退出
				if atomic.LoadInt64(&totalRequests) >= int64(config.TotalRequests) {
					return
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 计算统计信息
	mu.Lock()
	defer mu.Unlock()

	if len(latencies) == 0 {
		return nil, fmt.Errorf("没有收集到延迟数据")
	}

	// 排序延迟（简化版，使用选择排序）
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	for i := 0; i < len(sortedLatencies)-1; i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	avgLatency := time.Duration(totalLatency) / time.Duration(len(latencies))
	p50 := sortedLatencies[len(sortedLatencies)*50/100]
	p95 := sortedLatencies[len(sortedLatencies)*95/100]
	p99 := sortedLatencies[len(sortedLatencies)*99/100]

	requestsPerSec := float64(totalRequests) / duration.Seconds()

	return &LoadTestResult{
		TotalRequests:   totalRequests,
		SuccessRequests: successRequests,
		FailedRequests:  failedRequests,
		TotalLatency:    time.Duration(totalLatency),
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
		AvgLatency:      avgLatency,
		P50Latency:      p50,
		P95Latency:      p95,
		P99Latency:      p99,
		RequestsPerSec:  requestsPerSec,
		Errors:          errors[:min(10, len(errors))], // 只保留前10个错误
	}, nil
}

// TestLoadTest 负载测试（示例）
func TestLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过负载测试（使用 -short 标志）")
	}

	config := LoadTestConfig{
		BaseURL:       "http://localhost:8000",
		Concurrency:   10,
		TotalRequests: 1000,
		Duration:      60 * time.Second,
		Username:      "admin",
		Password:      "admin",
	}

	result, err := RunLoadTest(config)
	if err != nil {
		t.Fatalf("负载测试失败: %v", err)
	}

	// 输出结果
	t.Logf("负载测试结果:")
	t.Logf("  总请求数: %d", result.TotalRequests)
	t.Logf("  成功请求: %d", result.SuccessRequests)
	t.Logf("  失败请求: %d", result.FailedRequests)
	t.Logf("  成功率: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("  平均延迟: %v", result.AvgLatency)
	t.Logf("  P50延迟: %v", result.P50Latency)
	t.Logf("  P95延迟: %v", result.P95Latency)
	t.Logf("  P99延迟: %v", result.P99Latency)
	t.Logf("  QPS: %.2f", result.RequestsPerSec)

	// 断言
	if result.SuccessRequests < int64(config.TotalRequests)*90/100 {
		t.Errorf("成功率过低: %.2f%%", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	}

	if result.AvgLatency > 500*time.Millisecond {
		t.Errorf("平均延迟过高: %v", result.AvgLatency)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
