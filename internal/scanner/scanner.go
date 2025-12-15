package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/exchange"
	"github.com/yourusername/nofx-go/internal/indicators"
	"github.com/yourusername/nofx-go/internal/utils"
	"github.com/yourusername/nofx-go/pkg/types"
)

// Scanner 市场扫描器
type Scanner struct {
	exchange *exchange.BinanceExchange
	redis    utils.RedisClient
}

var globalScanner *Scanner

// GetScanner 获取扫描器实例（单例）
func GetScanner() *Scanner {
	if globalScanner == nil {
		globalScanner = &Scanner{
			exchange: exchange.GetBinanceExchange(),
			redis:    utils.GetRedisClient(),
		}
	}
	return globalScanner
}

// ScanSymbol 扫描单个交易对
func (s *Scanner) ScanSymbol(ctx context.Context, symbol string) (*types.MarketData, error) {
	logger := utils.GetLogger("scanner")
	cfg := config.Get()

	// 规范化symbol
	symbol = utils.NormalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("invalid symbol")
	}

	// 并发获取多周期K线数据
	type ohlcvResult struct {
		data  []types.OHLCV
		err   error
		index int
	}

	timeframes := []string{"1m", "3m", "5m", "15m", "30m", "1h", "4h", "1d"}
	limits := []int{50, 50, 50, 200, 100, 200, 200, 200}

	ohlcvResults := make([]ohlcvResult, len(timeframes))
	var wg sync.WaitGroup

	for i, tf := range timeframes {
		wg.Add(1)
		go func(idx int, timeframe string, limit int) {
			defer wg.Done()
			data, err := s.exchange.GetOHLCV(symbol, timeframe, limit)
			ohlcvResults[idx] = ohlcvResult{data: data, err: err, index: idx}
		}(i, tf, limits[i])
	}

	// 同时获取ticker、资金费率、持仓量
	var tickerPrice float64
	var fundingRate float64
	var openInterest float64
	var tickerErr error

	wg.Add(3)
	go func() {
		defer wg.Done()
		tickerPrice, tickerErr = s.exchange.GetTickerPrice(symbol)
	}()
	go func() {
		defer wg.Done()
		fundingRate, _ = s.exchange.GetFundingRate(symbol)
	}()
	go func() {
		defer wg.Done()
		openInterest, _ = s.exchange.GetOpenInterest(symbol)
	}()

	wg.Wait()

	// 检查核心周期是否成功
	if ohlcvResults[0].err != nil || len(ohlcvResults[0].data) == 0 { // 1m
		return nil, fmt.Errorf("failed to get 1m OHLCV")
	}
	if ohlcvResults[1].err != nil || len(ohlcvResults[1].data) == 0 { // 3m
		return nil, fmt.Errorf("failed to get 3m OHLCV")
	}
	if ohlcvResults[3].err != nil || len(ohlcvResults[3].data) == 0 { // 15m
		return nil, fmt.Errorf("failed to get 15m OHLCV")
	}

	// 提取价格和成交量
	prices := make(map[string][]float64)
	volumes := make(map[string][]float64)
	ohlcvMap := make(map[string][]types.OHLCV)

	for i, result := range ohlcvResults {
		if result.err == nil && len(result.data) > 0 {
			tf := timeframes[i]
			ohlcvMap[tf] = result.data
			prices[tf] = make([]float64, 0, len(result.data))
			volumes[tf] = make([]float64, 0, len(result.data))
			for _, candle := range result.data {
				prices[tf] = append(prices[tf], candle.Close)
				volumes[tf] = append(volumes[tf], candle.Volume)
			}
		}
	}

	// 获取当前价格
	currentPrice := tickerPrice
	if currentPrice == 0 && len(prices["1m"]) > 0 {
		currentPrice = prices["1m"][len(prices["1m"])-1]
	}

	// 计算24h涨跌幅（从ticker获取，如果失败则计算）
	priceChangePct24h := 0.0
	if tickerErr == nil {
		// 尝试从ticker获取24h涨跌幅
		// 这里需要调用GetTicker24h，暂时用0
	}

	// 计算技术指标
	ema20_3m := indicators.CalculateEMA(prices["3m"], cfg.IndEMAPeriod20)
	ema50_3m := indicators.CalculateEMA(prices["3m"], cfg.IndEMAPeriod50)
	ema200_1h := indicators.CalculateEMA(prices["1h"], cfg.IndEMAPeriod200)

	// 计算RSI
	rsi1h := indicators.CalculateRSI(prices["1h"], cfg.IndRSIPeriod)

	// 计算布林带
	bbUpper1h, bbMiddle1h, bbLower1h := indicators.CalculateBollingerBands(
		prices["1h"], cfg.IndBBPeriod, cfg.IndBBStdDev)

	bb1h := &types.BollingerBands{
		Upper:   bbUpper1h,
		Middle:  bbMiddle1h,
		Lower:   bbLower1h,
		Squeeze: indicators.IsBollingerSqueeze(bbUpper1h, bbMiddle1h, bbLower1h, cfg.BBSqueezeBandwidth),
	}

	// 计算CVD和OBV
	cvd1h := calculateCVD(ohlcvMap["1h"])
	obv1h := calculateOBV(ohlcvMap["1h"])

	// 计算持仓量变化
	oiChange := s.calculateOIChange(symbol, openInterest)

	// 构建市场数据
	marketData := &types.MarketData{
		Symbol:             symbol,
		CurrentPrice:       currentPrice,
		PriceChangePct24h:  priceChangePct24h,
		OpenInterest:       openInterest,
		OpenInterestChange: oiChange,
		FundingRate:        fundingRate,
		Volume:             volumes["1m"][len(volumes["1m"])-1],
		Volume24h:          0, // 需要从ticker获取
		Timestamp:          time.Now().Unix(),
		OHLCV1m:            ohlcvMap["1m"],
		OHLCV3m:            ohlcvMap["3m"],
		OHLCV5m:            ohlcvMap["5m"],
		OHLCV15m:           ohlcvMap["15m"],
		OHLCV30m:           ohlcvMap["30m"],
		OHLCV1h:            ohlcvMap["1h"],
		OHLCV4h:            ohlcvMap["4h"],
		OHLCV1d:            ohlcvMap["1d"],
		EMA20:              ema20_3m,
		EMA50:              ema50_3m,
		EMA200:             ema200_1h,
		RSI:                rsi1h,
		BB:                 bb1h,
		CVD:                cvd1h,
		OBV:                obv1h,
	}

	// 保存市场快照到Redis
	s.saveMarketSnapshot(symbol, marketData)

	logger.Debugw("Symbol scanned",
		"symbol", symbol,
		"price", currentPrice,
		"ema20", ema20_3m,
		"rsi", rsi1h,
	)

	return marketData, nil
}

