package exchange

import (
	"strconv"
	"sync"
	"time"
)

// RateLimiter 令牌桶限流器
type RateLimiter struct {
	rate     float64
	capacity float64
	tokens   float64
	lastUpdate time.Time
	mu       sync.Mutex
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		capacity:   float64(capacity),
		tokens:     float64(capacity),
		lastUpdate: time.Now(),
	}
}

// Acquire 尝试获取令牌
func (rl *RateLimiter) Acquire(tokens int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	newTokens := elapsed * rl.rate

	rl.tokens = min(rl.capacity, rl.tokens+newTokens)
	rl.lastUpdate = now

	if rl.tokens >= float64(tokens) {
		rl.tokens -= float64(tokens)
		return true
	}

	return false
}

// Wait 等待直到可以获取令牌
func (rl *RateLimiter) Wait(tokens int) {
	for !rl.Acquire(tokens) {
		waitTime := (float64(tokens) - rl.tokens) / rl.rate
		if waitTime < 0 {
			waitTime = 0.1
		}
		if waitTime > 1.0 {
			waitTime = 1.0
		}
		time.Sleep(time.Duration(waitTime * float64(time.Second)))
	}
}

// BackoffManager 退避管理器（处理429/418限流）
type BackoffManager struct {
	backoffUntil map[string]time.Time
	backoffLevel map[string]int
	mu           sync.RWMutex
	maxLevel     int
	maxSec       float64
}

var globalBackoff = &BackoffManager{
	backoffUntil: make(map[string]time.Time),
	backoffLevel: make(map[string]int),
	maxLevel:     6,
	maxSec:       60.0,
}

// GetGlobalBackoff 获取全局退避管理器（供测试使用）
func GetGlobalBackoff() *BackoffManager {
	return globalBackoff
}

// WaitBackoff 等待退避窗口
func (bm *BackoffManager) WaitBackoff(endpoint string) {
	for {
		bm.mu.RLock()
		until, exists := bm.backoffUntil[endpoint]
		bm.mu.RUnlock()

		if !exists || time.Now().After(until) {
			return
		}

		wait := time.Until(until)
		if wait > time.Second {
			wait = time.Second
		}
		time.Sleep(wait)
	}
}

// SetBackoff 设置退避窗口
func (bm *BackoffManager) SetBackoff(endpoint string, waitSec float64) {
	if waitSec <= 0 {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	until := time.Now().Add(time.Duration(waitSec * float64(time.Second)))
	cur, exists := bm.backoffUntil[endpoint]
	if !exists || until.After(cur) {
		bm.backoffUntil[endpoint] = until
	}
}

// ResetBackoff 重置退避
func (bm *BackoffManager) ResetBackoff(endpoint string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.backoffUntil, endpoint)
	bm.backoffLevel[endpoint] = 0
}

// OnRateLimited 处理限流，返回建议等待时间
func (bm *BackoffManager) OnRateLimited(endpoint string, status int, retryAfter *float64) float64 {
	bm.mu.Lock()
	level := bm.backoffLevel[endpoint]
	bm.backoffLevel[endpoint] = min(bm.maxLevel, level+1)
	bm.mu.Unlock()

	var waitSec float64
	if retryAfter != nil {
		waitSec = *retryAfter
	} else {
		// 418通常表示更严格的限流
		base := 60.0
		if status != 418 {
			base = 1.0
		}
		exp := min(level, bm.maxLevel)
		multiplier := 1.0
		for i := 0; i < exp; i++ {
			multiplier *= 2.0
		}
		waitSec = min(base*multiplier, bm.maxSec)
	}

	// 添加抖动（避免多协程同时恢复）
	waitSec = max(1.0, min(waitSec, bm.maxSec))
	jitter := 0.1 * waitSec
	if jitter > 1.0 {
		jitter = 1.0
	}
	waitSec += jitter

	bm.SetBackoff(endpoint, waitSec)
	return waitSec
}

// ParseRetryAfter 解析Retry-After头
func ParseRetryAfter(value string) *float64 {
	if value == "" {
		return nil
	}

	// 尝试解析为秒数
	if sec, err := strconv.ParseFloat(value, 64); err == nil {
		if sec >= 0 {
			return &sec
		}
	}

	// 尝试解析为HTTP日期
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		wait := time.Until(t).Seconds()
		if wait >= 0 {
			return &wait
		}
	}

	return nil
}

