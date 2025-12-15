package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/pkg/types"
)

// checkAndSetDedupe 检查并设置去重标记
func (e *ExecutionEngine) checkAndSetDedupe(ctx context.Context, symbol string, signal *types.Signal, windowSec int) bool {
	cfg := config.Get()

	// 构建去重键（包含symbol、side、price、signal_id）
	dedupeKey := fmt.Sprintf("dedupe:%s:%s:%s:%.8f",
		symbol,
		signal.Action,
		signal.Side,
		signal.EntryPrice,
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

