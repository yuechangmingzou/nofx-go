package tests

import (
	"testing"

	"github.com/yourusername/nofx-go/internal/exchange"
	"github.com/yourusername/nofx-go/pkg/types"
)

func TestBinanceExchange_PlaceOrder_DryRun(t *testing.T) {
	be := exchange.GetBinanceExchange()

	req := types.OrderRequest{
		Symbol:       "BTCUSDT",
		Side:         "BUY",
		PositionSide: "LONG",
		OrderType:    "LIMIT",
		Quantity:     0.001,
		Price:        floatPtr(50000.0),
	}

	order, err := be.PlaceOrder(req)
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}

	if order == nil {
		t.Fatal("Order should not be nil")
	}

	if order.Symbol != "BTCUSDT" {
		t.Errorf("Expected symbol BTCUSDT, got %s", order.Symbol)
	}

	if order.Status != "NEW" {
		t.Errorf("Expected status NEW, got %s", order.Status)
	}
}

func TestBinanceExchange_CancelOrder_DryRun(t *testing.T) {
	be := exchange.GetBinanceExchange()

	err := be.CancelOrder("BTCUSDT", "12345")
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}
}

func TestBinanceExchange_GetPositions_DryRun(t *testing.T) {
	be := exchange.GetBinanceExchange()

	positions, err := be.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}

	if positions == nil {
		t.Fatal("Positions should not be nil")
	}

	// DRY_RUN模式下应该返回空列表
	if len(positions) != 0 {
		t.Errorf("Expected empty positions in DRY_RUN mode, got %d", len(positions))
	}
}

func TestBinanceExchange_GenerateSignature(t *testing.T) {
	be := exchange.GetBinanceExchange()

	queryString := "symbol=BTCUSDT&side=BUY&type=LIMIT&quantity=0.001&timestamp=1234567890"
	signature := be.GenerateSignature(queryString)

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	if len(signature) != 64 {
		t.Errorf("Signature should be 64 characters (SHA256 hex), got %d", len(signature))
	}
}

func TestBinanceExchange_GetBalance_DryRun(t *testing.T) {
	be := exchange.GetBinanceExchange()

	balance, err := be.GetBalance()
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance == nil {
		t.Fatal("Balance should not be nil")
	}

	// DRY_RUN模式下应该有默认余额
	if balance["total"] == 0 {
		t.Error("Expected non-zero balance in DRY_RUN mode")
	}
}

func floatPtr(f float64) *float64 {
	return &f
}

