package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/metrics"
	"github.com/yourusername/nofx-go/internal/utils"
	"github.com/yourusername/nofx-go/pkg/types"
)

// TradingDecision 交易决策结果
type TradingDecision struct {
	Action       string                 `json:"action"`        // open_long, open_short, close_long, close_short, hold, wait
	Signal       *types.Signal          `json:"signal"`        // 交易信号（open/close时不为nil）
	Reason       string                 `json:"reason"`        // 决策原因
	FullDecision map[string]interface{} `json:"full_decision"` // 完整决策数据
}

// AITrader AI交易员
type AITrader struct {
	provider AIProvider
	redis    utils.RedisClient
}

var globalAITrader *AITrader

// GetAITrader 获取AI交易员实例（单例）
func GetAITrader() (*AITrader, error) {
	if globalAITrader == nil {
		provider, err := GetAIProvider()
		if err != nil {
			return nil, fmt.Errorf("获取AI提供商失败: %w", err)
		}

		globalAITrader = &AITrader{
			provider: provider,
			redis:    utils.GetRedisClient(),
		}
	}
	return globalAITrader, nil
}

// LoadStrategy 加载策略文档
func (t *AITrader) LoadStrategy() string {
	cfg := config.Get()
	strategyFile := cfg.StrategyFile

	// 尝试多个路径
	possiblePaths := []string{
		strategyFile,
		"strategies/" + strategyFile,
		"../strategies/" + strategyFile,
	}

	for _, path := range possiblePaths {
		if data, err := os.ReadFile(path); err == nil {
			content := string(data)
			if strings.TrimSpace(content) != "" {
				return content
			}
		}
	}

	// 默认策略
	return "顺势狙击手策略：基于EMA趋势、布林带、RSI等技术指标进行交易决策。"
}

// FormatMarketData 格式化市场数据为AI可理解的文本
func (t *AITrader) FormatMarketData(marketData *types.MarketData) (string, error) {
	cfg := config.Get()
	logger := utils.GetLogger("ai_trader")

	// 获取系统提示词
	systemPrompt := cfg.AITraderSystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一名经验丰富的加密货币合约交易员，请根据提供的市场数据（包括链上数据、衍生品与资金数据、市场情绪指标、技术分析指标、全球宏观经济环境）自行分析交易并做出交易决策。"
	}
	systemPromptLower := strings.ToLower(systemPrompt)

	// 构建过滤后的市场数据
	filteredData := make(map[string]interface{})

	// 基础字段（始终包含）
	filteredData["symbol"] = marketData.Symbol
	filteredData["current_price"] = marketData.CurrentPrice
	filteredData["price_change_pct_24h"] = marketData.PriceChangePct24h
	filteredData["timestamp"] = marketData.Timestamp

	// 技术分析指标（如果提示词提到）
	if containsAny(systemPromptLower, []string{"技术分析", "技术指标", "指标", "ema", "rsi", "布林", "cvd", "obv"}) {
		filteredData["ema_20"] = marketData.EMA20
		filteredData["ema_50"] = marketData.EMA50
		filteredData["ema_200"] = marketData.EMA200
		filteredData["rsi"] = marketData.RSI
		if marketData.BB != nil {
			filteredData["bb"] = map[string]interface{}{
				"upper":   marketData.BB.Upper,
				"middle":  marketData.BB.Middle,
				"lower":   marketData.BB.Lower,
				"squeeze": marketData.BB.Squeeze,
			}
		}
		filteredData["cvd"] = marketData.CVD
		filteredData["obv"] = marketData.OBV
	}

	// 衍生品与资金数据（如果提示词提到）
	if containsAny(systemPromptLower, []string{"衍生品", "资金", "funding", "持仓", "open interest"}) {
		filteredData["funding_rate"] = marketData.FundingRate
		filteredData["open_interest"] = marketData.OpenInterest
		filteredData["open_interest_change"] = marketData.OpenInterestChange
	}

	// 限制数据大小（使用默认值）
	maxFieldChars := 5000
	maxTotalBytes := 120000
	filteredData = limitDictSize(filteredData, maxFieldChars, maxTotalBytes)

	// 转换为JSON
	jsonData, err := json.Marshal(filteredData)
	if err != nil {
		logger.Errorw("序列化市场数据失败", "error", err)
		return "", err
	}

	return string(jsonData), nil
}

