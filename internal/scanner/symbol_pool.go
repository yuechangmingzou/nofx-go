package scanner

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/exchange"
	"github.com/yourusername/nofx-go/internal/utils"
)

// GetSymbolPool 获取币种池
func (s *Scanner) GetSymbolPool(forceFull bool) ([]string, error) {
	logger := utils.GetLogger("scanner")

	// 优先从波动率池获取（波动最大的20个币种）
	if !forceFull {
		volatilityPool := s.getVolatilityPool()
		if len(volatilityPool) > 0 {
			logger.Infow("Using volatility pool",
				"count", len(volatilityPool),
			)
			return volatilityPool, nil
		}
	}

	// 如果波动率池为空或forceFull=true，获取所有USDT交易对
	allSymbols, err := exchange.GetUSDTSymbols()
	if err != nil {
		return nil, fmt.Errorf("failed to get USDT symbols: %w", err)
	}

	cfg := config.Get()
	// 过滤：只保留上市时间≥N天的合约
	filteredSymbols := s.filterSymbolsByOnlineDays(allSymbols, cfg.BinanceMinOnlineDays)

	logger.Infow("Using full symbol pool",
		"total", len(allSymbols),
		"filtered", len(filteredSymbols),
	)

	return filteredSymbols, nil
}

// getVolatilityPool 获取波动率池
func (s *Scanner) getVolatilityPool() []string {
	key := config.GetRedisKey("volatility_pool")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	members, err := s.redis.SMembers(ctx, key).Result()
	if err != nil {
		return []string{}
	}

	return members
}

// filterSymbolsByOnlineDays 过滤币种（只保留上市时间≥N天的）
func (s *Scanner) filterSymbolsByOnlineDays(symbols []string, minDays int) []string {
	if minDays <= 0 {
		return symbols
	}

	exchange := exchange.GetBinanceExchange()
	now := time.Now().Unix() * 1000 // 毫秒时间戳
	minOnlineMs := int64(minDays) * 24 * 60 * 60 * 1000

	filtered := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		// 从交易所获取市场信息
		marketInfo, err := exchange.GetMarketInfo(symbol)
		if err != nil {
			// 如果获取失败，默认保留（保守策略）
			filtered = append(filtered, symbol)
			continue
		}

		// 检查onboardDate
		onboardDate, ok := marketInfo["onboardDate"]
		if !ok || onboardDate == nil {
			// 如果没有onboardDate，默认保留
			filtered = append(filtered, symbol)
			continue
		}

		// 解析onboardDate（可能是字符串、数字或时间戳）
		var onboardTs int64
		switch v := onboardDate.(type) {
		case float64:
			onboardTs = int64(v)
		case int64:
			onboardTs = v
		case string:
			if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
				onboardTs = ts
			} else {
				// 解析失败，默认保留
				filtered = append(filtered, symbol)
				continue
			}
		default:
			// 未知类型，默认保留
			filtered = append(filtered, symbol)
			continue
		}

		// 检查是否满足最小上市天数
		if (now - onboardTs) >= minOnlineMs {
			filtered = append(filtered, symbol)
		}
	}

	return filtered
}

// UpdateSymbolPool 更新币种池
func (s *Scanner) UpdateSymbolPool(activeSymbols []string) error {
	if len(activeSymbols) == 0 {
		return nil
	}

	cfg := config.Get()
	key := config.GetRedisKey("symbol_pool")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 添加到集合
	if len(activeSymbols) > 0 {
		pipe := s.redis.Pipeline()
		pipe.SAdd(ctx, key, activeSymbols)
		ttl := time.Duration(cfg.SymbolPoolTTLSec) * time.Second
		pipe.Expire(ctx, key, ttl)
		_, err := pipe.Exec(ctx)
		return err
	}

	return nil
}

// CalculateVolatility 计算币种的波动率（基于24h涨跌幅的绝对值）
func (s *Scanner) CalculateVolatility(symbol string) float64 {
	ticker, err := exchange.GetBinanceExchange().GetTicker24h(symbol)
	if err != nil {
		return 0.0
	}

	if priceChangePct, ok := ticker["priceChangePercent"].(float64); ok {
		return math.Abs(priceChangePct)
	}

	return 0.0
}

// UpdateVolatilityPool 更新波动率池
func (s *Scanner) UpdateVolatilityPool() ([]string, error) {
	logger := utils.GetLogger("scanner")
	cfg := config.Get()

	// 获取所有USDT币种
	allSymbols, err := exchange.GetUSDTSymbols()
	if err != nil {
		return nil, fmt.Errorf("failed to get USDT symbols: %w", err)
	}

	logger.Infow("Calculating volatility",
		"total_symbols", len(allSymbols),
	)

	// 并发计算所有币种的波动率
	type volResult struct {
		symbol string
		vol    float64
	}

	volChan := make(chan volResult, len(allSymbols))
	var wg sync.WaitGroup

	// 限制并发数，避免触发API限流
	maxConcurrency := 20
	sem := make(chan struct{}, maxConcurrency)

	for _, symbol := range allSymbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			sem <- struct{}{} // 获取信号量
			defer func() { <-sem }() // 释放信号量

			vol := s.CalculateVolatility(sym)
			volChan <- volResult{symbol: sym, vol: vol}
		}(symbol)
	}

	go func() {
		wg.Wait()
		close(volChan)
	}()

	// 收集结果
	symbolVolatility := make([]volResult, 0, len(allSymbols))
	for result := range volChan {
		symbolVolatility = append(symbolVolatility, result)
	}

	// 按波动率降序排序，取前N个
	sort.Slice(symbolVolatility, func(i, j int) bool {
		return symbolVolatility[i].vol > symbolVolatility[j].vol
	})

	poolSize := cfg.AIBatchSize * 10 // 默认20个
	if poolSize > len(symbolVolatility) {
		poolSize = len(symbolVolatility)
	}

	topSymbols := make([]string, 0, poolSize)
	for i := 0; i < poolSize; i++ {
		topSymbols = append(topSymbols, symbolVolatility[i].symbol)
	}

	// 更新Redis缓存
	key := config.GetRedisKey("volatility_pool")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := s.redis.Pipeline()
	pipe.Del(ctx, key)
	if len(topSymbols) > 0 {
		pipe.SAdd(ctx, key, topSymbols)
	}
	ttl := time.Duration(cfg.SymbolPoolTTLSec) * time.Second
	pipe.Expire(ctx, key, ttl)
	_, err = pipe.Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update volatility pool: %w", err)
	}

	logger.Infow("Volatility pool updated",
		"count", len(topSymbols),
		"top_volatility", symbolVolatility[0].vol,
	)

	return topSymbols, nil
}

