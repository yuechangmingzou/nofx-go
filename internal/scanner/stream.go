package scanner

import (
	"context"
	"fmt"
	"sync"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/utils"
	"github.com/yourusername/nofx-go/pkg/types"
)

// ScanMarketStream 流式扫描市场
func (s *Scanner) ScanMarketStream(ctx context.Context, forceFull bool) (<-chan *types.MarketData, error) {
	logger := utils.GetLogger("scanner")
	cfg := config.Get()

	// 获取币种池
	symbols, err := s.GetSymbolPool(forceFull)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol pool: %w", err)
	}

	if len(symbols) == 0 {
		logger.Warn("No symbols to scan")
		return nil, fmt.Errorf("no symbols to scan")
	}

	snapTTL := cfg.MarketSnapshotTTLSec
	scanConc := cfg.ScanConcurrency

	logger.Infow("Starting stream scan",
		"symbol_count", len(symbols),
		"force_full", forceFull,
		"concurrency", scanConc,
		"snapshot_ttl", snapTTL,
	)

	// 创建结果通道
	resultChan := make(chan *types.MarketData, scanConc*2)

	// 使用goroutine池进行并发扫描
	sem := make(chan struct{}, scanConc)
	var wg sync.WaitGroup

	go func() {
		defer close(resultChan)

		for _, symbol := range symbols {
			// 检查上下文是否取消
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			go func(sym string) {
				defer wg.Done()

				// 获取信号量
				sem <- struct{}{}
				defer func() { <-sem }()

				// 扫描币种
				data, err := s.ScanSymbol(ctx, sym)
				if err != nil {
					logger.Debugw("Scan symbol failed",
						"symbol", sym,
						"error", err,
					)
					return
				}

				if data != nil {
					select {
					case resultChan <- data:
					case <-ctx.Done():
						return
					}
				}
			}(symbol)
		}

		// 等待所有goroutine完成
		wg.Wait()
	}()

	return resultChan, nil
}

// ScanMarketStreamSync 同步版本的流式扫描（返回所有结果）
func (s *Scanner) ScanMarketStreamSync(ctx context.Context, forceFull bool) ([]*types.MarketData, error) {
	stream, err := s.ScanMarketStream(ctx, forceFull)
	if err != nil {
		return nil, err
	}

	results := make([]*types.MarketData, 0)
	activeSymbols := make([]string, 0)

	for data := range stream {
		if data != nil {
			results = append(results, data)
			if data.Symbol != "" {
				activeSymbols = append(activeSymbols, data.Symbol)
			}
		}
	}

	// 更新币种池
	if len(activeSymbols) > 0 {
		_ = s.UpdateSymbolPool(activeSymbols)
	}

	return results, nil
}
