package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
)

// GetBalance 获取账户余额
func (be *BinanceExchange) GetBalance() (map[string]float64, error) {
	cfg := config.Get()
	if cfg.DryRun {
		return map[string]float64{
			"total": 10000.0,
			"free":  10000.0,
			"used":  0.0,
		}, nil
	}

	params := map[string]string{}

	reqURL, err := be.buildSignedURL("/fapi/v2/balance", params, http.MethodGet)
	if err != nil {
		return nil, fmt.Errorf("build signed URL failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	cfg = config.Get()
	httpReq.Header.Set("X-MBX-APIKEY", cfg.BinanceAPIKey)

	resp, err := be.client.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get balance failed: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	var balances []map[string]interface{}
	if err := json.Unmarshal(body, &balances); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// 查找USDT余额
	result := map[string]float64{
		"total": 0.0,
		"free":  0.0,
		"used":  0.0,
	}

	for _, bal := range balances {
		if asset, ok := bal["asset"].(string); ok && asset == "USDT" {
			result["total"], _ = parseFloatValue(bal["balance"])
			result["free"], _ = parseFloatValue(bal["availableBalance"])
			result["used"] = result["total"] - result["free"]
			break
		}
	}

	return result, nil
}

// GetMarketInfo 获取市场信息（注意：此方法在binance.go中实现，这里只是占位）
// 实际实现在binance.go中，因为需要访问markets字段

// GetTickSize 获取最小价格跳动单位
func (be *BinanceExchange) GetTickSize(symbol string) (float64, error) {
	market, err := be.GetMarketInfo(symbol)
	if err != nil {
		return 0.01, err // 默认值
	}

	// 尝试从filters中获取tickSize
	if filters, ok := market["filters"].([]interface{}); ok {
		for _, f := range filters {
			if filter, ok := f.(map[string]interface{}); ok {
				if filterType, _ := filter["filterType"].(string); filterType == "PRICE_FILTER" {
					if tickSize, ok := filter["tickSize"].(string); ok {
						if ts, err := strconv.ParseFloat(tickSize, 64); err == nil {
							return ts, nil
						}
					}
				}
			}
		}
	}

	// 兜底：从precision获取
	if prec, ok := market["precision"].(map[string]interface{}); ok {
		if pricePrec, ok := prec["price"].(float64); ok {
			return 1.0 / (10.0 * pricePrec), nil
		}
	}

	return 0.01, nil // 默认值
}

// ValidatePrice 验证价格合理性
func (be *BinanceExchange) ValidatePrice(symbol string, price float64, side string) (bool, string) {
	if price <= 0 {
		return false, "价格必须大于0"
	}

	// 获取当前价格
	currentPrice, err := be.GetTickerPrice(symbol)
	if err != nil {
		return false, fmt.Sprintf("无法获取市场价格: %v", err)
	}

	if currentPrice == 0 {
		return false, "市场价格无效"
	}

	// 计算价格偏差
	priceDeviation := abs(price-currentPrice) / currentPrice * 100

	// 允许的最大偏差（5%）
	maxDeviation := 5.0
	if priceDeviation > maxDeviation {
		return false, fmt.Sprintf("价格偏差过大(%.2f%%)，疑似价格操纵", priceDeviation)
	}

	// 检查异常跳空（10%）
	gapPct := abs(price-currentPrice) / currentPrice * 100
	if gapPct > 10.0 {
		return false, fmt.Sprintf("价格异常跳空(%.2f%%)，拒绝交易", gapPct)
	}

	return true, ""
}

// abs 计算绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