// MakeTradingDecision 做出交易决策
func (t *AITrader) MakeTradingDecision(ctx context.Context, marketData *types.MarketData) (*TradingDecision, error) {
	logger := utils.GetLogger("ai_trader")
	cfg := config.Get()

	if t.provider == nil {
		return &TradingDecision{
			Action: "wait",
			Reason: "AI交易员未启用",
		}, nil
	}

	symbol := marketData.Symbol
	startTime := time.Now()

	// 加载策略
	strategy := t.LoadStrategy()

	// 格式化市场数据
	marketDataJSON, err := t.FormatMarketData(marketData)
	if err != nil {
		return &TradingDecision{
			Action: "wait",
			Reason: fmt.Sprintf("格式化市场数据失败: %v", err),
		}, err
	}

	// 构建提示词
	systemPrompt := cfg.AITraderSystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是一名经验丰富的加密货币合约交易员，请根据提供的市场数据（包括链上数据、衍生品与资金数据、市场情绪指标、技术分析指标、全球宏观经济环境）自行分析交易并做出交易决策。"
	}

	userPrompt := fmt.Sprintf(`策略文档：
%s

市场数据（JSON格式）：
%s

请根据策略文档和市场数据，做出交易决策。请以JSON格式返回，包含以下字段：
- action: 动作（open_long, open_short, close_long, close_short, hold, wait）
- entry_price: 入场价格（open时）
- stop_loss: 止损价格
- take_profit_1: 止盈1价格
- take_profit_2: 止盈2价格
- reason: 决策原因
- summary: 分析摘要`, strategy, marketDataJSON)

	// 调用AI API
	temperature := cfg.AITemperature
	maxTokens := cfg.AIMaxTokens

	var aiResponse *ChatResponse
	var lastError error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		attemptStart := time.Now()
		req := ChatRequest{
			Model: t.provider.GetModel(),
			Messages: []Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			Temperature: temperature,
			MaxTokens:   maxTokens,
		}

		resp, err := t.provider.ChatCompletion(ctx, req)
		attemptLatency := time.Since(attemptStart)

		if err == nil && resp.Content != "" {
			aiResponse = resp
			// 记录成功的AI请求
			metrics.RecordAIRequest(true, attemptLatency)
			break
		}

		// 记录失败的AI请求
		metrics.RecordAIRequest(false, attemptLatency)
		lastError = err
		if attempt < maxRetries-1 {
			waitTime := time.Duration(attempt+1) * 2 * time.Second
			logger.Warnw("AI API调用失败，重试中",
				"symbol", symbol,
				"attempt", attempt+1,
				"error", err,
				"wait", waitTime,
			)
			time.Sleep(waitTime)
		}
	}

	if aiResponse == nil || aiResponse.Content == "" {
		totalMs := int(time.Since(startTime).Milliseconds())
		// 记录失败的AI请求
		metrics.RecordAIRequest(false, time.Since(startTime))
		t.writeAIStats(symbol, false, "wait", 0, totalMs, maxRetries, lastError.Error())
		return &TradingDecision{
			Action: "wait",
			Reason: fmt.Sprintf("无法获取AI响应: %v", lastError),
		}, lastError
	}

	// 解析AI响应
	decision, err := t.parseAIResponse(aiResponse.Content, symbol)
	if err != nil {
		totalMs := int(time.Since(startTime).Milliseconds())
		t.writeAIStats(symbol, false, "wait", aiResponse.LatencyMs, totalMs, maxRetries, err.Error())
		return &TradingDecision{
			Action: "wait",
			Reason: fmt.Sprintf("解析AI响应失败: %v", err),
		}, err
	}

	// 保存历史记录
	t.saveDecisionHistory(symbol, decision, aiResponse.LatencyMs, int(time.Since(startTime).Milliseconds()))

	// 记录统计
	t.writeAIStats(symbol, true, decision.Action, aiResponse.LatencyMs, int(time.Since(startTime).Milliseconds()), 1, "")

	logger.Infow("AI交易决策完成",
		"symbol", symbol,
		"action", decision.Action,
		"latency_ms", aiResponse.LatencyMs,
	)

	return decision, nil
}

