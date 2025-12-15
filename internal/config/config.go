package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 配置结构体
type Config struct {
	// Redis配置
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// Binance配置
	BinanceAPIKey    string
	BinanceSecretKey string
	BinanceTestnet   bool

	// Dry-run模式
	DryRun bool

	// AI提供商
	AIProvider string

	// DeepSeek配置
	DeepSeekEnabled     bool
	DeepSeekAPIKey      string
	DeepSeekBaseURL     string
	DeepSeekModel       string
	DeepSeekTemperature float64
	DeepSeekMaxTokens   int

	// OpenAI配置
	OpenAIEnabled     bool
	OpenAIAPIKey      string
	OpenAIBaseURL     string
	OpenAIModel       string
	OpenAITemperature float64
	OpenAIMaxTokens   int

	// Gemini配置
	GeminiEnabled     bool
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTemperature float64
	GeminiMaxTokens   int

	// AI通用参数
	AITemperature float64
	AIMaxTokens   int

	// AI Trader提示词
	AITraderSystemPrompt string

	// 策略文件
	StrategyFile string
	RuleStrategy string

	// 默认交易模式
	DefaultTradingMode string

	// 扫描配置
	ScanInterval         int
	PriceChangeThreshold float64
	ScanConcurrency      int

	// 市场快照配置
	MarketSnapshotTTLSec    int
	MarketSnapshotMaxAgeSec int

	// 交易信号配置
	SignalTTLSec      int
	MaxTradeQueueSize int

	// 币种池配置
	SymbolPoolTTLSec int
	OILastTTLSec     int

	// 执行引擎风控参数
	MaxNotionalPerTrade    float64
	MaxLeverage            float64
	MaxConcurrentPositions int
	SymbolCooldownSec      int
	OrderDedupeWindow      int
	BreakoutTimeoutSec     int

	// 订单审计
	OrderAuditMaxLen        int
	OrderAuditEventMaxChars int

	// SL/TP守护
	SLTPGuardIntervalSec float64
	GuardStatsTTLSec     int
	ProtectionTTLSec     int
	TP1PartialRatio      float64
	TPMatchTolerancePct  float64
	TakeProfitOrderType  string
	MaxTPDeviationPct    float64

	// 交易所配置
	ExchangeCacheTTLSec          float64
	BinanceFAPIBaseURL           string
	BinanceHTTPTimeoutSec        float64
	BinanceConnectorLimit        int
	BinanceConnectorLimitPerHost int
	BinanceRateLimitMaxSleepSec  float64
	BinanceMinOnlineDays         int

	// 策略阈值
	RSIOverbought      float64
	RSIOversold        float64
	VolumeShrinkRatio  float64
	BBSqueezeBandwidth float64

	// 指标参数
	IndEMAPeriod20  int
	IndEMAPeriod50  int
	IndEMAPeriod200 int
	IndRSIPeriod    int
	IndBBPeriod     int
	IndBBStdDev     float64
	IndCVDPeriod    int

	// 规则/AI共用策略阈值
	StratConsecutiveMin        int
	StratEMADivergenceMin      float64
	StratEMA200WallPct         float64
	StratEMA200HoldWarnPct     float64
	StratZoneTolPct            float64
	StratBreakoutVolRatio      float64
	StratSqueezeRejectVolRatio float64
	StratOIDropRejectPct       float64
	StratMinProfitPct          float64
	StratMinRR                 float64
	StratSLEMA50BufferPct      float64
	StratBreakevenPct          float64
	StratTP2RMult              float64
	StratTP2FallbackPct        float64
	StratDefaultNotionalUSDT   float64

	// WebSocket token
	WSTokenTTLSec int

	// AI批量分析
	AIAnalysisIntervalSec       int
	AIAnalysisConcurrency       int
	AIBatchSize                 int
	AIForceFullPoolWhenNoAction bool
	AIStatsTTLSec               int

	// AI预过滤
	AIPrefilterEnabled             bool
	AIPrefilterMinAbsPct24h        float64
	AIPrefilterMinAbsOIChange      float64
	AIPrefilterMinVolumePeakRatio  float64
	AIPrefilterMinConsecutiveCount int

	// AI历史
	DeepSeekHistoryMaxLen   int
	AIRequestHistoryMaxLen  int
	AIResponseHistoryMaxLen int
	AIDecisionHistoryMaxLen int
	SignalHistoryMaxLen     int
	TradeHistoryMaxLen      int

	// 告警推送
	AlertEnabled        bool
	AlertWebhookURL     string
	AlertDedupeTTLSec   int
	AlertMinIntervalSec int

	// 指标采集
	MetricsEnabled                   bool
	MetricsSymbolSource              string
	MetricsMaxSymbols                int
	MetricsSymbols                   string
	MetricsTimeframes                string
	MetricsGlobalRefreshSec          int
	MetricsGlobalTTLSec              int
	MetricsSymbolTTLSec              int
	MetricsHTTPTimeoutSec            float64
	MetricsHTTPConnectorLimit        int
	MetricsHTTPConnectorLimitPerHost int
	MetricsConcurrency               int
	MetricsOHLCVLimit                int
	MetricsForceOrdersLimit          int

	// 公开数据源
	CoinGeckoBaseURL string

	// 链上指标
	GlassnodeAPIKey string

	// 巨鲸转账
	WhaleAlertAPIKey      string
	WhaleAlertCurrencies  string
	WhaleAlertMinValueUSD float64
	WhaleAlertLookbackSec int
	WhaleAlertLimit       int

	// OI异动池
	OIEnabled         bool
	OIThreshold       float64
	OIIntervalMinutes int
	OIExpireMinutes   int
	OIConcurrency     int
	OIUseWhitelist    bool
	OIWhitelist       []string

	// Web配置
	WebPort               int
	WebBasicAuthUser      string
	WebBasicAuthPass      string
	WebStaticDir          string
	WebTemplatesDir       string
	WebDashboardTemplate  string
	WebStatusCacheTTLSec  float64
	WebChartJSSrc         string
	WebChartJSIntegrity   string
	WebChartJSCrossOrigin string

	// Runtime Config
	RuntimeConfigCacheTTLSec  float64
	RuntimeConfigWriteEnabled bool
	RuntimeConfigAuditMaxLen  int

	// 日志配置
	LogLevel string
}

