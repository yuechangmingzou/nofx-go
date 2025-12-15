package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/internal/utils"
	"github.com/yuechangmingzou/nofx-go/pkg/types"
)

// EnsureSLTPGuardOnce 确保止损止盈守护（单次执行）
func (e *ExecutionEngine) EnsureSLTPGuardOnce(ctx context.Context, intervalTag string) {
	logger := utils.GetLogger("execution_guard")
	cfg := config.Get()

	// 获取所有持仓
	positions, err := e.exchange.GetPositions()
	if err != nil {
		logger.Warnw("获取持仓失败", "error", err)
		return
	}

	if len(positions) == 0 {
		return
	}

	// 构建持仓映射
	posMap := make(map[string]map[string]float64)
	for _, pos := range positions {
		if posMap[pos.Symbol] == nil {
			posMap[pos.Symbol] = make(map[string]float64)
		}
		posMap[pos.Symbol][strings.ToLower(pos.Side)] = pos.Size
	}

	// 遍历每个持仓，检查并补挂止损止盈
	for _, pos := range positions {
		symbol := pos.Symbol
		side := strings.ToLower(pos.Side)
		positionSide := strings.ToUpper(pos.Side)
		size := pos.Size

		if size <= 0 {
			continue
		}

		// 获取分布式锁
		lockKey := fmt.Sprintf("guard:lock:%s:%s", symbol, positionSide)
		// 使用60秒TTL，确保有足够时间完成操作
		lockToken, err := e.acquireLock(ctx, lockKey, 60*time.Second)
		if err != nil {
			continue
		}

		func() {
			defer e.releaseLock(ctx, lockKey, lockToken)

			// 从Redis读取保护信息
			protectionKey := config.GetRedisKey(fmt.Sprintf("protection:%s:%s", symbol, positionSide))
			protectionJSON, err := e.redis.Get(ctx, protectionKey).Result()
			if err != nil {
				// 没有保护信息，跳过
				return
			}

			var protection map[string]interface{}
			if err := json.Unmarshal([]byte(protectionJSON), &protection); err != nil {
				return
			}

			stopLoss := utils.GetFloat(protection, "stop_loss", 0)
			takeProfit1 := utils.GetFloat(protection, "take_profit_1", 0)
			takeProfit2 := utils.GetFloat(protection, "take_profit_2", 0)
			tp1Ratio := utils.GetFloat(protection, "tp1_ratio", cfg.TP1PartialRatio)
			signalID := utils.GetString(protection, "signal_id", "")

			if stopLoss <= 0 || takeProfit1 <= 0 {
				e.saveAudit(ctx, map[string]interface{}{
					"ts":            time.Now().Unix(),
					"event":         "guard_invalid_protection_params",
					"symbol":        symbol,
					"side":          side,
					"interval":      intervalTag,
					"stop_loss":     stopLoss,
					"take_profit_1": takeProfit1,
				})
				return
			}

			// 获取当前挂单，检查是否已有止损止盈单
			hasSL := false
			hasTP1 := false
			hasTP2 := false
			
			orders, err := e.exchange.GetOpenOrders(symbol)
			if err == nil && orders != nil {
				for _, o := range orders {
					// 检查止损单
					if o.ReduceOnly && (o.OrderType == "STOP" || o.OrderType == "STOP_MARKET") {
						if (side == "LONG" && o.Side == "SELL") || (side == "SHORT" && o.Side == "BUY") {
							hasSL = true
						}
					}
					// 检查止盈单
					if o.ReduceOnly && (o.OrderType == "TAKE_PROFIT" || o.OrderType == "TAKE_PROFIT_MARKET") {
						if (side == "LONG" && o.Side == "SELL") || (side == "SHORT" && o.Side == "BUY") {
							// 根据价格判断是TP1还是TP2
							if takeProfit1 > 0 && math.Abs(o.Price-takeProfit1) < math.Abs(o.Price-takeProfit2) {
								hasTP1 = true
							} else if takeProfit2 > 0 {
								hasTP2 = true
							}
						}
					}
				}
			}

			// 计算分批止盈数量
			tp1Ratio = math.Max(0.0, math.Min(tp1Ratio, 1.0))
			amt1 := math.Round(size*tp1Ratio*1e8) / 1e8
			amt2 := math.Round(math.Max(0.0, size-amt1)*1e8) / 1e8
			if amt1 <= 0 {
				amt1 = size
				amt2 = 0
			}
			needTP2 := takeProfit2 > 0 && amt2 > 0

			// 补挂止损单
			if !hasSL {
				slOrder, err := e.placeStopLossOrder(ctx, symbol, positionSide, size, stopLoss)
				if err != nil {
					logger.Warnw("补挂止损单失败",
						"symbol", symbol,
						"error", err,
					)
				} else {
					e.saveAudit(ctx, map[string]interface{}{
						"ts":        time.Now().Unix(),
						"event":     "guard_stop_loss_placed",
						"symbol":    symbol,
						"signal_id": signalID,
						"side":      side,
						"amount":    size,
						"stop_loss": stopLoss,
						"order_id":  slOrder.ID,
						"interval":  intervalTag,
					})
				}
			}

			// 补挂止盈单1
			if !hasTP1 {
				tpOrder1, err := e.placeTakeProfitOrder(ctx, symbol, positionSide, amt1, takeProfit1)
				if err != nil {
					logger.Warnw("补挂止盈单1失败",
						"symbol", symbol,
						"error", err,
					)
				} else {
					e.saveAudit(ctx, map[string]interface{}{
						"ts":          time.Now().Unix(),
						"event":       "guard_take_profit_placed",
						"symbol":      symbol,
						"signal_id":   signalID,
						"side":        side,
						"amount":      amt1,
						"tp_level":    1,
						"take_profit": takeProfit1,
						"order_id":    tpOrder1.ID,
						"interval":    intervalTag,
					})
				}
			}

			// 补挂止盈单2
			if needTP2 && !hasTP2 {
				tpOrder2, err := e.placeTakeProfitOrder(ctx, symbol, positionSide, amt2, takeProfit2)
				if err != nil {
					logger.Warnw("补挂止盈单2失败",
						"symbol", symbol,
						"error", err,
					)
				} else {
					e.saveAudit(ctx, map[string]interface{}{
						"ts":          time.Now().Unix(),
						"event":       "guard_take_profit_placed",
						"symbol":      symbol,
						"signal_id":   signalID,
						"side":        side,
						"amount":      amt2,
						"tp_level":    2,
						"take_profit": takeProfit2,
						"order_id":    tpOrder2.ID,
						"interval":    intervalTag,
					})
				}
			}
		}()
	}

	// 清理已平仓的保护信息
	e.cleanupProtection(ctx, posMap)
}

