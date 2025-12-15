package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yuechangmingzou/nofx-go/internal/ai"
	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/internal/exchange"
	"github.com/yuechangmingzou/nofx-go/internal/execution"
	"github.com/yuechangmingzou/nofx-go/internal/metrics"
	"github.com/yuechangmingzou/nofx-go/internal/strategies"
	"github.com/yuechangmingzou/nofx-go/internal/utils"
	"github.com/yuechangmingzou/nofx-go/pkg/types"
)

// Bot 交易机器人
type Bot struct {
	aiTrader         *ai.AITrader
	execEngine       *execution.ExecutionEngine
	exchange         types.Exchange
	redis            utils.RedisClient
	warnedAIDisabled bool
}

var globalBot *Bot

// GetBot 获取交易机器人实例（单例）
func GetBot() (*Bot, error) {
	if globalBot == nil {
		aiTrader, err := ai.GetAITrader()
		if err != nil {
			// AI未配置，使用nil（会降级到规则模式）
			aiTrader = nil
		}

		globalBot = &Bot{
			aiTrader:         aiTrader,
			execEngine:       execution.GetExecutionEngine(),
			exchange:         exchange.GetBinanceExchange(),
			redis:            utils.GetRedisClient(),
			warnedAIDisabled: false,
		}
	}
	return globalBot, nil
}

// ProcessSignal 处理交易信号
func (b *Bot) ProcessSignal(ctx context.Context, marketData *types.MarketData) bool {
	logger := utils.GetLogger("bot")
	cfg := config.Get()

	symbol := marketData.Symbol
	currentPrice := marketData.CurrentPrice

	if currentPrice > 0 {
		logger.Infow("收到行情",
			"symbol", symbol,
			"price", currentPrice,
		)
	} else {
		logger.Infow("收到行情",
			"symbol", symbol,
		)
	}

	// 获取交易模式（AI/规则）
	mode := b.getAIMode()

	// 获取账户快照（用于AI决策）
	accountSnapshot := b.getAccountSnapshot()
	
	// 补充账户信息到市场数据
	if accountSnapshot != nil {
		// 检查是否有错误
		if errorMsg, hasError := accountSnapshot["error"].(string); hasError {
			logger.Debugw("账户信息获取失败，继续使用市场数据",
				"symbol", symbol,
				"error", errorMsg,
			)
		} else {
			accountInfo := &types.AccountInfo{}
			if balance, ok := accountSnapshot["balance"].(map[string]float64); ok && balance != nil {
				accountInfo.Balance = balance
			}
			if positions, ok := accountSnapshot["positions"].([]map[string]interface{}); ok && positions != nil {
				accountInfo.Positions = positions
			}
			// 只有当有有效数据时才设置
			if accountInfo.Balance != nil || (accountInfo.Positions != nil && len(accountInfo.Positions) > 0) {
				marketData.Account = accountInfo
			}
		}
	}

	var action string
	var signal *types.Signal
	var reason string

	if mode == "ai" {
		if b.aiTrader == nil {
			if !b.warnedAIDisabled {
				logger.Warn("AI未配置或未启用：已自动降级到规则策略模式（rule）")
				b.warnedAIDisabled = true
			}
			mode = "rule"
		} else {
			decision, err := b.aiTrader.MakeTradingDecision(ctx, marketData)
			if err != nil {
				logger.Warnw("AI决策失败",
					"symbol", symbol,
					"error", err,
				)
				return false
			}
			action = decision.Action
			signal = decision.Signal
			reason = decision.Reason
		}
	}

	if mode == "rule" {
		// 使用规则策略
		ruleStrategy := strategies.GetRuleStrategy()
		var fullDecision map[string]interface{}
		action, signal, reason, fullDecision = ruleStrategy.MakeDecision(marketData)

		// 保存规则决策历史（类似AI决策）
		b.saveRuleDecisionHistory(symbol, action, fullDecision)

		// 如果规则策略返回了信号，使用它
		if signal != nil {
			// signal已经设置好了
		}
	}

	// 如果是交易动作，保存信号并推送到队列
	if (action == "open_long" || action == "open_short" || action == "close_long" || action == "close_short") && signal != nil {
		// 生成唯一signalID（如果还没有）
		if signal.SignalID == "" {
			signal.SignalID = fmt.Sprintf("%s_%d_%d", symbol, time.Now().UnixNano(), signal.Timestamp)
		}
		
		// 保存信号到Redis
		signalKey := config.GetRedisKey(fmt.Sprintf("signal:%s", symbol))
		signalData := map[string]interface{}{
			"symbol":       signal.Symbol,
			"action":       signal.Action,
			"side":         signal.Side,
			"entry_price":  signal.EntryPrice,
			"stop_loss":    signal.StopLoss,
			"take_profit":  signal.TakeProfit,
			"take_profit_2": signal.TakeProfit2,
			"quantity":     signal.Quantity,
			"leverage":     signal.Leverage,
			"reason":       signal.Reason,
			"signal_id":    signal.SignalID,
			"status":       "pending",
			"timestamp":    time.Now().Unix(),
		}

		signalJSON, _ := json.Marshal(signalData)
		ttl := time.Duration(cfg.SignalTTLSec) * time.Second
		b.redis.Set(ctx, signalKey, signalJSON, ttl)

		// 追加信号历史
		historyKey := config.GetRedisKey("signal_history")
		b.redis.LPush(ctx, historyKey, signalJSON)
		maxLen := cfg.SignalHistoryMaxLen
		if maxLen <= 0 {
			maxLen = 500
		}
		b.redis.LTrim(ctx, historyKey, 0, int64(maxLen-1))

		// 推送到交易队列
		queueKey := config.GetRedisKey("trade_queue")
		b.redis.LPush(ctx, queueKey, signalJSON)
		maxQueueSize := cfg.MaxTradeQueueSize
		if maxQueueSize <= 0 {
			maxQueueSize = 100
		}
		b.redis.LTrim(ctx, queueKey, 0, int64(maxQueueSize-1))

		// 记录指标
		metrics.RecordSignal(true)

		logger.Infow("信号已推送到队列",
			"symbol", symbol,
			"action", action,
		)
		return true
	}

	// 记录指标
	metrics.RecordSignal(false)

	logger.Infow("信号处理完成",
		"symbol", symbol,
		"action", action,
		"reason", reason,
	)
	return false
}

