package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/utils"
)

// handleStatus 获取系统状态（带缓存）
func (s *Server) handleStatus(c *gin.Context) {
	// 尝试从缓存获取
	if cached, ok := globalStatusCache.get(); ok {
		c.JSON(http.StatusOK, cached)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"dry_run":   s.config.DryRun,
	}

	// Redis状态
	if err := s.redis.Ping(ctx).Err(); err != nil {
		status["redis"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		status["redis"] = map[string]interface{}{
			"status": "ok",
		}
	}

	// Binance状态（异步获取，避免阻塞）
	binanceStatus := s.probeBinance(ctx)
	status["binance"] = binanceStatus

	// AI模式
	aiMode := s.getAIMode()
	status["ai_mode"] = aiMode

	// 更新缓存
	globalStatusCache.set(status)

	c.JSON(http.StatusOK, status)
}

// handleMarketData 获取市场数据（带超时和错误处理）
func (s *Server) handleMarketData(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 从Redis读取最近扫描的市场数据
	key := config.GetRedisKey("scanner_last_scan")
	raw, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		// Redis错误时返回空数组，不返回错误（前端可以处理）
		c.JSON(http.StatusOK, gin.H{
			"items": []interface{}{},
		})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		s.logger.Warnw("解析市场数据失败", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"items": []interface{}{},
		})
		return
	}

	items, _ := data["items"].([]interface{})
	if items == nil {
		items = []interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
	})
}

// handleGetAIMode 获取AI模式
func (s *Server) handleGetAIMode(c *gin.Context) {
	mode := s.getAIMode()
	c.JSON(http.StatusOK, gin.H{
		"mode":     mode["mode"],
		"override": mode["override"],
		"default":  mode["default"],
	})
}

// handleSetAIMode 设置AI模式
func (s *Server) handleSetAIMode(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	if req.Mode != "ai" && req.Mode != "rule" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_mode"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("ai_mode")
	if err := s.redis.Set(ctx, key, req.Mode, 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mode": req.Mode,
	})
}

// handleGetAIPrompt 获取AI提示词
func (s *Server) handleGetAIPrompt(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("ai_prompt")
	raw, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		// 返回默认提示词
		c.JSON(http.StatusOK, gin.H{
			"prompt": s.config.AITraderSystemPrompt,
		})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"prompt": s.config.AITraderSystemPrompt,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prompt": data["prompt"],
	})
}

// handleSetAIPrompt 设置AI提示词
func (s *Server) handleSetAIPrompt(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("ai_prompt")
	data := map[string]interface{}{
		"prompt":    req.Prompt,
		"timestamp": time.Now().Unix(),
	}
	dataJSON, _ := json.Marshal(data)
	if err := s.redis.Set(ctx, key, dataJSON, 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleDeleteAIPrompt 删除AI提示词（恢复默认）
func (s *Server) handleDeleteAIPrompt(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("ai_prompt")
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleGetRuntimeConfig 获取运行时配置
func (s *Server) handleGetRuntimeConfig(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("runtime_config")
	raw, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"overrides": map[string]interface{}{},
		})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"overrides": map[string]interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"overrides": data["overrides"],
	})
}