// parseAIResponse 解析AI响应
func (t *AITrader) parseAIResponse(content, symbol string) (*TradingDecision, error) {
	// 尝试提取JSON（可能被```json包裹）
	jsonContent := content
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = strings.TrimSpace(content[start : start+end])
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.Index(content[start:], "```")
		if end > 0 {
			jsonContent = strings.TrimSpace(content[start : start+end])
		}
	}

	var decisionData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &decisionData); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 提取action
	action, _ := decisionData["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))

	// 验证action
	allowedActions := map[string]bool{
		"open_long":   true,
		"open_short":  true,
		"close_long":  true,
		"close_short": true,
		"hold":        true,
		"wait":        true,
	}

	if !allowedActions[action] {
		action = "wait"
	}

	decision := &TradingDecision{
		Action:       action,
		Reason:       getString(decisionData, "reason", ""),
		FullDecision: decisionData,
	}

	// 如果是open或close动作，构建Signal
	if action == "open_long" || action == "open_short" || action == "close_long" || action == "close_short" {
		signal := &types.Signal{
			Symbol:    symbol,
			Action:    action,
			Timestamp: time.Now().Unix(),
		}

		if action == "open_long" || action == "open_short" {
			signal.Side = "long"
			if action == "open_short" {
				signal.Side = "short"
			}
			signal.EntryPrice = getFloat(decisionData, "entry_price", 0)
			signal.StopLoss = getFloat(decisionData, "stop_loss", 0)
			signal.TakeProfit = getFloat(decisionData, "take_profit_1", 0)
		}

		decision.Signal = signal
	}

	return decision, nil
}

// writeAIStats 记录AI统计信息
func (t *AITrader) writeAIStats(symbol string, ok bool, action string, latencyMs, totalMs, attempts int, errorMsg string) {
	cfg := config.Get()
	key := config.GetRedisKey("ai_api_stats")

	stats := map[string]interface{}{
		"ts":         time.Now().Unix(),
		"symbol":     symbol,
		"ok":         ok,
		"action":     action,
		"model":      t.provider.GetModel(),
		"latency_ms": latencyMs,
		"total_ms":   totalMs,
		"attempts":   attempts,
		"error":      errorMsg,
	}

	statsJSON, _ := json.Marshal(stats)
	ttl := time.Duration(cfg.AIStatsTTLSec) * time.Second
	t.redis.Set(context.Background(), key, statsJSON, ttl)
}

// saveDecisionHistory 保存决策历史
func (t *AITrader) saveDecisionHistory(symbol string, decision *TradingDecision, latencyMs, totalMs int) {
	cfg := config.Get()

	historyData := map[string]interface{}{
		"symbol":        symbol,
		"action":        decision.Action,
		"reason":        decision.Reason,
		"latency_ms":    latencyMs,
		"total_ms":      totalMs,
		"timestamp":     time.Now().Unix(),
		"full_decision": decision.FullDecision,
	}

	if decision.Signal != nil {
		historyData["signal"] = decision.Signal
	}

	historyJSON, _ := json.Marshal(historyData)

	// 保存到决策历史列表
	key := config.GetRedisKey("ai_decision_history")
	ctx := context.Background()
	t.redis.LPush(ctx, key, historyJSON)
	maxLen := cfg.AIDecisionHistoryMaxLen
	t.redis.LTrim(ctx, key, 0, int64(maxLen-1))
}

// 辅助函数
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func getString(m map[string]interface{}, key string, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func getFloat(m map[string]interface{}, key string, defaultValue float64) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return defaultValue
}

func limitDictSize(data map[string]interface{}, maxFieldChars, maxTotalBytes int) map[string]interface{} {
	// 简化实现：只限制字符串字段长度
	result := make(map[string]interface{})
	for k, v := range data {
		if str, ok := v.(string); ok && len(str) > maxFieldChars {
			result[k] = str[:maxFieldChars] + "...[已截断]"
		} else {
			result[k] = v
		}
	}
	return result
}
