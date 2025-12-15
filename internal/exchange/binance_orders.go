package exchange

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/pkg/types"
	"github.com/yourusername/nofx-go/internal/utils"
)

// PlaceOrder 下单
func (be *BinanceExchange) PlaceOrder(req types.OrderRequest) (*types.Order, error) {
	cfg := config.Get()
	if cfg.DryRun {
		// DRY_RUN模式：只记录，不下单
		logger := utils.GetLogger("exchange")
		logger.Infow("DRY_RUN: Order would be placed",
			"symbol", req.Symbol,
			"side", req.Side,
			"order_type", req.OrderType,
			"quantity", req.Quantity,
			"price", req.Price,
		)
		return &types.Order{
			ID:          "dry_run_" + strconv.FormatInt(time.Now().UnixNano(), 10),
			Symbol:      be.normalizeSymbol(req.Symbol),
			Side:        req.Side,
			PositionSide: req.PositionSide,
			OrderType:   req.OrderType,
			Status:      "NEW",
			Quantity:    req.Quantity,
			Price:       getFloatValue(req.Price),
			Timestamp:   time.Now().Unix(),
		}, nil
	}

	// 实盘下单需要API密钥和签名
	if cfg.BinanceAPIKey == "" || cfg.BinanceSecretKey == "" {
		return nil, fmt.Errorf("API keys required for real trading")
	}

	// 规范化symbol
	symbol := be.normalizeSymbol(req.Symbol)

	// 构建请求参数
	params := make(map[string]string)
	params["symbol"] = symbol
	params["side"] = strings.ToUpper(req.Side)
	params["type"] = strings.ToUpper(req.OrderType)

	// 数量
	params["quantity"] = formatFloat(req.Quantity)

	// 持仓方向（Hedge Mode）
	if req.PositionSide != "" {
		params["positionSide"] = strings.ToUpper(req.PositionSide)
	}

	// 价格（限价单需要）
	if req.Price != nil && *req.Price > 0 {
		params["price"] = formatFloat(*req.Price)
	}

	// 止损价格（STOP/STOP_MARKET需要）
	if req.StopPrice != nil && *req.StopPrice > 0 {
		params["stopPrice"] = formatFloat(*req.StopPrice)
	}

	// 时间条件
	if req.TimeInForce != "" {
		params["timeInForce"] = strings.ToUpper(req.TimeInForce)
	} else if req.OrderType == "LIMIT" {
		params["timeInForce"] = "GTC"
	}

	// ReduceOnly（平仓单）
	if req.ReduceOnly {
		params["reduceOnly"] = "true"
	}

	// 构建签名URL
	reqURL, err := be.buildSignedURL("/fapi/v1/order", params, http.MethodPost)
	if err != nil {
		return nil, fmt.Errorf("build signed URL failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

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
		return nil, fmt.Errorf("place order failed: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	var orderResp map[string]interface{}
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// 解析订单响应
	orderID := parseStringValue(orderResp["orderId"])
	status := parseStringValue(orderResp["status"])
	filledQty, _ := parseFloatValue(orderResp["executedQty"])
	avgPrice, _ := parseFloatValue(orderResp["avgPrice"])

	order := &types.Order{
		ID:           orderID,
		Symbol:       symbol,
		Side:         strings.ToUpper(req.Side),
		PositionSide: strings.ToUpper(req.PositionSide),
		OrderType:    strings.ToUpper(req.OrderType),
		Status:       status,
		Quantity:     req.Quantity,
		FilledQty:    filledQty,
		AvgPrice:     avgPrice,
		Timestamp:    time.Now().Unix(),
	}

	if req.Price != nil {
		order.Price = *req.Price
	}
	if req.StopPrice != nil {
		order.StopPrice = *req.StopPrice
	}

	return order, nil
}

// GetOpenOrders 获取当前挂单
func (be *BinanceExchange) GetOpenOrders(symbol string) ([]*types.Order, error) {
	cfg := config.Get()
	if cfg.DryRun {
		// DRY_RUN模式：返回空列表
		return []*types.Order{}, nil
	}

	if cfg.BinanceAPIKey == "" || cfg.BinanceSecretKey == "" {
		return nil, fmt.Errorf("API keys required")
	}

	symbol = be.normalizeSymbol(symbol)
	params := map[string]string{
		"symbol": symbol,
	}

	reqURL, err := be.buildSignedURL("/fapi/v1/openOrders", params, http.MethodGet)
	if err != nil {
		return nil, fmt.Errorf("build signed URL failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

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
		return nil, fmt.Errorf("get open orders failed: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	var ordersResp []map[string]interface{}
	if err := json.Unmarshal(body, &ordersResp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	orders := make([]*types.Order, 0, len(ordersResp))
	for _, o := range ordersResp {
		orderID := parseStringValue(o["orderId"])
		side := parseStringValue(o["side"])
		positionSide := parseStringValue(o["positionSide"])
		orderType := parseStringValue(o["type"])
		status := parseStringValue(o["status"])
		quantity, _ := parseFloatValue(o["origQty"])
		price, _ := parseFloatValue(o["price"])
		stopPrice, _ := parseFloatValue(o["stopPrice"])
		filledQty, _ := parseFloatValue(o["executedQty"])
		avgPrice, _ := parseFloatValue(o["avgPrice"])
		timeVal, _ := parseFloatValue(o["time"])

		orders = append(orders, &types.Order{
			ID:           orderID,
			Symbol:       symbol,
			Side:         side,
			PositionSide: positionSide,
			OrderType:    orderType,
			Status:       status,
			Quantity:     quantity,
			Price:        price,
			StopPrice:    stopPrice,
			FilledQty:    filledQty,
			AvgPrice:     avgPrice,
			Timestamp:    int64(timeVal / 1000),
		})
	}

	return orders, nil
}

// CancelOrder 取消订单
func (be *BinanceExchange) CancelOrder(orderID, symbol string) (bool, error) {
	cfg := config.Get()
	if cfg.DryRun {
		logger := utils.GetLogger("exchange")
		logger.Infow("DRY_RUN: Order would be cancelled",
			"order_id", orderID,
			"symbol", symbol,
		)
		return true, nil
	}

	if cfg.BinanceAPIKey == "" || cfg.BinanceSecretKey == "" {
		return false, fmt.Errorf("API keys required")
	}

	symbol = be.normalizeSymbol(symbol)
	params := map[string]string{
		"symbol":  symbol,
		"orderId": orderID,
	}

	reqURL, err := be.buildSignedURL("/fapi/v1/order", params, http.MethodDelete)
	if err != nil {
		return false, fmt.Errorf("build signed URL failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("X-MBX-APIKEY", cfg.BinanceAPIKey)

	resp, err := be.client.client.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("cancel order failed: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	return true, nil
}

// GetOrder 获取订单状态
func (be *BinanceExchange) GetOrder(orderID, symbol string) (*types.Order, error) {
	cfg := config.Get()
	if cfg.DryRun {
		return &types.Order{
			ID:        orderID,
			Symbol:    be.normalizeSymbol(symbol),
			Status:    "FILLED",
			Timestamp: time.Now().Unix(),
		}, nil
	}

	if cfg.BinanceAPIKey == "" || cfg.BinanceSecretKey == "" {
		return nil, fmt.Errorf("API keys required")
	}

	symbol = be.normalizeSymbol(symbol)
	params := map[string]string{
		"symbol":  symbol,
		"orderId": orderID,
	}

	reqURL, err := be.buildSignedURL("/fapi/v1/order", params, http.MethodGet)
	if err != nil {
		return nil, fmt.Errorf("build signed URL failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

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
		return nil, fmt.Errorf("get order failed: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	var orderResp map[string]interface{}
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	orderIDStr := parseStringValue(orderResp["orderId"])
	side := parseStringValue(orderResp["side"])
	positionSide := parseStringValue(orderResp["positionSide"])
	orderType := parseStringValue(orderResp["type"])
	status := parseStringValue(orderResp["status"])
	quantity, _ := parseFloatValue(orderResp["origQty"])
	price, _ := parseFloatValue(orderResp["price"])
	stopPrice, _ := parseFloatValue(orderResp["stopPrice"])
	filledQty, _ := parseFloatValue(orderResp["executedQty"])
	avgPrice, _ := parseFloatValue(orderResp["avgPrice"])
	timeVal, _ := parseFloatValue(orderResp["time"])

	return &types.Order{
		ID:           orderIDStr,
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		OrderType:    orderType,
		Status:       status,
		Quantity:     quantity,
		Price:        price,
		StopPrice:    stopPrice,
		FilledQty:    filledQty,
		AvgPrice:     avgPrice,
		Timestamp:    int64(timeVal / 1000),
	}, nil
}

// GetPosition 获取单个持仓
func (be *BinanceExchange) GetPosition(symbol string) (*types.Position, error) {
	positions, err := be.GetPositions()
	if err != nil {
		return nil, err
	}

	symbol = be.normalizeSymbol(symbol)
	for _, pos := range positions {
		if pos.Symbol == symbol {
			return pos, nil
		}
	}

	return nil, nil // 无持仓
}

// GetPositions 获取所有持仓
func (be *BinanceExchange) GetPositions() ([]*types.Position, error) {
	cfg := config.Get()
	if cfg.DryRun {
		return []*types.Position{}, nil
	}

	if cfg.BinanceAPIKey == "" || cfg.BinanceSecretKey == "" {
		return nil, fmt.Errorf("API keys required")
	}

	params := map[string]string{}

	reqURL, err := be.buildSignedURL("/fapi/v2/positionRisk", params, http.MethodGet)
	if err != nil {
		return nil, fmt.Errorf("build signed URL failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

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
		return nil, fmt.Errorf("get positions failed: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	var positionsResp []map[string]interface{}
	if err := json.Unmarshal(body, &positionsResp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	positions := make([]*types.Position, 0)
	for _, p := range positionsResp {
		size, _ := parseFloatValue(p["positionAmt"])
		if size == 0 {
			continue // 跳过空仓
		}

		entryPrice, _ := parseFloatValue(p["entryPrice"])
		markPrice, _ := parseFloatValue(p["markPrice"])
		unrealizedPnl, _ := parseFloatValue(p["unRealizedProfit"])
		leverage, _ := parseFloatValue(p["leverage"])

		side := "LONG"
		if size < 0 {
			side = "SHORT"
			size = -size
		}

		symbol, _ := p["symbol"].(string)
		positions = append(positions, &types.Position{
			Symbol:       symbol,
			Side:         side,
			Size:         size,
			EntryPrice:   entryPrice,
			MarkPrice:    markPrice,
			UnrealizedPnl: unrealizedPnl,
			Leverage:     int(leverage),
		})
	}

	return positions, nil
}

// buildSignedURL 构建带签名的URL
func (be *BinanceExchange) buildSignedURL(endpoint string, params map[string]string, method string) (string, error) {
	cfg := config.Get()

	// 添加时间戳
	params["timestamp"] = strconv.FormatInt(time.Now().Unix()*1000, 10)

	// 排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建查询字符串
	var queryParts []string
	for _, k := range keys {
		queryParts = append(queryParts, k+"="+url.QueryEscape(params[k]))
	}
	queryString := strings.Join(queryParts, "&")

	// 生成签名
	signature := be.generateSignature(queryString)
	queryString += "&signature=" + signature

	// 构建完整URL
	baseURL := cfg.BinanceFAPIBaseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	if strings.HasPrefix(endpoint, "/") {
		endpoint = endpoint[1:]
	}
	return baseURL + endpoint + "?" + queryString, nil
}

// generateSignature 生成HMAC-SHA256签名（内部方法）
func (be *BinanceExchange) generateSignature(queryString string) string {
	cfg := config.Get()
	mac := hmac.New(sha256.New, []byte(cfg.BinanceSecretKey))
	mac.Write([]byte(queryString))
	return hex.EncodeToString(mac.Sum(nil))
}

// 辅助函数
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func getFloatValue(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
