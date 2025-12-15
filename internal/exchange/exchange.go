package exchange

import (
	"github.com/yuechangmingzou/nofx-go/pkg/types"
)

// 确保BinanceExchange实现了types.Exchange接口（编译时检查）
var _ types.Exchange = (*BinanceExchange)(nil)

