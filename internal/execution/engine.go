package execution

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/exchange"
	"github.com/yourusername/nofx-go/internal/utils"
	"github.com/yourusername/nofx-go/pkg/types"
)

// ExecutionEngine 执行引擎
type ExecutionEngine struct {
	exchange *exchange.BinanceExchange
	redis    utils.RedisClient
}

var globalEngine *ExecutionEngine

// GetExecutionEngine 获取执行引擎实例（单例）
func GetExecutionEngine() *ExecutionEngine {
	if globalEngine == nil {
		globalEngine = &ExecutionEngine{
			exchange: exchange.GetBinanceExchange(),
			redis:    utils.GetRedisClient(),
		}
	}
	return globalEngine
}

// PlaceOrderFromSignal 从交易信号下单
func (e *ExecutionEngine) PlaceOrderFromSignal(ctx context.Context, signal *types.Signal) (bool, string, *types.Order) {
	logger := utils.GetLogger("execution")
	cfg := config.Get()

	symbol := signal.Symbol
	signalID := signal.Symbol // 简化：使用symbol作为signal_id

	// 第一步：获取分布式锁
	lockKey := fmt.Sprintf("execution:lock:%s", symbol)
	lockToken, err := e.acquireLock(ctx, lockKey, 30*time.Second)
	if err != nil {
		return false, "获取锁失败（可能有并发下单）", nil
	}
	defer e.releaseLock(ctx, lockKey, lockToken)

	// 第二步：去重检查
	if !e.checkAndSetDedupe(ctx, symbol, signal, cfg.OrderDedupeWindow) {
		return false, "去重命中（短时间重复信号）", nil
	}

	// 第三步：保存审计日志
	e.saveAudit(ctx, map[string]interface{}{
		"ts":        time.Now().Unix(),
		"event":     "pre_order",
		"symbol":    symbol,
		"signal_id": signalID,
		"action":    signal.Action,
		"side":      signal.Side,
		"entry":     signal.EntryPrice,
		"stop_loss": signal.StopLoss,
		"take_profit": signal.TakeProfit,
	})

	// 第四步：计算下单数量
	notionalUSDT := cfg.StratDefaultNotionalUSDT
	if signal.Quantity > 0 {
		notionalUSDT = signal.Quantity * signal.EntryPrice
	}

	// 验证价格合理性
	if signal.EntryPrice <= 0 {
		return false, "入场价格无效", nil
	}

	// 第五步：下单
	orderReq := types.OrderRequest{
		Symbol:       symbol,
		Side:         e.mapSide(signal.Action),
		PositionSide: strings.ToUpper(signal.Side),
		OrderType:    "LIMIT",
		Quantity:     notionalUSDT / signal.EntryPrice,
		Price:        &signal.EntryPrice,
		TimeInForce:  "GTC",
	}

	order, err := e.exchange.PlaceOrder(orderReq)
	if err != nil {
		e.saveAudit(ctx, map[string]interface{}{
			"ts":     time.Now().Unix(),
			"event":  "order_failed",
			"symbol": symbol,
			"error":  err.Error(),
		})
		return false, fmt.Sprintf("下单失败: %v", err), nil
	}

	// 第六步：订单确认
	confirmed, confirmReason := e.confirmOrder(ctx, symbol, order.ID, 30*time.Second)
	if !confirmed {
		logger.Warnw("订单确认失败",
			"symbol", symbol,
			"order_id", order.ID,
			"reason", confirmReason,
		)
		// 即使确认失败，也返回订单（可能只是网络延迟）
	}

	// 第七步：保存保护信息（用于守护进程）
	if signal.StopLoss > 0 || signal.TakeProfit > 0 {
		e.SaveProtection(ctx, symbol, signal.Side, signal.StopLoss, signal.TakeProfit, 0, signalID)
	}

	// 第八步：下止损单（由守护进程补挂，这里先保存保护信息）
	// 注意：实际下单由守护进程确保，避免重复下单

	// 第九步：保存交易历史
	e.pushTradeHistory(ctx, map[string]interface{}{
		"ts":        time.Now().Unix(),
		"event":     "order_placed",
		"symbol":    symbol,
		"signal_id": signalID,
		"order_id":  order.ID,
		"action":    signal.Action,
		"side":      signal.Side,
		"entry":     signal.EntryPrice,
		"quantity":  order.Quantity,
	})

	logger.Infow("订单执行成功",
		"symbol", symbol,
		"order_id", order.ID,
		"action", signal.Action,
	)

	return true, "订单执行成功", order
}

