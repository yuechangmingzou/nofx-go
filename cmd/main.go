package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/yourusername/nofx-go/internal/bot"
	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/metrics"
	"github.com/yourusername/nofx-go/internal/scanner"
	"github.com/yourusername/nofx-go/internal/utils"
	"github.com/yourusername/nofx-go/internal/web"
	"github.com/yourusername/nofx-go/pkg/types"
	"go.uber.org/zap"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// 加载配置
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置
	config.ValidateAndExit()

	cfg := config.Get()

	// 初始化日志
	if err := utils.InitLogger(cfg.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger := utils.GetLogger("main")

	// 初始化Redis
	_ = utils.GetRedisClient()
	defer utils.CloseRedisClient()

	logger.Infow("🚀 NOFX Go版本启动",
		"redis_host", cfg.RedisHost,
		"redis_port", cfg.RedisPort,
		"dry_run", cfg.DryRun,
		"log_level", cfg.LogLevel,
	)

	// 创建主上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 等待组，用于等待所有goroutine完成
	var wg sync.WaitGroup

	// 启动扫描器
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Errorw("扫描器panic", "error", r)
			}
		}()
		runScanner(ctx, logger)
	}()

	// 启动交易机器人
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Errorw("交易机器人panic", "error", r)
			}
		}()
		runBot(ctx, logger)
	}()

	// TODO: 启动指标采集器
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			logger.Errorw("指标采集器panic", "error", r)
	// 		}
	// 	}()
	// 	runMetricsCollector(ctx, logger)
	// }()

	// 启动Web服务
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Errorw("Web服务panic", "error", r)
			}
		}()
		runWebServer(ctx, logger)
	}()

	// 启动性能指标收集器
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Errorw("指标收集器panic", "error", r)
			}
		}()
		metrics.StartCollector(ctx)
	}()

	// 启动配置优化器
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Errorw("配置优化器panic", "error", r)
			}
		}()
		// 初始化Redis客户端（避免循环导入）
		optimizer := config.GetOptimizer()
		if adapter, ok := optimizer.GetRedisAdapter(); ok {
			adapter.SetClient(utils.GetRedisClient())
		}
		config.StartOptimizer(ctx)
	}()

	logger.Info("✅ 所有服务已启动")

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	logger.Info("收到停止信号，正在关闭...")

	// 取消上下文，通知所有goroutine停止
	cancel()

	// 给服务一些时间优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 等待所有goroutine完成（带超时）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("✅ 所有服务已停止")
	case <-shutdownCtx.Done():
		logger.Warn("⚠️  关闭超时，强制退出")
	}
}

// runScanner 运行扫描器
func runScanner(ctx context.Context, logger *zap.SugaredLogger) {
	logger.Info("🚀 市场扫描器启动")

	// 初始化扫描器
	sc := scanner.GetScanner()

	// 初始化交易机器人（用于处理信号）
	b, err := bot.GetBot()
	if err != nil {
		logger.Fatalw("初始化交易机器人失败", "error", err)
		return
	}

	forceFullNext := false
	cfg := config.Get()

	for {
		select {
		case <-ctx.Done():
			logger.Info("扫描器停止")
			return
		default:
		}

		t0 := time.Now()
		scannedTotal := 0
		scannedOK := 0
		anyAction := false

		// 批次投喂：每次只投喂2个币种给AI交易员
		aiBatchSize := cfg.AIBatchSize
		if aiBatchSize <= 0 {
			aiBatchSize = 2
		}

		// 流式扫描
		marketDataChan, err := sc.ScanMarketStream(ctx, forceFullNext)
		if err != nil {
			logger.Warnw("扫描市场失败", "error", err)
			time.Sleep(60 * time.Second)
			continue
		}

		// 处理市场数据（使用worker池模式）
		sem := make(chan struct{}, aiBatchSize) // 信号量限制并发
		var wg sync.WaitGroup

		for marketData := range marketDataChan {
			select {
			case <-ctx.Done():
				break
			default:
			}

			scannedTotal++
			if marketData == nil {
				continue
			}
			scannedOK++

			// 预过滤：跳过不感兴趣的市场数据
			if !shouldAnalyze(marketData) {
				continue
			}

			// 获取信号量
			sem <- struct{}{}
			wg.Add(1)

			go func(md *types.MarketData) {
				defer func() {
					<-sem // 释放信号量
					wg.Done()
					if r := recover(); r != nil {
						logger.Errorw("处理信号panic", "error", r, "symbol", md.Symbol)
					}
				}()

				action := b.ProcessSignal(ctx, md)
				if action {
					anyAction = true
				}
			}(marketData)
		}

		// 等待所有任务完成
		wg.Wait()

		// 保存扫描结果到Redis
		saveScanResult(ctx, scannedTotal, scannedOK, time.Since(t0))

		// 决定下一轮是否强制全量扫描
		// 如果扫描成功数量为0，或者所有扫描的数据都已处理，则认为已分析全部
		analyzedAll := scannedOK == 0 || scannedOK <= aiBatchSize
		forceFullNext = cfg.AIForceFullPoolWhenNoAction && !anyAction && analyzedAll

		// 等待下一个扫描周期
		interval := cfg.AIAnalysisIntervalSec
		if interval <= 0 {
			interval = cfg.ScanInterval
		}
		if interval < 10 {
			interval = 10
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(interval) * time.Second):
		}
	}
}

// runBot 运行交易机器人
func runBot(ctx context.Context, logger *zap.SugaredLogger) {
	b, err := bot.GetBot()
	if err != nil {
		logger.Fatalw("初始化交易机器人失败", "error", err)
		return
	}

	// 运行主循环（阻塞）
	if err := b.RunBot(ctx); err != nil && err != context.Canceled {
		logger.Errorw("交易机器人错误", "error", err)
	}
}

// runWebServer 运行Web服务器
func runWebServer(ctx context.Context, logger *zap.SugaredLogger) {
	server := web.GetServer()
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		logger.Errorw("Web服务器错误", "error", err)
	}
}

// shouldAnalyze 预过滤：判断是否应该分析该市场数据
func shouldAnalyze(md *types.MarketData) bool {
	cfg := config.Get()

	if !cfg.AIPrefilterEnabled {
		return true
	}

	// 检查24小时价格变化
	if abs(md.PriceChangePct24h) >= cfg.AIPrefilterMinAbsPct24h {
		return true
	}

	// 检查持仓量变化
	if abs(md.OpenInterestChange) >= cfg.AIPrefilterMinAbsOIChange {
		return true
	}

	// 检查成交量峰值
	if md.VolumePeakRatio >= cfg.AIPrefilterMinVolumePeakRatio {
		return true
	}

	// 检查连续计数
	if md.ConsecutiveCount >= cfg.AIPrefilterMinConsecutiveCount {
		return true
	}

	// 检查布林带挤压
	if md.BB != nil && md.BB.Squeeze {
		return true
	}

	return false
}

// saveScanResult 保存扫描结果到Redis
func saveScanResult(ctx context.Context, total, ok int, cost time.Duration) {
	redis := utils.GetRedisClient()
	cfg := config.Get()

	key := config.GetRedisKey("scanner_last_scan")
	payload := map[string]interface{}{
		"ts":       time.Now().Unix(),
		"cost_sec": cost.Seconds(),
		"total":    total,
		"ok":       ok,
	}

	payloadJSON, _ := json.Marshal(payload)
	ttl := time.Duration(cfg.ScanInterval*3) * time.Second
	if ttl < 60*time.Second {
		ttl = 60 * time.Second
	}
	redis.Set(ctx, key, payloadJSON, ttl)
}

// abs 返回浮点数的绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
