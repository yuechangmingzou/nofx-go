package tests

import (
	"math"
	"testing"

	"github.com/yuechangmingzou/nofx-go/internal/indicators"
)

func TestCalculateEMA(t *testing.T) {
	prices := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	period := 3

	ema := indicators.CalculateEMA(prices, period)
	if ema == 0 {
		t.Error("EMA should not be zero")
	}
	if ema < 0 {
		t.Error("EMA should be positive")
	}
}

func TestCalculateEMA_EmptyPrices(t *testing.T) {
	prices := []float64{}
	period := 3

	ema := indicators.CalculateEMA(prices, period)
	if ema != 0 {
		t.Errorf("Expected EMA to be 0 for empty prices, got %f", ema)
	}
}

func TestCalculateEMA_InsufficientData(t *testing.T) {
	prices := []float64{1.0, 2.0}
	period := 3

	ema := indicators.CalculateEMA(prices, period)
	if ema != 0 {
		t.Errorf("Expected EMA to be 0 for insufficient data, got %f", ema)
	}
}

func TestCalculateRSI(t *testing.T) {
	// 测试数据：前7个价格上升，后7个价格下降
	prices := []float64{100, 102, 104, 106, 108, 110, 112, 110, 108, 106, 104, 102, 100, 98}
	period := 14

	rsi := indicators.CalculateRSI(prices, period)
	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI should be between 0 and 100, got %f", rsi)
	}
}

func TestCalculateRSI_InsufficientData(t *testing.T) {
	prices := []float64{100, 102}
	period := 14

	rsi := indicators.CalculateRSI(prices, period)
	if rsi != 50.0 {
		t.Errorf("Expected RSI to be 50.0 for insufficient data, got %f", rsi)
	}
}

func TestCalculateBollingerBands(t *testing.T) {
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120}
	period := 20
	stdDev := 2.0

	upper, middle, lower := indicators.CalculateBollingerBands(prices, period, stdDev)

	if upper <= middle {
		t.Errorf("Upper band should be greater than middle, got upper=%f, middle=%f", upper, middle)
	}
	if lower >= middle {
		t.Errorf("Lower band should be less than middle, got lower=%f, middle=%f", lower, middle)
	}
	if math.IsNaN(upper) || math.IsNaN(middle) || math.IsNaN(lower) {
		t.Error("Bollinger Bands should not be NaN")
	}
}

func TestIsBollingerSqueeze(t *testing.T) {
	// 测试挤压情况
	upper := 100.5
	middle := 100.0
	lower := 99.5
	bandwidthThreshold := 0.01

	isSqueeze := indicators.IsBollingerSqueeze(upper, middle, lower, bandwidthThreshold)
	// 带宽 = (100.5 - 99.5) / 100.0 = 0.01，应该判断为挤压
	if !isSqueeze {
		t.Error("Expected squeeze to be true")
	}

	// 测试非挤压情况
	upper2 := 102.0
	lower2 := 98.0
	isSqueeze2 := indicators.IsBollingerSqueeze(upper2, middle, lower2, bandwidthThreshold)
	if isSqueeze2 {
		t.Error("Expected squeeze to be false")
	}
}

func TestCalculateCVD(t *testing.T) {
	ohlcv := []struct {
		Open   float64
		High   float64
		Low    float64
		Close  float64
		Volume float64
	}{
		{Open: 100, High: 102, Low: 99, Close: 101, Volume: 1000}, // 上涨
		{Open: 101, High: 103, Low: 100, Close: 102, Volume: 1500}, // 上涨
		{Open: 102, High: 103, Low: 100, Close: 101, Volume: 800},  // 下跌
	}

	cvd := indicators.CalculateCVD(ohlcv)
	expected := 1000.0 + 1500.0 - 800.0 // 1700
	if math.Abs(cvd-expected) > 0.01 {
		t.Errorf("Expected CVD to be %f, got %f", expected, cvd)
	}
}

func TestCalculateOBV(t *testing.T) {
	ohlcv := []struct {
		Close  float64
		Volume float64
	}{
		{Close: 100, Volume: 1000},
		{Close: 101, Volume: 1500}, // 上涨
		{Close: 102, Volume: 1200}, // 上涨
		{Close: 101, Volume: 800},  // 下跌
		{Close: 100, Volume: 600},  // 下跌
	}

	obv := indicators.CalculateOBV(ohlcv)
	expected := 1500.0 + 1200.0 - 800.0 - 600.0 // 1300
	if math.Abs(obv-expected) > 0.01 {
		t.Errorf("Expected OBV to be %f, got %f", expected, obv)
	}
}