// cleanupProtection 清理已平仓的保护信息
func (e *ExecutionEngine) cleanupProtection(ctx context.Context, posMap map[string]map[string]float64) {
	logger := utils.GetLogger("execution_guard")
	pattern := config.GetRedisKey("protection:*")

	// 使用SCAN命令替代KEYS，避免阻塞Redis
	var keys []string
	var cursor uint64 = 0
	for {
		var err error
		var batch []string
		batch, cursor, err = e.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Debugw("清理保护信息失败", "error", err)
			return
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break // 扫描完成
		}
	}

	cancelledTotal := 0
	deletedProt := 0

	for _, key := range keys {
		// 解析key: nofx:protection:{SYMBOL}:{LONG/SHORT}
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			continue
		}
		symbol := strings.ToUpper(parts[len(parts)-2])
		positionSide := strings.ToUpper(parts[len(parts)-1])

		if posMap[symbol] == nil {
			posMap[symbol] = make(map[string]float64)
		}
		side := strings.ToLower(positionSide)
		curAmt := posMap[symbol][side]

		if curAmt > 0 {
			continue // 仍有持仓，不清理
		}

		// 持仓已平，撤销残留的reduceOnly订单
		orders, err := e.exchange.GetOpenOrders(symbol)
		if err == nil && orders != nil {
			cancelled := 0
			for _, o := range orders {
				if !isReduceOnly(o) {
					continue
				}
				if getOrderPositionSide(o) != positionSide {
					continue
				}

				// 撤销订单
				if err := e.exchange.CancelOrder(symbol, o.ID); err == nil {
					cancelled++
				}
			}

			if cancelled > 0 {
				cancelledTotal += cancelled
				e.saveAudit(ctx, map[string]interface{}{
					"ts":            time.Now().Unix(),
					"event":         "auto_cancel_reduceonly_after_flat",
					"symbol":        symbol,
					"position_side": positionSide,
					"count":         cancelled,
				})
			}
		}

		// 删除保护信息
		if err := e.redis.Del(ctx, key).Err(); err == nil {
			deletedProt++
		}
	}

	if deletedProt > 0 {
		logger.Debugw("清理保护信息完成",
			"deleted", deletedProt,
			"cancelled_orders", cancelledTotal,
		)
	}
}

// isReduceOnly 判断订单是否是reduceOnly
func isReduceOnly(order *types.Order) bool {
	// Binance API返回的订单中，reduceOnly字段在响应中
	// 这里简化判断：如果是TAKE_PROFIT或STOP_MARKET类型，通常是reduceOnly
	return order.OrderType == "TAKE_PROFIT" || order.OrderType == "TAKE_PROFIT_MARKET" ||
		order.OrderType == "STOP" || order.OrderType == "STOP_MARKET"
}

// isStopLossOrder 判断是否是止损单
func isStopLossOrder(order *types.Order) bool {
	return order.OrderType == "STOP" || order.OrderType == "STOP_MARKET"
}

// isTakeProfitOrder 判断是否是止盈单
func isTakeProfitOrder(order *types.Order) bool {
	return order.OrderType == "TAKE_PROFIT" || order.OrderType == "TAKE_PROFIT_MARKET"
}

// getOrderPositionSide 获取订单的持仓方向
func getOrderPositionSide(order *types.Order) string {
	return strings.ToUpper(order.PositionSide)
}

// getTakeProfitPrice 获取止盈价格
func getTakeProfitPrice(order *types.Order) float64 {
	if order.Price > 0 {
		return order.Price
	}
	return order.StopPrice
}

// SaveProtection 保存保护信息（止损止盈价格）
func (e *ExecutionEngine) SaveProtection(ctx context.Context, symbol, side string, stopLoss, takeProfit1, takeProfit2 float64, signalID string) {
	cfg := config.Get()
	key := config.GetRedisKey(fmt.Sprintf("protection:%s:%s", symbol, strings.ToUpper(side)))

	protection := map[string]interface{}{
		"stop_loss":     stopLoss,
		"take_profit_1": takeProfit1,
		"take_profit_2": takeProfit2,
		"tp1_ratio":     cfg.TP1PartialRatio,
		"signal_id":     signalID,
		"timestamp":     time.Now().Unix(),
	}

	protectionJSON, _ := json.Marshal(protection)
	ttl := time.Duration(cfg.ProtectionTTLSec) * time.Second
	e.redis.Set(ctx, key, protectionJSON, ttl)
}

// 辅助函数已迁移到utils包，使用utils.GetFloat和utils.GetString