// RunBot 运行交易机器人主循环
func (b *Bot) RunBot(ctx context.Context) error {
	logger := utils.GetLogger("bot")
	cfg := config.Get()

	logger.Info("🚀 交易机器人启动（生产模式）")
	logger.Infow("风控参数",
		"max_notional_per_trade", cfg.MaxNotionalPerTrade,
		"max_concurrent_positions", cfg.MaxConcurrentPositions,
		"market_snapshot_max_age_sec", cfg.MarketSnapshotMaxAgeSec,
		"market_snapshot_ttl_sec", cfg.MarketSnapshotTTLSec,
	)

	queueKey := config.GetRedisKey("trade_queue")
	lastGuardTS := time.Now()

	for {
		select {
		case <-ctx.Done():
			logger.Info("交易机器人停止")
			return ctx.Err()
		default:
		}

		// 后台守护：每N秒轮询一次，确保持仓有止盈止损
		now := time.Now()
		interval := time.Duration(cfg.SLTPGuardIntervalSec) * time.Second
		if now.Sub(lastGuardTS) >= interval {
			intervalTag := fmt.Sprintf("%.0fs", interval.Seconds())
			b.execEngine.EnsureSLTPGuardOnce(ctx, intervalTag)
			lastGuardTS = now
		}

		// 从队列获取信号（阻塞等待）
		result, err := b.redis.BRPop(ctx, 10*time.Second, queueKey).Result()
		if err != nil {
			// 超时或其他错误，继续循环
			continue
		}

		if len(result) < 2 {
			continue
		}

		signalJSON := result[1]
		var signalData map[string]interface{}
		if err := json.Unmarshal([]byte(signalJSON), &signalData); err != nil {
			logger.Warnw("解析信号失败", "error", err)
			continue
		}

		symbol, _ := signalData["symbol"].(string)
		action, _ := signalData["action"].(string)

		logger.Infow("收到交易指令",
			"symbol", symbol,
			"action", action,
		)

		// 构建Signal对象
		signal := &types.Signal{
			Symbol:      symbol,
			Action:      action,
			Side:        utils.GetString(signalData, "side", ""),
			EntryPrice:  utils.GetFloat(signalData, "entry_price", 0),
			StopLoss:    utils.GetFloat(signalData, "stop_loss", 0),
			TakeProfit:  utils.GetFloat(signalData, "take_profit", 0),
			TakeProfit2: utils.GetFloat(signalData, "take_profit_2", 0),
			Quantity:    utils.GetFloat(signalData, "quantity", 0),
			Leverage:    int(utils.GetFloat(signalData, "leverage", 0)),
			Reason:      utils.GetString(signalData, "reason", ""),
			SignalID:    utils.GetString(signalData, "signal_id", ""),
			Timestamp:   int64(utils.GetFloat(signalData, "timestamp", 0)),
		}

		// 执行交易
		var ok bool
		var reason string
		var order *types.Order

		if action == "close_long" || action == "close_short" {
			ok, reason, order = b.execEngine.ClosePositionFromAction(ctx, signal)
		} else if action == "open_long" || action == "open_short" {
			if signal.EntryPrice > 0 {
				ok, reason, order = b.execEngine.PlaceOrderFromSignal(ctx, signal)
			} else {
				ok, reason, order = false, "开仓信号缺少必要字段（entry_price）", nil
			}
		} else {
			ok, reason, order = false, fmt.Sprintf("跳过执行（action=%s）", action), nil
		}

		// 记录指标
		if action == "open_long" || action == "open_short" || action == "close_long" || action == "close_short" {
			metrics.RecordOrder(ok)
		}

		// 记录执行结果
		if ok {
			logger.Infow("执行成功",
				"symbol", symbol,
				"action", action,
				"order_id", order.ID,
				"reason", reason,
			)
		} else {
			logger.Warnw("执行失败",
				"symbol", symbol,
				"action", action,
				"reason", reason,
			)
		}
	}
}

