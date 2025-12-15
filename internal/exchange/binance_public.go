package exchange

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// GetUSDTSymbols 获取所有USDT交易对（公开方法）
func GetUSDTSymbols() ([]string, error) {
	be := GetBinanceExchange()
	return be.getUSDTSymbols()
}

// getUSDTSymbols 获取所有USDT交易对（内部方法）
func (be *BinanceExchange) getUSDTSymbols() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/exchangeInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}

	var symbols []string
	if dataMap, ok := data.(map[string]interface{}); ok {
		if symbolsList, ok := dataMap["symbols"].([]interface{}); ok {
			for _, sym := range symbolsList {
				if symMap, ok := sym.(map[string]interface{}); ok {
					// 只获取USDT永续合约
					if quote, ok := symMap["quoteAsset"].(string); ok && quote == "USDT" {
						if contractType, ok := symMap["contractType"].(string); ok && contractType == "PERPETUAL" {
							if symbol, ok := symMap["symbol"].(string); ok {
								symbols = append(symbols, symbol)
							}
						}
					}
				}
			}
		}
	}

	return symbols, nil
}

// GetTicker24h 获取24小时Ticker数据
func (be *BinanceExchange) GetTicker24h(symbol string) (map[string]interface{}, error) {
	symbol = be.normalizeSymbol(symbol)

	params := map[string]string{
		"symbol": symbol,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/ticker/24hr", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker: %w", err)
	}

	if dataMap, ok := data.(map[string]interface{}); ok {
		return dataMap, nil
	}

	return nil, fmt.Errorf("invalid ticker data format")
}

// GetOpenInterestHistChange 获取持仓量历史变化
func (be *BinanceExchange) GetOpenInterestHistChange(symbol string, period string, limit int) ([]map[string]interface{}, error) {
	symbol = be.normalizeSymbol(symbol)

	params := map[string]string{
		"symbol": symbol,
		"period": period, // 5m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 1d
		"limit":  strconv.Itoa(limit),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/openInterestHist", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get open interest hist: %w", err)
	}

	if dataList, ok := data.([]interface{}); ok {
		result := make([]map[string]interface{}, 0, len(dataList))
		for _, item := range dataList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("invalid open interest hist data format")
}

