package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/pkg/types"
)

// checkAndSetDedupe 检查并设置去重标记
func (e *ExecutionEngine) checkAndSetDedupe(ctx context.Context, symbol string, signal *types.Signal, windowSec int) bool {
	cfg := config.Get()

	// 构建去重键（包含symbol、side、price、action和时间窗口）
	// 使用时间窗口确保相同价格但不同时间的信号不会被误去重
	timeWindow := time.Now().Unix() / int64(windowSec) // 时间窗口
	dedupeKey := fmt.Sprintf("dedupe:%s:%s:%s:%.8f:%d",
		symbol,
		signal.Action,
		signal.Side,
		signal.EntryPrice,
		timeWindow,
	)

	// 检查是否已存在
	exists, err := e.redis.Exists(ctx, dedupeKey).Result()
	if err != nil {
		return true // 出错时允许继续（避免阻塞）
	}

	if exists > 0 {
		return false // 去重命中
	}

	// 设置去重标记
	ttl := time.Duration(windowSec) * time.Second
	if ttl <= 0 {
		ttl = time.Duration(cfg.OrderDedupeWindow) * time.Second
	}

	e.redis.Set(ctx, dedupeKey, "1", ttl)
	return true
}

