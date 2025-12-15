package execution

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yuechangmingzou/nofx-go/internal/config"
)

// saveAudit 保存审计日志
func (e *ExecutionEngine) saveAudit(ctx context.Context, event map[string]interface{}) {
	cfg := config.Get()
	key := config.GetRedisKey("order_audit")

	// 限制事件大小
	maxChars := cfg.OrderAuditEventMaxChars
	if maxChars <= 0 {
		maxChars = 2000
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}

	eventStr := string(eventJSON)
	if len(eventStr) > maxChars {
		eventStr = eventStr[:maxChars] + "...[已截断]"
	}

	// 添加到列表
	e.redis.LPush(ctx, key, eventStr)
	maxLen := cfg.OrderAuditMaxLen
	if maxLen <= 0 {
		maxLen = 2000
	}
	e.redis.LTrim(ctx, key, 0, int64(maxLen-1))
}

// pushTradeHistory 推送交易历史
func (e *ExecutionEngine) pushTradeHistory(ctx context.Context, event map[string]interface{}) {
	cfg := config.Get()
	key := config.GetRedisKey("trade_history")

	event["ts"] = time.Now().Unix()
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}

	e.redis.LPush(ctx, key, string(eventJSON))
	maxLen := cfg.TradeHistoryMaxLen
	if maxLen <= 0 {
		maxLen = 500
	}
	e.redis.LTrim(ctx, key, 0, int64(maxLen-1))
}

