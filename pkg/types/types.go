package types

// MarketData 市场数据
type MarketData struct {
	Symbol            string  `json:"symbol"`
	CurrentPrice      float64 `json:"current_price"`
	PriceChangePct24h float64 `json:"price_change_pct_24h"`
	OpenInterest      float64 `json:"open_interest"`
	OpenInterestChange float64 `json:"open_interest_change"`
	FundingRate       float64 `json:"funding_rate"`
	Volume            float64 `json:"volume"`
	Volume24h         float64 `json:"volume_24h"`
	Timestamp         int64   `json:"timestamp"`
	
	// K线数据
	OHLCV1m  []OHLCV `json:"ohlcv_1m,omitempty"`
	OHLCV3m  []OHLCV `json:"ohlcv_3m,omitempty"`
	OHLCV5m  []OHLCV `json:"ohlcv_5m,omitempty"`
	OHLCV15m []OHLCV `json:"ohlcv_15m,omitempty"`
	OHLCV30m []OHLCV `json:"ohlcv_30m,omitempty"`
	OHLCV1h  []OHLCV `json:"ohlcv_1h,omitempty"`
	OHLCV4h  []OHLCV `json:"ohlcv_4h,omitempty"`
	OHLCV1d  []OHLCV `json:"ohlcv_1d,omitempty"`
	
	// 技术指标
	EMA20    float64 `json:"ema_20,omitempty"`
	EMA50    float64 `json:"ema_50,omitempty"`
	EMA200   float64 `json:"ema_200,omitempty"`
	RSI      float64 `json:"rsi,omitempty"`
	BB       *BollingerBands `json:"bb,omitempty"`
	CVD      float64 `json:"cvd,omitempty"`
	OBV      float64 `json:"obv,omitempty"`

	// 预过滤字段
	VolumePeakRatio  float64 `json:"volume_peak_ratio,omitempty"`
	ConsecutiveCount int     `json:"consecutive_count,omitempty"`
	
	// 账户信息（可选，用于AI决策）
	Account *AccountInfo `json:"account,omitempty"`
}

// AccountInfo 账户信息
type AccountInfo struct {
	Balance   map[string]float64        `json:"balance,omitempty"`
	Positions []map[string]interface{}  `json:"positions,omitempty"`
}

// OHLCV K线数据
type OHLCV struct {
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
	Time   int64   `json:"time"`
}

// BollingerBands 布林带
type BollingerBands struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
	Squeeze bool   `json:"squeeze"`
}

// Signal 交易信号
type Signal struct {
	Symbol       string  `json:"symbol"`
	Action       string  `json:"action"` // open_long, open_short, close_long, close_short, hold, wait
	Side         string  `json:"side"`   // long, short
	EntryPrice   float64 `json:"entry_price,omitempty"`
	StopLoss     float64 `json:"stop_loss,omitempty"`
	TakeProfit   float64 `json:"take_profit,omitempty"`
	TakeProfit2  float64 `json:"take_profit_2,omitempty"` // 二级止盈
	Quantity     float64 `json:"quantity,omitempty"`
	Leverage     int     `json:"leverage,omitempty"`
	Reason       string  `json:"reason,omitempty"`
	SignalID     string  `json:"signal_id,omitempty"` // 唯一信号ID
	Timestamp    int64   `json:"timestamp"`
}

// Order 订单
type Order struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"` // BUY, SELL
	PositionSide  string  `json:"position_side"` // LONG, SHORT
	OrderType     string  `json:"order_type"` // LIMIT, MARKET, STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price,omitempty"`
	StopPrice     float64 `json:"stop_price,omitempty"`
	Status        string  `json:"status"` // NEW, FILLED, CANCELED, REJECTED
	FilledQty     float64 `json:"filled_qty"`
	AvgPrice      float64 `json:"avg_price,omitempty"`
	ReduceOnly    bool    `json:"reduce_only,omitempty"`
	Timestamp     int64   `json:"timestamp"`
}

// Position 持仓
type Position struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"` // LONG, SHORT
	Size         float64 `json:"size"`
	EntryPrice   float64 `json:"entry_price"`
	MarkPrice    float64 `json:"mark_price"`
	UnrealizedPnl float64 `json:"unrealized_pnl"`
	Leverage     int     `json:"leverage"`
}

// Exchange 交易所接口
type Exchange interface {
	// 获取K线数据
	GetOHLCV(symbol, timeframe string, limit int) ([]OHLCV, error)
	
	// 下单
	PlaceOrder(order OrderRequest) (*Order, error)
	
	// 取消订单
	CancelOrder(symbol, orderID string) error
	
	// 查询订单
	GetOrder(symbol, orderID string) (*Order, error)
	
	// 查询持仓
	GetPosition(symbol string) (*Position, error)
	
	// 查询所有持仓
	GetPositions() ([]*Position, error)
	
	// 获取当前价格
	GetTickerPrice(symbol string) (float64, error)
	
	// 获取资金费率
	GetFundingRate(symbol string) (float64, error)
	
	// 获取持仓量
	GetOpenInterest(symbol string) (float64, error)
	
	// 获取账户余额（可选）
	GetBalance() (map[string]float64, error)
	
	// 获取当前挂单
	GetOpenOrders(symbol string) ([]*Order, error)
}

// OrderRequest 订单请求
type OrderRequest struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"` // BUY, SELL
	PositionSide string  `json:"position_side"` // LONG, SHORT
	OrderType    string  `json:"order_type"` // LIMIT, MARKET, STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET
	Quantity     float64 `json:"quantity"`
	Price        *float64 `json:"price,omitempty"`
	StopPrice    *float64 `json:"stop_price,omitempty"`
	ReduceOnly   bool    `json:"reduce_only,omitempty"`
	TimeInForce  string  `json:"time_in_force,omitempty"` // GTC, IOC, FOK
}

