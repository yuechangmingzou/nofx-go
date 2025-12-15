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
	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/metrics"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境应该检查Origin
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // 允许无Origin的请求（可能是非浏览器客户端）
		}
		// TODO: 在生产环境中验证Origin白名单
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
			wsCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			marketData := s.getMarketDataForWS(wsCtx)
			cancel()

			data := map[string]interface{}{
				"timestamp":   time.Now().Unix(),
				"market_data": marketData,
			}

			dataJSON, err := json.Marshal(data)
			if err != nil {
				s.logger.Warnw("WebSocket序列化失败", "error", err)
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, dataJSON); err != nil {
				s.logger.Warnw("WebSocket发送失败", "error", err)
				metrics.RecordWebSocketMessage(false)
				return
			}
			metrics.RecordWebSocketMessage(true)
		}
	}
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

