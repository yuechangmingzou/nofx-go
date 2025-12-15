package execution

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// acquireLock 获取分布式锁
func (e *ExecutionEngine) acquireLock(ctx context.Context, key string, ttl time.Duration) (string, error) {
	// 生成随机token
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	lockKey := fmt.Sprintf("lock:%s", key)

	// 使用SET NX EX实现分布式锁
	ok, err := e.redis.SetNX(ctx, lockKey, token, ttl).Result()
	if err != nil {
		return "", fmt.Errorf("获取锁失败: %w", err)
	}

	if !ok {
		return "", fmt.Errorf("锁已被占用")
	}

	return token, nil
}

// releaseLock 释放分布式锁
func (e *ExecutionEngine) releaseLock(ctx context.Context, key, token string) error {
	lockKey := fmt.Sprintf("lock:%s", key)

	// 使用Lua脚本确保只释放自己的锁
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	_, err := e.redis.Eval(ctx, script, []string{lockKey}, token).Result()
	return err
}