// calculateCVD 计算累计成交量差
func calculateCVD(ohlcv []types.OHLCV) float64 {
	cvd := 0.0
	for _, candle := range ohlcv {
		if candle.Close > candle.Open {
			cvd += candle.Volume
		} else if candle.Close < candle.Open {
			cvd -= candle.Volume
		}
	}
	return cvd
}

// calculateOBV 计算能量潮指标
func calculateOBV(ohlcv []types.OHLCV) float64 {
	if len(ohlcv) < 2 {
		return 0
	}
	obv := 0.0
	for i := 1; i < len(ohlcv); i++ {
		if ohlcv[i].Close > ohlcv[i-1].Close {
			obv += ohlcv[i].Volume
		} else if ohlcv[i].Close < ohlcv[i-1].Close {
			obv -= ohlcv[i].Volume
		}
	}
	return obv
}

// calculateVolumePeakRatio 计算成交量峰值比率
func calculateVolumePeakRatio(volumes []float64) float64 {
	if len(volumes) < 20 {
		return 1.0
	}
	peakVolume := 0.0
	for i := len(volumes) - 20; i < len(volumes)-1; i++ {
		if volumes[i] > peakVolume {
			peakVolume = volumes[i]
		}
	}
	currentVolume := volumes[len(volumes)-1]
	if peakVolume > 0 {
		return currentVolume / peakVolume
	}
	return 1.0
}

