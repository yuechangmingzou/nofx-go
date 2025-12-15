package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/nofx-go/internal/config"
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
			panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
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

