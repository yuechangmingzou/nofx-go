package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/utils"
)

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	client      *http.Client
	rateLimiter *RateLimiter
	baseURL     string
}

var globalHTTPClient *HTTPClient

// GetHTTPClient 获取全局HTTP客户端（单例）
func GetHTTPClient() *HTTPClient {
	if globalHTTPClient == nil {
		cfg := config.Get()
		globalHTTPClient = &HTTPClient{
			client: &http.Client{
				Timeout: time.Duration(cfg.BinanceHTTPTimeoutSec) * time.Second,
			},
			rateLimiter: NewRateLimiter(10.0, 20), // 10 req/s, capacity 20
			baseURL:     cfg.BinanceFAPIBaseURL,
		}
	}
	return globalHTTPClient
}

// FetchJSON 获取JSON数据（带限流和重试）
func (c *HTTPClient) FetchJSON(ctx context.Context, endpoint string, params map[string]string) (interface{}, error) {
	// 等待退避窗口（如果有）
	globalBackoff := GetGlobalBackoff()
	globalBackoff.WaitBackoff("binance")

	// 应用限流
	c.rateLimiter.Wait(1)

	// 构建URL
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// 添加查询参数
	if params != nil {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 处理响应
	if resp.StatusCode == http.StatusOK {
		// 成功，重置退避
		globalBackoff := GetGlobalBackoff()
		globalBackoff.ResetBackoff("binance")

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read body failed: %w", err)
		}

		// 解析JSON
		var result interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse JSON failed: %w", err)
		}

		return result, nil
	} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 418 {
		// 限流处理
		retryAfterStr := resp.Header.Get("Retry-After")
		var retryAfter *float64
		if retryAfterStr != "" {
			retryAfter = ParseRetryAfter(retryAfterStr)
		}

		globalBackoff := GetGlobalBackoff()
		waitSec := globalBackoff.OnRateLimited("binance", resp.StatusCode, retryAfter)
		logger := utils.GetLogger("exchange")
		logger.Warnw("API rate limited",
			"status", resp.StatusCode,
			"endpoint", endpoint,
			"wait_sec", waitSec,
		)

		return nil, fmt.Errorf("rate limited: HTTP %d, wait %.1fs", resp.StatusCode, waitSec)
	} else {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
}

