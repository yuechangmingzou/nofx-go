package config

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// StringCmd Redis字符串命令结果
type StringCmd struct {
	val string
	err error
}

// Result 获取结果
func (c *StringCmd) Result() (string, error) {
	return c.val, c.err
}

// StatusCmd Redis状态命令结果
type StatusCmd struct {
	err error
}

// Err 获取错误
func (c *StatusCmd) Err() error {
	return c.err
}

// RedisAdapter Redis适配器（避免循环导入）
type RedisAdapter struct {
	client *redis.Client
}

// NewRedisAdapter 创建Redis适配器（延迟初始化）
func NewRedisAdapter() *RedisAdapter {
	return &RedisAdapter{}
}

// SetClient 设置Redis客户端（延迟初始化）
func (r *RedisAdapter) SetClient(client *redis.Client) {
	r.client = client
}

// Get 获取值
func (r *RedisAdapter) Get(ctx context.Context, key string) *StringCmd {
	if r.client == nil {
		return &StringCmd{err: fmt.Errorf("redis client not initialized")}
	}
	val, err := r.client.Get(ctx, key).Result()
	return &StringCmd{val: val, err: err}
}

// Set 设置值
func (r *RedisAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *StatusCmd {
	if r.client == nil {
		return &StatusCmd{err: fmt.Errorf("redis client not initialized")}
	}
	err := r.client.Set(ctx, key, value, expiration).Err()
	return &StatusCmd{err: err}
}

