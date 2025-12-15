package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yuechangmingzou/nofx-go/internal/config"
	"go.uber.org/zap"
)

var redisClient *redis.Client

// RedisClient Redis客户端类型别名（供其他包使用）
type RedisClient = *redis.Client

// GetRedisClient 获取Redis客户端（单例模式）
func GetRedisClient() *redis.Client {
	if redisClient == nil {
		cfg := config.Get()
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		// 测试连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			// Redis连接失败，记录错误
			// 注意：Redis是核心依赖，连接失败应该被处理
			// 这里不panic，让调用者决定如何处理
			logger, _ := zap.NewDevelopment()
			logger.Error("Redis连接失败",
				zap.Error(err),
				zap.String("host", cfg.RedisHost),
				zap.Int("port", cfg.RedisPort),
			)
			// 返回client实例，但后续操作可能会失败
			// 如果Redis是必需的，应该在main.go中检查并退出
		}
	}
	return redisClient
}

// CloseRedisClient 关闭Redis客户端
func CloseRedisClient() error {
	if redisClient != nil {
		return redisClient.Close()
	}
	return nil
}

