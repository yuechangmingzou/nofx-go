package web

import (
	"sync"
	"time"
)

// statusCache 状态缓存
type statusCache struct {
	mu        sync.RWMutex
	data      map[string]interface{}
	timestamp time.Time
	ttl       time.Duration
}

var globalStatusCache = &statusCache{
	ttl: 15 * time.Second,
}

// get 获取缓存数据
func (c *statusCache) get() (map[string]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.timestamp) < c.ttl {
		return c.data, true
	}

	return nil, false
}

// set 设置缓存数据
func (c *statusCache) set(data map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.timestamp = time.Now()
}

// clear 清除缓存
func (c *statusCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = nil
	c.timestamp = time.Time{}
}

