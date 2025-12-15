package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

// InitLogger 初始化日志器
func InitLogger(level string) error {
	var zapLevel zapcore.Level
	switch level {
	case "DEBUG":
		zapLevel = zap.DebugLevel
	case "INFO":
		zapLevel = zap.InfoLevel
	case "WARNING", "WARN":
		zapLevel = zap.WarnLevel
	case "ERROR":
		zapLevel = zap.ErrorLevel
	case "CRITICAL", "FATAL":
		zapLevel = zap.FatalLevel
	default:
		zapLevel = zap.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// GetLogger 获取日志器
func GetLogger(name string) *zap.SugaredLogger {
	if logger == nil {
		// 使用默认配置
		_ = InitLogger("INFO")
	}
	return logger.Named(name).Sugar()
}