// ClosePositionFromAction 从动作平仓
func (e *ExecutionEngine) ClosePositionFromAction(ctx context.Context, signal *types.Signal) (bool, string, *types.Order) {
	logger := utils.GetLogger("execution")

	symbol := signal.Symbol
	action := signal.Action

	// 判断平仓方向
	var side string
	var positionSide string
	if action == "close_long" {
		side = "SELL"
		positionSide = "LONG"
	} else if action == "close_short" {
		side = "BUY"
		positionSide = "SHORT"
	} else {
		return false, fmt.Sprintf("无效的平仓动作: %s", action), nil
	}

	// 获取当前持仓
	position, err := e.exchange.GetPosition(symbol)
	if err != nil {
		return false, fmt.Sprintf("获取持仓失败: %v", err), nil
	}

	if position == nil || position.Size == 0 {
		return false, "当前无持仓", nil
	}

	// 验证持仓方向
	if position.Side != positionSide {
		return false, fmt.Sprintf("持仓方向不匹配: 期望%s, 实际%s", positionSide, position.Side), nil
	}

	// 获取分布式锁
	lockKey := fmt.Sprintf("execution:lock:%s", symbol)
	lockToken, err := e.acquireLock(ctx, lockKey, 30*time.Second)
	if err != nil {
		return false, "获取锁失败", nil
	}
	defer e.releaseLock(ctx, lockKey, lockToken)

	// 下平仓单（市价单）
	orderReq := types.OrderRequest{
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		OrderType:    "MARKET",
		Quantity:     position.Size,
		ReduceOnly:   true,
	}

	order, err := e.exchange.PlaceOrder(orderReq)
	if err != nil {
		e.saveAudit(ctx, map[string]interface{}{
			"ts":     time.Now().Unix(),
			"event":  "close_failed",
			"symbol": symbol,
			"error":  err.Error(),
		})
		return false, fmt.Sprintf("平仓失败: %v", err), nil
	}

	// 保存交易历史
	e.pushTradeHistory(ctx, map[string]interface{}{
		"ts":       time.Now().Unix(),
		"event":    "position_closed",
		"symbol":   symbol,
		"order_id": order.ID,
		"action":   action,
		"size":     position.Size,
	})

	logger.Infow("平仓成功",
		"symbol", symbol,
		"order_id", order.ID,
		"action", action,
	)

	return true, "平仓成功", order
}

// mapSide 映射交易方向
func (e *ExecutionEngine) mapSide(action string) string {
	if action == "open_long" || action == "close_short" {
		return "BUY"
	}
	return "SELL"
}

// placeStopLossOrder 下止损单
func (e *ExecutionEngine) placeStopLossOrder(ctx context.Context, symbol, side string, quantity, stopPrice float64) (*types.Order, error) {
	orderSide := "SELL"
	if side == "SHORT" {
		orderSide = "BUY"
	}

	orderReq := types.OrderRequest{
		Symbol:       symbol,
		Side:         orderSide,
		PositionSide: strings.ToUpper(side),
		OrderType:    "STOP_MARKET",
		Quantity:     quantity,
		StopPrice:    &stopPrice,
		ReduceOnly:   true,
	}

	return e.exchange.PlaceOrder(orderReq)
}

// placeTakeProfitOrder 下止盈单
func (e *ExecutionEngine) placeTakeProfitOrder(ctx context.Context, symbol, side string, quantity, tpPrice float64) (*types.Order, error) {
	orderSide := "SELL"
	if side == "SHORT" {
		orderSide = "BUY"
	}

	orderReq := types.OrderRequest{
		Symbol:       symbol,
		Side:         orderSide,
		PositionSide: strings.ToUpper(side),
		OrderType:    "TAKE_PROFIT_MARKET",
		Quantity:     quantity,
		StopPrice:    &tpPrice,
		ReduceOnly:   true,
	}

	return e.exchange.PlaceOrder(orderReq)
}

// confirmOrder 确认订单状态
func (e *ExecutionEngine) confirmOrder(ctx context.Context, symbol, orderID string, timeout time.Duration) (bool, string) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, "上下文取消"
		case <-ticker.C:
			if time.Now().After(deadline) {
				return false, "确认超时"
			}

			order, err := e.exchange.GetOrder(symbol, orderID)
			if err != nil {
				continue
			}

			if order.Status == "FILLED" {
				return true, "订单已成交"
			}
			if order.Status == "CANCELED" || order.Status == "REJECTED" {
				return false, fmt.Sprintf("订单状态: %s", order.Status)
			}
		}
	}
}

