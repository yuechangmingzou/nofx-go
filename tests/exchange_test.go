package tests

import (
	"testing"

	"github.com/yuechangmingzou/nofx-go/internal/exchange"
)

func TestRateLimiter(t *testing.T) {
	rl := exchange.NewRateLimiter(10.0, 20)

	// 测试获取令牌
	if !rl.Acquire(1) {
		t.Error("Should be able to acquire token immediately")
	}

	// 测试等待
	rl.Wait(1) // 应该立即返回（有足够令牌）
}

func TestBackoffManager(t *testing.T) {
	bm := exchange.GetGlobalBackoff()

	// 测试设置退避
	bm.SetBackoff("test", 1.0)

	// 测试等待退避
	bm.WaitBackoff("test") // 应该等待约1秒

	// 测试重置
	bm.ResetBackoff("test")
	bm.WaitBackoff("test") // 应该立即返回
}

func TestParseRetryAfter(t *testing.T) {
	// 测试秒数格式
	retryAfter := exchange.ParseRetryAfter("60")
	if retryAfter == nil || *retryAfter != 60.0 {
		t.Errorf("Expected 60.0, got %v", retryAfter)
	}

	// 测试空字符串
	retryAfter = exchange.ParseRetryAfter("")
	if retryAfter != nil {
		t.Error("Expected nil for empty string")
	}
}

func TestBinanceExchange_GetOHLCV(t *testing.T) {
	// 注意：这个测试需要网络连接，可能在实际环境中失败
	// 在CI/CD中应该标记为集成测试
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	be := exchange.GetBinanceExchange()
	ohlcv, err := be.GetOHLCV("BTCUSDT", "1m", 10)
	if err != nil {
		t.Logf("GetOHLCV failed (may be network issue): %v", err)
		return
	}

	if len(ohlcv) == 0 {
		t.Error("Expected at least one OHLCV data point")
	}

	// 验证数据结构
	first := ohlcv[0]
	if first.Open <= 0 || first.High <= 0 || first.Low <= 0 || first.Close <= 0 {
		t.Error("OHLCV data should have positive values")
	}
}

func TestBinanceExchange_NormalizeSymbol(t *testing.T) {
	be := exchange.GetBinanceExchange()

	tests := []struct {
		input    string
		expected string
	}{
		{"BTCUSDT", "BTCUSDT"},
		{"BTC/USDT", "BTCUSDT"},
		{"BTC-USDT", "BTCUSDT"},
		{"btc", "BTCUSDT"},
		{"ETH", "ETHUSDT"},
	}

	for _, tt := range tests {
		result := be.NormalizeSymbol(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizeSymbol(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