// calculateConsecutiveCount 计算连续K线在EMA20趋势侧的数量
func (s *Scanner) calculateConsecutiveCount(ohlcv []types.OHLCV, ema20, ema50 float64) int {
	if len(ohlcv) < 6 || ema20 == 0 {
		return 0
	}

	directionHint := "long"
	if ema20 < ema50 {
		directionHint = "short"
	}

	consecutiveCount := 0
	start := len(ohlcv) - 6
	if start < 0 {
		start = 0
	}

	for i := start; i < len(ohlcv)-1; i++ {
		if directionHint == "long" && ohlcv[i].Close > ema20 {
			consecutiveCount++
		} else if directionHint == "short" && ohlcv[i].Close < ema20 {
			consecutiveCount++
		} else {
			consecutiveCount = 0
		}
	}

	return consecutiveCount
}

// detectCandlePattern 检测蜡烛图形态
func detectCandlePattern(ohlcv []types.OHLCV) string {
	if len(ohlcv) < 2 {
		return "unknown"
	}

	current := ohlcv[len(ohlcv)-1]
	prev := ohlcv[len(ohlcv)-2]

	currentBody := math.Abs(current.Close - current.Open)
	currentUpperShadow := current.High - math.Max(current.Open, current.Close)
	currentLowerShadow := math.Min(current.Open, current.Close) - current.Low

	isBullish := current.Close > current.Open
	isPrevBullish := prev.Close > prev.Open

	// 锤子线
	if currentLowerShadow > currentBody*2 && currentUpperShadow < currentBody*0.1 {
		if isBullish {
			return "hammer"
		}
		return "hanging_man"
	}

	// 吞没形态
	if isBullish && !isPrevBullish && current.Close > prev.Open && current.Open < prev.Close {
		return "bullish_engulfing"
	}
	if !isBullish && isPrevBullish && current.Close < prev.Open && current.Open > prev.Close {
		return "bearish_engulfing"
	}

	// 十字星
	if currentBody < (current.High-current.Low)*0.1 {
		return "doji"
	}

	return "normal"
}

// calculateOIChange 计算持仓量变化百分比
func (s *Scanner) calculateOIChange(symbol string, currentOI float64) float64 {
	cfg := config.Get()
	key := config.GetRedisKey(fmt.Sprintf("oi:last:%s", symbol))

	lastOIStr, err := s.redis.Get(context.Background(), key).Result()
	if err != nil {
		return 0.0
	}

	var lastOI float64
	fmt.Sscanf(lastOIStr, "%f", &lastOI)

	if lastOI > 0 {
		change := ((currentOI - lastOI) / lastOI) * 100.0
		if currentOI > 0 {
			ttl := time.Duration(cfg.OILastTTLSec) * time.Second
			s.redis.Set(context.Background(), key, fmt.Sprintf("%f", currentOI), ttl)
		}
		return change
	}

	if currentOI > 0 {
		ttl := time.Duration(cfg.OILastTTLSec) * time.Second
		s.redis.Set(context.Background(), key, fmt.Sprintf("%f", currentOI), ttl)
	}

	return 0.0
}

// saveMarketSnapshot 保存市场快照到Redis
func (s *Scanner) saveMarketSnapshot(symbol string, data *types.MarketData) {
	cfg := config.Get()
	key := config.GetRedisKey(fmt.Sprintf("market_snapshot:%s", symbol))

	// 序列化市场数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	ttl := time.Duration(cfg.MarketSnapshotTTLSec) * time.Second
	s.redis.Set(context.Background(), key, jsonData, ttl)
}
