package exchange

import (
	"github.com/yourusername/nofx-go/pkg/types"
)

// Exchange 交易所接口
// 对应 pkg/types/types.go 中的 Exchange 接口
// 这里重新定义以确保包内可用

// ExchangeInterface 交易所接口
type ExchangeInterface interface {
	// 获取K线数据
	GetOHLCV(symbol, timeframe string, limit int) ([]types.OHLCV, error)

	// 获取当前价格
	GetTickerPrice(symbol string) (float64, error)

	// 获取资金费率
	GetFundingRate(symbol string) (float64, error)

	// 获取持仓量
	GetOpenInterest(symbol string) (float64, error)
}

// 确保BinanceExchange实现了ExchangeInterface
var _ ExchangeInterface = (*BinanceExchange)(nil)

