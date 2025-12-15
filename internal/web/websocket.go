package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/internal/metrics"
	"github.com/yuechangmingzou/nofx-go/internal/utils"
	"github.com/yuechangmingzou/nofx-go/pkg/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		cfg := config.Get()
		origin := r.Header.Get("Origin")
		
		// 允许无Origin的请求（可能是非浏览器客户端）
		if origin == "" {
			return true
		}
		
		// 如果配置了允许的Origin列表，进行验证
		allowedOrigins := cfg.WebAllowedOrigins
		if len(allowedOrigins) > 0 {
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			// 如果配置了白名单但不在列表中，拒绝
			return false
		}
		
		// 如果没有配置白名单，在非生产环境允许所有Origin
		// 生产环境建议配置WEB_ALLOWED_ORIGINS环境变量
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// handleWebSocket WebSocket处理
func (s *Server) handleWebSocket(c *gin.Context) {
	// 从Sec-WebSocket-Protocol获取token
	wsToken := ""
	protocols := c.GetHeader("Sec-WebSocket-Protocol")
	if protocols != "" {
		// 解析协议：["nofx", "<token>"]
		parts := splitAndTrim(protocols, ",")
		if len(parts) >= 2 && parts[0] == "nofx" {
			wsToken = parts[1]
		} else if len(parts) > 0 {
			// 兜底：取最后一个非nofx的值
			for i := len(parts) - 1; i >= 0; i-- {
				if parts[i] != "nofx" {
					wsToken = parts[i]
					break
				}
			}
		}
	}

	if wsToken == "" {
		c.JSON(400, gin.H{"error": "missing_ws_token"})
		return
	}

	// 验证token
	ctx, cancel := utils.WithDefaultTimeout(context.Background())
	defer cancel()

	key := config.GetRedisKey(fmt.Sprintf("ws_token:%s", wsToken))
	_, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid_ws_token"})
		return
	}

	// 一次性消费token
	_ = s.redis.Del(ctx, key)

	// 升级连接
	header := make(http.Header)
	header.Set("Sec-WebSocket-Protocol", "nofx")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, header)
	if err != nil {
		s.logger.Warnw("WebSocket升级失败", "error", err)
		return
	}
	defer conn.Close()

	metrics.RecordWebSocketConnection(true)

	// 设置读写超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// 发送数据循环
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 心跳检测
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// 错误通道
	errChan := make(chan error, 1)

	// 读取goroutine（用于检测连接关闭）
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errChan:
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Warnw("WebSocket连接异常关闭", "error", err)
			}
			return
		case <-pingTicker.C:
			// 发送ping保持连接
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Warnw("WebSocket ping失败", "error", err)
				return
			}
		case <-ticker.C:
			// 获取最新市场数据
			wsCtx, cancel := utils.WithShortTimeout(context.Background())
			
			// 获取状态
			status := s.getStatusForWS(wsCtx)
			
			// 获取持仓
			positions := s.getPositionsForWS(wsCtx)
			
			// 获取余额
			balance := s.getBalanceForWS(wsCtx)
			
			// 获取市场数据
			marketData := s.getMarketDataForWS(wsCtx)
			cancel()

			// 发送状态更新
			if status != nil {
				data := map[string]interface{}{
					"type":      "status",
					"timestamp": time.Now().Unix(),
				}
				for k, v := range status {
					data[k] = v
				}
				if err := s.sendWSMessage(conn, data); err != nil {
					return
				}
			}

			// 发送持仓更新
			if positions != nil {
				data := map[string]interface{}{
					"type":      "positions",
					"positions": positions,
					"timestamp": time.Now().Unix(),
				}
				if err := s.sendWSMessage(conn, data); err != nil {
					return
				}
			}

			// 发送余额更新
			if balance != nil {
				data := map[string]interface{}{
					"type":      "balance",
					"balance":   balance,
					"timestamp": time.Now().Unix(),
				}
				if err := s.sendWSMessage(conn, data); err != nil {
					return
				}
			}

			// 发送市场数据更新
			if marketData != nil && marketData["items"] != nil {
				data := map[string]interface{}{
					"type":        "market_data",
					"market_data": marketData,
					"timestamp":   time.Now().Unix(),
				}
				if err := s.sendWSMessage(conn, data); err != nil {
					return
				}
			}
		}
	}
}

// sendWSMessage 发送WebSocket消息
func (s *Server) sendWSMessage(conn *websocket.Conn, data map[string]interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		s.logger.Warnw("WebSocket序列化失败", "error", err)
		return err
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, dataJSON); err != nil {
		s.logger.Warnw("WebSocket发送失败", "error", err)
		metrics.RecordWebSocketMessage(false)
		return err
	}
	metrics.RecordWebSocketMessage(true)
	return nil
}

// getStatusForWS 获取WebSocket用的状态数据
func (s *Server) getStatusForWS(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"dry_run": s.config.DryRun,
	}

	// AI模式
	aiMode := s.getAIMode()
	status["ai_mode"] = aiMode["mode"]

	return status
}

// getPositionsForWS 获取WebSocket用的持仓数据
func (s *Server) getPositionsForWS(ctx context.Context) []map[string]interface{} {
	positions, err := s.exchange.GetPositions()
	if err != nil {
		return nil
	}

	positionsList := make([]map[string]interface{}, 0, len(positions))
	for _, pos := range positions {
		unrealizedPnlPct := utils.CalculateUnrealizedPnlPct(pos)

		positionsList = append(positionsList, map[string]interface{}{
			"symbol":            pos.Symbol,
			"side":              pos.Side,
			"size":              pos.Size,
			"entry_price":       pos.EntryPrice,
			"mark_price":        pos.MarkPrice,
			"unrealized_pnl":    pos.UnrealizedPnl,
			"unrealized_pnl_pct": unrealizedPnlPct,
			"leverage":          pos.Leverage,
		})
	}

	return positionsList
}

// getBalanceForWS 获取WebSocket用的余额数据
func (s *Server) getBalanceForWS(ctx context.Context) float64 {
	balanceMap, err := s.exchange.GetBalance()
	if err != nil {
		return 0
	}

	// 提取USDT余额
	if total, ok := balanceMap["total"].(float64); ok {
		return total
	}
	return 0
}

// getMarketDataForWS 获取WebSocket用的市场数据
func (s *Server) getMarketDataForWS(ctx context.Context) map[string]interface{} {
	key := config.GetRedisKey("scanner_last_scan")
	raw, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return map[string]interface{}{
			"items": []interface{}{},
		}
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return map[string]interface{}{
			"items": []interface{}{},
		}
	}

	return map[string]interface{}{
		"items": data["items"],
	}
}

// splitAndTrim 分割字符串并去除空格
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