// getAIMode 获取AI模式
func (b *Bot) getAIMode() string {
	cfg := config.Get()
	key := config.GetRedisKey("ai_mode")

	// 使用传入的context（如果有），否则创建新的
	ctx, cancel := utils.WithDefaultTimeout(context.Background())
	defer cancel()

	mode, err := b.redis.Get(ctx, key).Result()
	if err == nil && (mode == "ai" || mode == "rule") {
		return mode
	}

	// 默认模式：如果有AI提供商则用AI，否则用规则
	if b.aiTrader != nil {
		return "ai"
	}

	// 检查配置
	if cfg.DeepSeekEnabled && cfg.DeepSeekAPIKey != "" {
		return "ai"
	}
	if cfg.OpenAIEnabled && cfg.OpenAIAPIKey != "" {
		return "ai"
	}
	if cfg.GeminiEnabled && cfg.GeminiAPIKey != "" {
		return "ai"
	}

	return "rule"
}

// getAccountSnapshot 获取账户快照
func (b *Bot) getAccountSnapshot() map[string]interface{} {
	balance, err := b.exchange.GetBalance()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error()[:200],
		}
	}

	positions, err := b.exchange.GetPositions()
	if err != nil {
		return map[string]interface{}{
			"balance": balance,
			"error":   err.Error()[:200],
		}
	}

	positionsList := make([]map[string]interface{}, 0, len(positions))
	for _, pos := range positions {
		positionsList = append(positionsList, map[string]interface{}{
			"symbol":         pos.Symbol,
			"side":           pos.Side,
			"size":           pos.Size,
			"entry_price":    pos.EntryPrice,
			"mark_price":     pos.MarkPrice,
			"unrealized_pnl": pos.UnrealizedPnl,
			"leverage":       pos.Leverage,
		})
	}

	return map[string]interface{}{
		"balance":   balance,
		"positions": positionsList,
	}
}

// 辅助函数已迁移到utils包，使用utils.GetString和utils.GetFloat

// saveRuleDecisionHistory 保存规则决策历史
func (b *Bot) saveRuleDecisionHistory(symbol, action string, fullDecision map[string]interface{}) {
	cfg := config.Get()
	key := config.GetRedisKey("deepseek_analysis_response_history")

	// 使用带超时的context，避免阻塞
	ctx, cancel := utils.WithDefaultTimeout(context.Background())
	defer cancel()

	payload := map[string]interface{}{
		"symbol":        symbol,
		"timestamp":     time.Now().Unix(),
		"action":        action,
		"decision":      action,
		"full_decision": fullDecision,
	}

	payloadJSON, _ := json.Marshal(payload)
	b.redis.LPush(ctx, key, payloadJSON)

	// 限制历史记录长度
	maxLen := cfg.AIDecisionHistoryMaxLen
	if maxLen <= 0 {
		maxLen = 500
	}
	b.redis.LTrim(ctx, key, 0, int64(maxLen-1))

	// 更新AI统计（标记为rule模式）
	statsKey := config.GetRedisKey("ai_api_stats")
	b.redis.HSet(ctx, statsKey,
		"ts", fmt.Sprintf("%d", time.Now().Unix()),
		"symbol", symbol,
		"ok", "1",
		"action", action,
		"model", "rule",
		"latency_ms", "0",
		"total_ms", "0",
		"attempts", "0",
		"error", "",
	)
	ttl := time.Duration(cfg.AIStatsTTLSec) * time.Second
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	b.redis.Expire(ctx, statsKey, ttl)
}
