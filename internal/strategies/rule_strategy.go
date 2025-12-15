package strategies

import (
	"github.com/yuechangmingzou/nofx-go/pkg/types"
)

// RuleStrategy 规则策略接口
type RuleStrategy interface {
	MakeDecision(marketData *types.MarketData) (string, *types.Signal, string, map[string]interface{})
}

// DefaultRuleStrategy 默认规则策略（简单示例）
type DefaultRuleStrategy struct{}

// MakeDecision 做出决策
func (s *DefaultRuleStrategy) MakeDecision(marketData *types.MarketData) (string, *types.Signal, string, map[string]interface{}) {
	// 简单规则：如果RSI < 30，做多；如果RSI > 70，做空
	if marketData.RSI > 0 {
		if marketData.RSI < 30 {
			signal := &types.Signal{
				Symbol:    marketData.Symbol,
				Action:    "open_long",
				Side:      "long",
				EntryPrice: marketData.CurrentPrice,
				StopLoss:  marketData.CurrentPrice * 0.98, // 2%止损
				TakeProfit: marketData.CurrentPrice * 1.05, // 5%止盈
			}
			return "open_long", signal, "RSI超卖，做多", map[string]interface{}{
				"rsi": marketData.RSI,
			}
		}
		if marketData.RSI > 70 {
			signal := &types.Signal{
				Symbol:    marketData.Symbol,
				Action:    "open_short",
				Side:      "short",
				EntryPrice: marketData.CurrentPrice,
				StopLoss:  marketData.CurrentPrice * 1.02, // 2%止损
				TakeProfit: marketData.CurrentPrice * 0.95, // 5%止盈
			}
			return "open_short", signal, "RSI超买，做空", map[string]interface{}{
				"rsi": marketData.RSI,
			}
		}
	}

	return "wait", nil, "无交易信号", map[string]interface{}{}
}

// GetRuleStrategy 获取规则策略实例
func GetRuleStrategy() RuleStrategy {
	// 可以根据配置选择不同的策略
	return &DefaultRuleStrategy{}
}