// handleSetRuntimeConfig 设置运行时配置
func (s *Server) handleSetRuntimeConfig(c *gin.Context) {
	var req struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 读取现有配置
	key := config.GetRedisKey("runtime_config")
	raw, _ := s.redis.Get(ctx, key).Result()
	var data map[string]interface{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &data)
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	if data["overrides"] == nil {
		data["overrides"] = make(map[string]interface{})
	}

	overrides := data["overrides"].(map[string]interface{})
	overrides[req.Key] = req.Value
	data["overrides"] = overrides
	data["timestamp"] = time.Now().Unix()

	dataJSON, _ := json.Marshal(data)
	if err := s.redis.Set(ctx, key, dataJSON, 0).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleDeleteRuntimeConfig 删除运行时配置
func (s *Server) handleDeleteRuntimeConfig(c *gin.Context) {
	keyParam := c.Query("key")
	if keyParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_key"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 读取现有配置
	key := config.GetRedisKey("runtime_config")
	raw, _ := s.redis.Get(ctx, key).Result()
	if raw == "" {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid_data"})
		return
	}

	if overrides, ok := data["overrides"].(map[string]interface{}); ok {
		delete(overrides, keyParam)
		data["overrides"] = overrides
		data["timestamp"] = time.Now().Unix()

		dataJSON, _ := json.Marshal(data)
		if err := s.redis.Set(ctx, key, dataJSON, 0).Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "redis_error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleRuntimeConfigAudit 获取运行时配置审计日志
func (s *Server) handleRuntimeConfigAudit(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 2000 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("runtime_config_audit")
	items, err := s.redis.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
		return
	}

	results := make([]interface{}, 0, len(items))
	for _, item := range items {
		var data interface{}
		if err := json.Unmarshal([]byte(item), &data); err == nil {
			results = append(results, data)
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": results})
}

// handleWSToken 获取WebSocket token
func (s *Server) handleWSToken(c *gin.Context) {
	ttl := s.config.WSTokenTTLSec
	if ttl <= 0 {
		ttl = 60
	}

	token := utils.GenerateToken(24)
	key := config.GetRedisKey(fmt.Sprintf("ws_token:%s", token))
	payload := map[string]interface{}{
		"issued_at": time.Now().Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redis.Set(ctx, key, payloadJSON, time.Duration(ttl)*time.Second).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "redis_not_ready"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ws_token":   token,
		"expires_in": ttl,
	})
}

// handleBalance 获取余额
func (s *Server) handleBalance(c *gin.Context) {
	balance, err := s.exchange.GetBalance()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

// handlePositions 获取持仓
func (s *Server) handlePositions(c *gin.Context) {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	positions, err := s.exchange.GetPositions()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	positionsList := make([]map[string]interface{}, 0, len(positions))
	for _, pos := range positions {
		positionsList = append(positionsList, map[string]interface{}{
			"symbol":        pos.Symbol,
			"side":          pos.Side,
			"size":          pos.Size,
			"entry_price":   pos.EntryPrice,
			"mark_price":    pos.MarkPrice,
			"unrealized_pnl": pos.UnrealizedPnl,
			"leverage":      pos.Leverage,
		})
	}

	c.JSON(http.StatusOK, gin.H{"positions": positionsList})
}

// handleEquity 获取权益
func (s *Server) handleEquity(c *gin.Context) {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balanceMap, err := s.exchange.GetBalance()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	// 提取USDT余额
	balance := 0.0
	if usdt, ok := balanceMap["USDT"]; ok {
		balance = usdt
	}

	positions, err := s.exchange.GetPositions()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	unrealizedPnl := 0.0
	for _, pos := range positions {
		unrealizedPnl += pos.UnrealizedPnl
	}

	equity := balance + unrealizedPnl

	c.JSON(http.StatusOK, gin.H{
		"balance":        balance,
		"unrealized_pnl": unrealizedPnl,
		"equity":         equity,
	})
}

// handleHistory 获取历史记录
func (s *Server) handleHistory(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("signal_history")
	items, err := s.redis.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
		return
	}

	results := make([]interface{}, 0, len(items))
	for _, item := range items {
		var data interface{}
		if err := json.Unmarshal([]byte(item), &data); err == nil {
			results = append(results, data)
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": results})
}

// handleLatestAIDecision 获取最新AI决策
func (s *Server) handleLatestAIDecision(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("deepseek_analysis_response_history")
	raw, err := s.redis.LIndex(ctx, key, 0).Result()
	if err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	c.JSON(http.StatusOK, data)
}

// handleScannedSymbols 获取扫描的币种
func (s *Server) handleScannedSymbols(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := config.GetRedisKey("scanner_last_scan")
	raw, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"items": []interface{}{},
		})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"items": []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": data["items"],
		"total": data["total"],
		"ok":    data["ok"],
		"ts":    data["ts"],
	})
}

// getAIMode 获取AI模式
func (s *Server) getAIMode() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := config.GetRedisKey("ai_mode")
	mode, err := s.redis.Get(ctx, key).Result()
	if err == nil && (mode == "ai" || mode == "rule") {
		return map[string]string{
			"mode":     mode,
			"override": mode,
			"default":  s.getDefaultAIMode(),
		}
	}

	defaultMode := s.getDefaultAIMode()
	return map[string]string{
		"mode":     defaultMode,
		"override": "",
		"default":  defaultMode,
	}
}

// getDefaultAIMode 获取默认AI模式
func (s *Server) getDefaultAIMode() string {
	if s.config.DeepSeekEnabled && s.config.DeepSeekAPIKey != "" {
		return "ai"
	}
	if s.config.OpenAIEnabled && s.config.OpenAIAPIKey != "" {
		return "ai"
	}
	if s.config.GeminiEnabled && s.config.GeminiAPIKey != "" {
		return "ai"
	}
	return "rule"
}

// probeBinance 探测Binance连接
func (s *Server) probeBinance(ctx context.Context) map[string]interface{} {
	configured := s.config.BinanceAPIKey != "" && s.config.BinanceSecretKey != ""
	if !configured {
		return map[string]interface{}{
			"configured": false,
			"status":    "not_configured",
		}
	}

	// 尝试获取余额（简单测试）
	_, err := s.exchange.GetBalance()
	if err != nil {
		return map[string]interface{}{
			"configured": true,
			"status":     "error",
			"error":      err.Error()[:100],
		}
	}

	return map[string]interface{}{
		"configured": true,
		"status":     "ok",
	}
}