var globalConfig *Config

// Load 加载配置
func Load() error {
	_ = godotenv.Load()

	globalConfig = &Config{
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getIntEnv("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),

		BinanceAPIKey:    getEnv("BINANCE_API_KEY", ""),
		BinanceSecretKey: getEnv("BINANCE_SECRET_KEY", ""),
		BinanceTestnet:   getBoolEnv("BINANCE_TESTNET", false),

		DryRun: getBoolEnv("DRY_RUN", true),

		AIProvider: strings.ToLower(getEnv("AI_PROVIDER", "deepseek")),

		DeepSeekEnabled:     getBoolEnv("DEEPSEEK_ENABLED", false),
		DeepSeekAPIKey:      getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekBaseURL:     getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:       getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		DeepSeekTemperature: getFloatEnv("DEEPSEEK_TEMPERATURE", 0.3),
		DeepSeekMaxTokens:   getIntEnv("DEEPSEEK_MAX_TOKENS", 4000),

		OpenAIEnabled:     getBoolEnv("OPENAI_ENABLED", false),
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:     getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:       getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAITemperature: getFloatEnv("OPENAI_TEMPERATURE", 0.3),
		OpenAIMaxTokens:   getIntEnv("OPENAI_MAX_TOKENS", 4000),

		GeminiEnabled:     getBoolEnv("GEMINI_ENABLED", false),
		GeminiAPIKey:      getEnv("GEMINI_API_KEY", ""),
		GeminiModel:       getEnv("GEMINI_MODEL", "gemini-pro"),
		GeminiTemperature: getFloatEnv("GEMINI_TEMPERATURE", 0.3),
		GeminiMaxTokens:   getIntEnv("GEMINI_MAX_TOKENS", 4000),

		AITemperature: getFloatEnv("AI_TEMPERATURE", 0.3),
		AIMaxTokens:   getIntEnv("AI_MAX_TOKENS", 4000),

		AITraderSystemPrompt: getEnv("AI_TRADER_SYSTEM_PROMPT",
			"你是一名经验丰富的加密货币合约交易员，请根据提供的市场数据（包括链上数据、衍生品与资金数据、市场情绪指标、技术分析指标、全球宏观经济环境）自行分析交易并做出交易决策。"),

		StrategyFile: getEnv("STRATEGY_FILE", "strategies/顺势狙击手.txt"),
		RuleStrategy: strings.ToLower(getEnv("RULE_STRATEGY", "shunshi_sniper")),

		DefaultTradingMode: strings.ToLower(getEnv("DEFAULT_TRADING_MODE", "")),

		ScanInterval:         getIntEnv("SCAN_INTERVAL", 180),
		PriceChangeThreshold: getFloatEnv("PRICE_CHANGE_THRESHOLD", 3.0),
		ScanConcurrency:      getIntEnv("SCAN_CONCURRENCY", 10),

		MarketSnapshotTTLSec:    getIntEnv("MARKET_SNAPSHOT_TTL_SEC", 600),
		MarketSnapshotMaxAgeSec: getIntEnv("MARKET_SNAPSHOT_MAX_AGE_SEC", 300),

		SignalTTLSec:      getIntEnv("SIGNAL_TTL_SEC", 3600),
		MaxTradeQueueSize: getIntEnv("MAX_TRADE_QUEUE_SIZE", 100),

		SymbolPoolTTLSec: getIntEnv("SYMBOL_POOL_TTL_SEC", 1800),
		OILastTTLSec:     getIntEnv("OI_LAST_TTL_SEC", 3600),

		MaxNotionalPerTrade:    getFloatEnv("MAX_NOTIONAL_PER_TRADE", 50.0),
		MaxLeverage:            getFloatEnv("MAX_LEVERAGE", 10.0),
		MaxConcurrentPositions: getIntEnv("MAX_CONCURRENT_POSITIONS", 5),
		SymbolCooldownSec:      getIntEnv("SYMBOL_COOLDOWN_SEC", 120),
		OrderDedupeWindow:      getIntEnv("ORDER_DEDUPE_WINDOW", 5),
		BreakoutTimeoutSec:     getIntEnv("BREAKOUT_TIMEOUT_SEC", 120),

		OrderAuditMaxLen:        getIntEnv("ORDER_AUDIT_MAX_LEN", 2000),
		OrderAuditEventMaxChars: getIntEnv("ORDER_AUDIT_EVENT_MAX_CHARS", 2000),

		SLTPGuardIntervalSec: getFloatEnv("SLTP_GUARD_INTERVAL_SEC", 10.0),
		GuardStatsTTLSec:     getIntEnv("GUARD_STATS_TTL_SEC", 86400*2),
		ProtectionTTLSec:     getIntEnv("PROTECTION_TTL_SEC", 86400),
		TP1PartialRatio:      getFloatEnv("TP1_PARTIAL_RATIO", 0.5),
		TPMatchTolerancePct:  getFloatEnv("TP_MATCH_TOLERANCE_PCT", 0.5),
		TakeProfitOrderType:  getEnv("TAKE_PROFIT_ORDER_TYPE", "limit"),
		MaxTPDeviationPct:    getFloatEnv("MAX_TP_DEVIATION_PCT", 25.0),

		ExchangeCacheTTLSec:          getFloatEnv("EXCHANGE_CACHE_TTL_SEC", 10.0),
		BinanceFAPIBaseURL:           getEnv("BINANCE_FAPI_BASE_URL", "https://fapi.binance.com"),
		BinanceHTTPTimeoutSec:        getFloatEnv("BINANCE_HTTP_TIMEOUT_SEC", 10.0),
		BinanceConnectorLimit:        getIntEnv("BINANCE_CONNECTOR_LIMIT", 100),
		BinanceConnectorLimitPerHost: getIntEnv("BINANCE_CONNECTOR_LIMIT_PER_HOST", 30),
		BinanceRateLimitMaxSleepSec:  getFloatEnv("BINANCE_RATE_LIMIT_MAX_SLEEP_SEC", 1.0),
		BinanceMinOnlineDays:         getIntEnv("BINANCE_MIN_ONLINE_DAYS", 30),

		RSIOverbought:      getFloatEnv("RSI_OVERBOUGHT", 78.0),
		RSIOversold:        getFloatEnv("RSI_OVERSOLD", 22.0),
		VolumeShrinkRatio:  getFloatEnv("VOLUME_SHRINK_RATIO", 0.85),
		BBSqueezeBandwidth: getFloatEnv("BB_SQUEEZE_BANDWIDTH", 0.01),

		IndEMAPeriod20:  getIntEnv("IND_EMA_PERIOD_20", 20),
		IndEMAPeriod50:  getIntEnv("IND_EMA_PERIOD_50", 50),
		IndEMAPeriod200: getIntEnv("IND_EMA_PERIOD_200", 200),
		IndRSIPeriod:    getIntEnv("IND_RSI_PERIOD", 14),
		IndBBPeriod:     getIntEnv("IND_BB_PERIOD", 20),
		IndBBStdDev:     getFloatEnv("IND_BB_STD_DEV", 2.0),
		IndCVDPeriod:    getIntEnv("IND_CVD_PERIOD", 50),

		StratConsecutiveMin:        getIntEnv("STRAT_CONSECUTIVE_MIN", 2),
		StratEMADivergenceMin:      getFloatEnv("STRAT_EMA_DIVERGENCE_MIN", 0.0008),
		StratEMA200WallPct:         getFloatEnv("STRAT_EMA200_WALL_PCT", 0.0015),
		StratEMA200HoldWarnPct:     getFloatEnv("STRAT_EMA200_HOLD_WARN_PCT", 0.002),
		StratZoneTolPct:            getFloatEnv("STRAT_ZONE_TOL_PCT", 0.003),
		StratBreakoutVolRatio:      getFloatEnv("STRAT_BREAKOUT_VOL_RATIO", 1.05),
		StratSqueezeRejectVolRatio: getFloatEnv("STRAT_SQUEEZE_REJECT_VOL_RATIO", 0.6),
		StratOIDropRejectPct:       getFloatEnv("STRAT_OI_DROP_REJECT_PCT", 8.0),
		StratMinProfitPct:          getFloatEnv("STRAT_MIN_PROFIT_PCT", 0.0020),
		StratMinRR:                 getFloatEnv("STRAT_MIN_RR", 1.1),
		StratSLEMA50BufferPct:      getFloatEnv("STRAT_SL_EMA50_BUFFER_PCT", 0.0005),
		StratBreakevenPct:          getFloatEnv("STRAT_BREAKEVEN_PCT", 0.0015),
		StratTP2RMult:              getFloatEnv("STRAT_TP2_R_MULT", 3.0),
		StratTP2FallbackPct:        getFloatEnv("STRAT_TP2_FALLBACK_PCT", 0.01),
		StratDefaultNotionalUSDT:   getFloatEnv("STRAT_DEFAULT_NOTIONAL_USDT", 20.0),

		WSTokenTTLSec: getIntEnv("WS_TOKEN_TTL_SEC", 60),

		AIAnalysisIntervalSec:          getIntEnv("AI_ANALYSIS_INTERVAL_SEC", 180),
		AIAnalysisConcurrency:          getIntEnv("AI_ANALYSIS_CONCURRENCY", 3),
		AIBatchSize:                    getIntEnv("AI_BATCH_SIZE", 2),
		AIForceFullPoolWhenNoAction:    getBoolEnv("AI_FORCE_FULL_POOL_WHEN_NO_ACTION", false),
		AIStatsTTLSec:                  getIntEnv("AI_STATS_TTL_SEC", 86400),
		AIPrefilterEnabled:             getBoolEnv("AI_PREFILTER_ENABLED", true),
		AIPrefilterMinAbsPct24h:        getFloatEnv("AI_PREFILTER_MIN_ABS_PCT_24H", 0.8),
		AIPrefilterMinAbsOIChange:      getFloatEnv("AI_PREFILTER_MIN_ABS_OI_CHANGE", 2.0),
		AIPrefilterMinVolumePeakRatio:  getFloatEnv("AI_PREFILTER_MIN_VOLUME_PEAK_RATIO", 1.05),
		AIPrefilterMinConsecutiveCount: getIntEnv("AI_PREFILTER_MIN_CONSECUTIVE_COUNT", 2),

		DeepSeekHistoryMaxLen:   getIntEnv("DEEPSEEK_HISTORY_MAX_LEN", 500),
		AIRequestHistoryMaxLen:  getIntEnv("AI_REQUEST_HISTORY_MAX_LEN", 500),
		AIResponseHistoryMaxLen: getIntEnv("AI_RESPONSE_HISTORY_MAX_LEN", 500),
		AIDecisionHistoryMaxLen: getIntEnv("AI_DECISION_HISTORY_MAX_LEN", 500),
		SignalHistoryMaxLen:     getIntEnv("SIGNAL_HISTORY_MAX_LEN", 500),
		TradeHistoryMaxLen:      getIntEnv("TRADE_HISTORY_MAX_LEN", 500),

		AlertEnabled:        getBoolEnv("ALERT_ENABLED", false),
		AlertWebhookURL:     getEnv("ALERT_WEBHOOK_URL", ""),
		AlertDedupeTTLSec:   getIntEnv("ALERT_DEDUPE_TTL_SEC", 300),
		AlertMinIntervalSec: getIntEnv("ALERT_MIN_INTERVAL_SEC", 10),

		MetricsEnabled:                   getBoolEnv("METRICS_ENABLED", true),
		MetricsSymbolSource:              strings.ToLower(getEnv("METRICS_SYMBOL_SOURCE", "scanner")),
		MetricsMaxSymbols:                getIntEnv("METRICS_MAX_SYMBOLS", 30),
		MetricsSymbols:                   getEnv("METRICS_SYMBOLS", ""),
		MetricsTimeframes:                getEnv("METRICS_TIMEFRAMES", "5m,15m,1h,4h,1d"),
		MetricsGlobalRefreshSec:          getIntEnv("METRICS_GLOBAL_REFRESH_SEC", 900),
		MetricsGlobalTTLSec:              getIntEnv("METRICS_GLOBAL_TTL_SEC", 86400),
		MetricsSymbolTTLSec:              getIntEnv("METRICS_SYMBOL_TTL_SEC", 86400*3),
		MetricsHTTPTimeoutSec:            getFloatEnv("METRICS_HTTP_TIMEOUT_SEC", 10.0),
		MetricsHTTPConnectorLimit:        getIntEnv("METRICS_HTTP_CONNECTOR_LIMIT", 50),
		MetricsHTTPConnectorLimitPerHost: getIntEnv("METRICS_HTTP_CONNECTOR_LIMIT_PER_HOST", 20),
		MetricsConcurrency:               getIntEnv("METRICS_CONCURRENCY", 3),
		MetricsOHLCVLimit:                getIntEnv("METRICS_OHLCV_LIMIT", 210),
		MetricsForceOrdersLimit:          getIntEnv("METRICS_FORCE_ORDERS_LIMIT", 50),

		CoinGeckoBaseURL: getEnv("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3"),

		GlassnodeAPIKey: getEnv("GLASSNODE_API_KEY", ""),

		WhaleAlertAPIKey:      getEnv("WHALE_ALERT_API_KEY", ""),
		WhaleAlertCurrencies:  getEnv("WHALE_ALERT_CURRENCIES", "btc,eth"),
		WhaleAlertMinValueUSD: getFloatEnv("WHALE_ALERT_MIN_VALUE_USD", 10000000.0),
		WhaleAlertLookbackSec: getIntEnv("WHALE_ALERT_LOOKBACK_SEC", 3600),
		WhaleAlertLimit:       getIntEnv("WHALE_ALERT_LIMIT", 20),

		OIEnabled:         getBoolEnv("OI_ENABLED", false),
		OIThreshold:       getFloatEnv("OI_THRESHOLD", 5.0),
		OIIntervalMinutes: getIntEnv("OI_INTERVAL_MINUTES", 5),
		OIExpireMinutes:   getIntEnv("OI_EXPIRE_MINUTES", 30),
		OIConcurrency:     getIntEnv("OI_CONCURRENCY", 20),
		OIUseWhitelist:    getBoolEnv("OI_USE_WHITELIST", false),
		OIWhitelist:       parseStringList(getEnv("OI_WHITELIST", "")),

		WebPort:               getIntEnv("WEB_PORT", 8000),
		WebBasicAuthUser:      getEnv("WEB_BASIC_AUTH_USER", ""),
		WebBasicAuthPass:      getEnv("WEB_BASIC_AUTH_PASS", ""),
		WebStaticDir:          getEnv("WEB_STATIC_DIR", "web/static"),
		WebTemplatesDir:       getEnv("WEB_TEMPLATES_DIR", "web/templates"),
		WebDashboardTemplate:  getEnv("WEB_DASHBOARD_TEMPLATE", "web/templates/dashboard.html"),
		WebStatusCacheTTLSec:  getFloatEnv("WEB_STATUS_CACHE_TTL_SEC", 15.0),
		WebChartJSSrc:         getEnv("WEB_CHARTJS_SRC", "https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"),
		WebChartJSIntegrity:   getEnv("WEB_CHARTJS_INTEGRITY", ""),
		WebChartJSCrossOrigin: getEnv("WEB_CHARTJS_CROSSORIGIN", "anonymous"),

		RuntimeConfigCacheTTLSec:  getFloatEnv("RUNTIME_CONFIG_CACHE_TTL_SEC", 3.0),
		RuntimeConfigWriteEnabled: getBoolEnv("RUNTIME_CONFIG_WRITE_ENABLED", true),
		RuntimeConfigAuditMaxLen:  getIntEnv("RUNTIME_CONFIG_AUDIT_MAX_LEN", 2000),

		LogLevel: getEnv("LOG_LEVEL", "INFO"),
	}

	return nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// GetRedisKey 生成Redis键名
func GetRedisKey(name string) string {
	return "nofx:" + name
}

// 辅助函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		value = strings.TrimSpace(value)
		if value == "" || value == "0" {
			return defaultValue
		}
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		value = strings.TrimSpace(value)
		if value == "" || value == "0" || value == "0.0" {
			return defaultValue
		}
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		value = strings.TrimSpace(value)
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func parseStringList(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(strings.ToUpper(part))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
