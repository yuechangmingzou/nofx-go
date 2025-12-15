package config

import (
	"fmt"
	"os"
	"strings"
)

// ValidateConfig 验证配置
func ValidateConfig() error {
	cfg := Get()
	var errors []string

	// 验证Redis配置
	if cfg.RedisHost == "" {
		errors = append(errors, "REDIS_HOST is required")
	}
	if cfg.RedisPort <= 0 || cfg.RedisPort > 65535 {
		errors = append(errors, fmt.Sprintf("REDIS_PORT must be between 1 and 65535, got %d", cfg.RedisPort))
	}

	// 验证Web认证（如果启用）
	if cfg.WebBasicAuthUser == "" {
		errors = append(errors, "WEB_BASIC_AUTH_USER is required")
	}
	if cfg.WebBasicAuthPass == "" {
		errors = append(errors, "WEB_BASIC_AUTH_PASS is required")
	}
	if cfg.WebBasicAuthPass == "change_me" {
		errors = append(errors, "WEB_BASIC_AUTH_PASS cannot be the default value 'change_me'")
	}
	if len(cfg.WebBasicAuthPass) < 8 {
		errors = append(errors, "WEB_BASIC_AUTH_PASS must be at least 8 characters")
	}

	// 验证Binance配置（如果非DRY_RUN模式）
	if !cfg.DryRun {
		if cfg.BinanceAPIKey == "" {
			errors = append(errors, "BINANCE_API_KEY is required when DRY_RUN=false")
		}
		if cfg.BinanceSecretKey == "" {
			errors = append(errors, "BINANCE_SECRET_KEY is required when DRY_RUN=false")
		}
		if len(cfg.BinanceAPIKey) < 20 {
			errors = append(errors, "BINANCE_API_KEY must be at least 20 characters")
		}
		if len(cfg.BinanceSecretKey) < 20 {
			errors = append(errors, "BINANCE_SECRET_KEY must be at least 20 characters")
		}
	}

	// 验证AI配置（如果启用AI模式）
	if cfg.AIProvider == "deepseek" && cfg.DeepSeekEnabled {
		if cfg.DeepSeekAPIKey == "" {
			errors = append(errors, "DEEPSEEK_API_KEY is required when DEEPSEEK_ENABLED=true")
		}
		if len(cfg.DeepSeekAPIKey) < 20 {
			errors = append(errors, "DEEPSEEK_API_KEY must be at least 20 characters")
		}
	}
	if cfg.AIProvider == "openai" && cfg.OpenAIEnabled {
		if cfg.OpenAIAPIKey == "" {
			errors = append(errors, "OPENAI_API_KEY is required when OPENAI_ENABLED=true")
		}
		if len(cfg.OpenAIAPIKey) < 20 {
			errors = append(errors, "OPENAI_API_KEY must be at least 20 characters")
		}
	}
	if cfg.AIProvider == "gemini" && cfg.GeminiEnabled {
		if cfg.GeminiAPIKey == "" {
			errors = append(errors, "GEMINI_API_KEY is required when GEMINI_ENABLED=true")
		}
		if len(cfg.GeminiAPIKey) < 20 {
			errors = append(errors, "GEMINI_API_KEY must be at least 20 characters")
		}
	}

	// 验证策略文件
	if cfg.StrategyFile == "" {
		errors = append(errors, "STRATEGY_FILE is required")
	}

	// 验证扫描配置
	if cfg.ScanInterval <= 0 {
		errors = append(errors, "SCAN_INTERVAL must be greater than 0")
	}
	if cfg.ScanConcurrency <= 0 {
		errors = append(errors, "SCAN_CONCURRENCY must be greater than 0")
	}
	if cfg.ScanConcurrency > 100 {
		errors = append(errors, "SCAN_CONCURRENCY should not exceed 100 (to avoid rate limiting)")
	}

	// 验证执行引擎配置
	if cfg.MaxNotionalPerTrade <= 0 {
		errors = append(errors, "MAX_NOTIONAL_PER_TRADE must be greater than 0")
	}
	if cfg.MaxLeverage <= 0 {
		errors = append(errors, "MAX_LEVERAGE must be greater than 0")
	}
	if cfg.MaxLeverage > 125 {
		errors = append(errors, "MAX_LEVERAGE should not exceed 125 (Binance maximum)")
	}
	if cfg.MaxConcurrentPositions <= 0 {
		errors = append(errors, "MAX_CONCURRENT_POSITIONS must be greater than 0")
	}

	// 验证指标参数
	if cfg.IndEMAPeriod20 <= 0 {
		errors = append(errors, "IND_EMA_PERIOD_20 must be greater than 0")
	}
	if cfg.IndEMAPeriod50 <= 0 {
		errors = append(errors, "IND_EMA_PERIOD_50 must be greater than 0")
	}
	if cfg.IndEMAPeriod200 <= 0 {
		errors = append(errors, "IND_EMA_PERIOD_200 must be greater than 0")
	}
	if cfg.IndRSIPeriod <= 0 {
		errors = append(errors, "IND_RSI_PERIOD must be greater than 0")
	}
	if cfg.IndBBPeriod <= 0 {
		errors = append(errors, "IND_BB_PERIOD must be greater than 0")
	}
	if cfg.IndBBStdDev <= 0 {
		errors = append(errors, "IND_BB_STD_DEV must be greater than 0")
	}

	// 验证Web配置
	if cfg.WebPort <= 0 || cfg.WebPort > 65535 {
		errors = append(errors, fmt.Sprintf("WEB_PORT must be between 1 and 65535, got %d", cfg.WebPort))
	}

	// 验证AI配置
	if cfg.AIAnalysisIntervalSec <= 0 {
		errors = append(errors, "AI_ANALYSIS_INTERVAL_SEC must be greater than 0")
	}
	if cfg.AIAnalysisConcurrency <= 0 {
		errors = append(errors, "AI_ANALYSIS_CONCURRENCY must be greater than 0")
	}
	if cfg.AIBatchSize <= 0 {
		errors = append(errors, "AI_BATCH_SIZE must be greater than 0")
	}

	// 验证告警配置（如果启用）
	if cfg.AlertEnabled && cfg.AlertWebhookURL == "" {
		errors = append(errors, "ALERT_WEBHOOK_URL is required when ALERT_ENABLED=true")
	}

	// 验证指标采集配置（如果启用）
	if cfg.MetricsEnabled {
		if cfg.MetricsMaxSymbols <= 0 {
			errors = append(errors, "METRICS_MAX_SYMBOLS must be greater than 0")
		}
		if cfg.MetricsConcurrency <= 0 {
			errors = append(errors, "METRICS_CONCURRENCY must be greater than 0")
		}
	}

	// 验证OI异动池配置（如果启用）
	if cfg.OIEnabled {
		if cfg.OIThreshold <= 0 {
			errors = append(errors, "OI_THRESHOLD must be greater than 0")
		}
		if cfg.OIIntervalMinutes <= 0 {
			errors = append(errors, "OI_INTERVAL_MINUTES must be greater than 0")
		}
		if cfg.OIConcurrency <= 0 {
			errors = append(errors, "OI_CONCURRENCY must be greater than 0")
		}
	}

	// 如果有错误，返回
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// ValidateAndExit 验证配置并在失败时退出
func ValidateAndExit() {
	if err := ValidateConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration validation failed:\n%v\n", err)
		os.Exit(1)
	}
}

