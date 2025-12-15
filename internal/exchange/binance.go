package exchange

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/pkg/types"
	"github.com/yuechangmingzou/nofx-go/internal/utils"
)

// BinanceExchange Binance交易所实现
type BinanceExchange struct {
	client    *HTTPClient
	cache     map[string]cacheEntry
	cacheMu   sync.RWMutex
	markets   map[string]interface{}
	marketsMu sync.RWMutex
}

type cacheEntry struct {
	data      interface{}
	timestamp time.Time
}

var globalBinanceExchange *BinanceExchange

// GetBinanceExchange 获取Binance交易所实例（单例）
func GetBinanceExchange() *BinanceExchange {
	if globalBinanceExchange == nil {
		globalBinanceExchange = &BinanceExchange{
			client:  GetHTTPClient(),
			cache:   make(map[string]cacheEntry),
			markets: make(map[string]interface{}),
		}
		globalBinanceExchange.loadMarkets()
	}
	return globalBinanceExchange
}

// loadMarkets 加载市场信息
func (be *BinanceExchange) loadMarkets() error {
	ctx, cancel := utils.WithMediumTimeout(context.Background())
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/exchangeInfo", nil)
	if err != nil {
		return fmt.Errorf("failed to load markets: %w", err)
	}

	// 解析markets数据
	if dataMap, ok := data.(map[string]interface{}); ok {
		if symbols, ok := dataMap["symbols"].([]interface{}); ok {
			be.marketsMu.Lock()
			be.markets = make(map[string]interface{})
			for _, sym := range symbols {
				if symMap, ok := sym.(map[string]interface{}); ok {
					if symbol, ok := symMap["symbol"].(string); ok {
						be.markets[symbol] = symMap
					}
				}
			}
			be.marketsMu.Unlock()

			logger := utils.GetLogger("exchange")
			logger.Infow("Markets loaded", "count", len(be.markets))
		}
	}

	return nil
}

// GetOHLCV 实现Exchange接口
func (be *BinanceExchange) GetOHLCV(symbol, timeframe string, limit int) ([]types.OHLCV, error) {
	// 规范化symbol
	symbol = be.normalizeSymbol(symbol)

	// 检查缓存
	cacheKey := fmt.Sprintf("ohlcv:%s:%s:%d", symbol, timeframe, limit)
	if cached := be.getCache(cacheKey); cached != nil {
		if ohlcv, ok := cached.([]types.OHLCV); ok {
			return ohlcv, nil
		}
	}

	// 构建参数
	params := map[string]string{
		"symbol":   symbol,
		"interval": timeframe,
		"limit":    strconv.Itoa(limit),
	}

	ctx, cancel := utils.WithMediumTimeout(context.Background())
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/klines", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get OHLCV: %w", err)
	}

	// 解析K线数据
	klines, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid klines data format")
	}

	result := make([]types.OHLCV, 0, len(klines))
	for _, k := range klines {
		if kline, ok := k.([]interface{}); ok && len(kline) >= 6 {
			open, err := parseFloatValue(kline[1])
			if err != nil {
				continue // 跳过解析失败的K线
			}
			high, err := parseFloatValue(kline[2])
			if err != nil {
				continue
			}
			low, err := parseFloatValue(kline[3])
			if err != nil {
				continue
			}
			closePrice, err := parseFloatValue(kline[4])
			if err != nil {
				continue
			}
			volume, err := parseFloatValue(kline[5])
			if err != nil {
				continue
			}
			timeMs, err := parseFloatValue(kline[0])
			if err != nil {
				continue
			}

			result = append(result, types.OHLCV{
				Open:   open,
				High:   high,
				Low:    low,
				Close:  closePrice,
				Volume: volume,
				Time:   int64(timeMs),
			})
		}
	}

	// 缓存结果
	be.setCache(cacheKey, result)

	return result, nil
}

// GetTickerPrice 获取当前价格
func (be *BinanceExchange) GetTickerPrice(symbol string) (float64, error) {
	symbol = be.normalizeSymbol(symbol)

	params := map[string]string{
		"symbol": symbol,
	}

	ctx, cancel := utils.WithMediumTimeout(context.Background())
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/ticker/price", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get ticker price: %w", err)
	}

	if dataMap, ok := data.(map[string]interface{}); ok {
		price, err := parseStringOrFloat(dataMap, "price")
		if err != nil {
			return 0, fmt.Errorf("failed to get ticker price: %w", err)
		}
		return price, nil
	}

	return 0, fmt.Errorf("invalid ticker data format")
}

