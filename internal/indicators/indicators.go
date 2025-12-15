package indicators

import (
	"math"
)

// CalculateEMA 计算指数移动平均线
func CalculateEMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	if len(prices) == 0 {
		return 0
	}

	// 计算初始SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}

	return ema
}

// CalculateRSI 计算相对强弱指标
func CalculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // 默认中性值
	}

	gains := make([]float64, 0, len(prices)-1)
	losses := make([]float64, 0, len(prices)-1)

	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	if len(gains) < period {
		return 50.0
	}

	// 计算初始平均收益和损失
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// 计算RSI
	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}

// CalculateBollingerBands 计算布林带
func CalculateBollingerBands(prices []float64, period int, stdDev float64) (float64, float64, float64) {
	if len(prices) < period {
		if len(prices) > 0 {
			mid := prices[len(prices)-1]
			return mid, mid, mid
		}
		return 0, 0, 0
	}

	// 计算SMA
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	sma := sum / float64(period)

	// 计算标准差
	variance := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - sma
		variance += diff * diff
	}
	variance /= float64(period)
	std := math.Sqrt(variance)

	upper := sma + stdDev*std
	lower := sma - stdDev*std

	return upper, sma, lower
}

// IsBollingerSqueeze 判断是否为布林带挤压
func IsBollingerSqueeze(upper, middle, lower float64, bandwidthThreshold float64) bool {
	if middle == 0 {
		return false
	}
	bandwidth := (upper - lower) / middle
	return bandwidth < bandwidthThreshold
}

// CalculateCVD 计算累计成交量差
func CalculateCVD(ohlcv []struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}) float64 {
	cvd := 0.0

	for _, candle := range ohlcv {
		if candle.Close > candle.Open {
			// 上涨，成交量计入买入
			cvd += candle.Volume
		} else if candle.Close < candle.Open {
			// 下跌，成交量计入卖出
			cvd -= candle.Volume
		}
		// 平盘不计入
	}

	return cvd
}

// CalculateOBV 计算能量潮指标
func CalculateOBV(ohlcv []struct {
	Close  float64
	Volume float64
}) float64 {
	if len(ohlcv) == 0 {
		return 0
	}

	obv := 0.0

	for i := 1; i < len(ohlcv); i++ {
		if ohlcv[i].Close > ohlcv[i-1].Close {
			obv += ohlcv[i].Volume
		} else if ohlcv[i].Close < ohlcv[i-1].Close {
			obv -= ohlcv[i].Volume
		}
		// 价格不变，OBV不变
	}

	return obv
}

// DetectCandlePattern 检测蜡烛图形态
func DetectCandlePattern(ohlcv []struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}) string {
	if len(ohlcv) < 2 {
		return "unknown"
	}

	current := ohlcv[len(ohlcv)-1]
	prev := ohlcv[len(ohlcv)-2]

	// 计算实体和影线
	currentBody := math.Abs(current.Close - current.Open)
	currentUpperShadow := current.High - math.Max(current.Open, current.Close)
	currentLowerShadow := math.Min(current.Open, current.Close) - current.Low

	_ = math.Abs(prev.Close - prev.Open) // prevBody not used in current implementation

	// 判断是否为阳线
	isBullish := current.Close > current.Open
	isPrevBullish := prev.Close > prev.Open

	// 锤子线
	if currentLowerShadow > currentBody*2 && currentUpperShadow < currentBody*0.1 {
		if isBullish {
			return "hammer"
		}
		return "hanging_man"
	}

	// 上吊线
	if currentLowerShadow > currentBody*2 && currentUpperShadow < currentBody*0.1 && !isBullish {
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