// GetFundingRate 获取资金费率
func (be *BinanceExchange) GetFundingRate(symbol string) (float64, error) {
	symbol = be.normalizeSymbol(symbol)

	params := map[string]string{
		"symbol": symbol,
	}

	ctx, cancel := utils.WithMediumTimeout(context.Background())
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/premiumIndex", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get funding rate: %w", err)
	}

	if dataMap, ok := data.(map[string]interface{}); ok {
		rate, err := parseStringOrFloat(dataMap, "lastFundingRate")
		if err != nil {
			return 0, fmt.Errorf("failed to get funding rate: %w", err)
		}
		return rate, nil
	}

	return 0, fmt.Errorf("invalid funding rate data format")
}

// GetOpenInterest 获取持仓量
func (be *BinanceExchange) GetOpenInterest(symbol string) (float64, error) {
	symbol = be.normalizeSymbol(symbol)

	params := map[string]string{
		"symbol": symbol,
	}

	ctx, cancel := utils.WithMediumTimeout(context.Background())
	defer cancel()

	data, err := be.client.FetchJSON(ctx, "/fapi/v1/openInterest", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get open interest: %w", err)
	}

	if dataMap, ok := data.(map[string]interface{}); ok {
		oi, err := parseStringOrFloat(dataMap, "openInterest")
		if err != nil {
			return 0, fmt.Errorf("failed to get open interest: %w", err)
		}
		return oi, nil
	}

	return 0, fmt.Errorf("invalid open interest data format")
}

// GetMarketInfo 获取市场信息
func (be *BinanceExchange) GetMarketInfo(symbol string) (map[string]interface{}, error) {
	symbol = be.normalizeSymbol(symbol)
	
	be.marketsMu.RLock()
	defer be.marketsMu.RUnlock()
	
	if market, ok := be.markets[symbol]; ok {
		if marketMap, ok := market.(map[string]interface{}); ok {
			return marketMap, nil
		}
	}
	
	return nil, fmt.Errorf("market info not found for symbol: %s", symbol)
}

// GetBalance 已在binance_account.go中实现

// 实现types.Exchange接口
// 注意：PlaceOrder, CancelOrder, GetOrder, GetPosition, GetPositions 已在 binance_orders.go 中实现

// NormalizeSymbol 规范化交易对符号（公开方法，供测试使用）
func (be *BinanceExchange) NormalizeSymbol(symbol string) string {
	return be.normalizeSymbol(symbol)
}

// normalizeSymbol 规范化交易对符号（内部方法）
func (be *BinanceExchange) normalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	symbol = strings.ReplaceAll(symbol, "/", "")
	symbol = strings.ReplaceAll(symbol, "-", "")
	symbol = strings.ReplaceAll(symbol, "_", "")
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}
	return symbol
}

// getCache 获取缓存
func (be *BinanceExchange) getCache(key string) interface{} {
	be.cacheMu.RLock()
	defer be.cacheMu.RUnlock()

	entry, exists := be.cache[key]
	if !exists {
		return nil
	}

	cfg := config.Get()
	ttl := time.Duration(cfg.ExchangeCacheTTLSec) * time.Second
	if time.Since(entry.timestamp) > ttl {
		// 缓存过期，删除
		be.cacheMu.RUnlock()
		be.cacheMu.Lock()
		delete(be.cache, key)
		be.cacheMu.Unlock()
		be.cacheMu.RLock()
		return nil
	}

	return entry.data
}

// setCache 设置缓存
func (be *BinanceExchange) setCache(key string, data interface{}) {
	be.cacheMu.Lock()
	defer be.cacheMu.Unlock()

	be.cache[key] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// parseFloatValue 使用utils包中的ParseFloatValue
func parseFloatValue(v interface{}) (float64, error) {
	return utils.ParseFloatValue(v)
}

func parseStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// parseBoolValue 使用utils包中的ParseBoolValue
func parseBoolValue(v interface{}) (bool, error) {
	return utils.ParseBoolValue(v)
}

