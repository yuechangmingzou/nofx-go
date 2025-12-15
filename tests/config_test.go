package tests

import (
	"os"
	"testing"

	"github.com/yourusername/nofx-go/internal/config"
)

func TestConfigLoad(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("REDIS_HOST", "test-host")
	os.Setenv("REDIS_PORT", "6380")
	os.Setenv("DRY_RUN", "false")

	// 加载配置
	err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	cfg := config.Get()
	if cfg == nil {
		t.Fatal("Config is nil")
	}

	// 验证配置值
	if cfg.RedisHost != "test-host" {
		t.Errorf("Expected RedisHost to be 'test-host', got '%s'", cfg.RedisHost)
	}

	if cfg.RedisPort != 6380 {
		t.Errorf("Expected RedisPort to be 6380, got %d", cfg.RedisPort)
	}

	if cfg.DryRun != false {
		t.Errorf("Expected DryRun to be false, got %v", cfg.DryRun)
	}
}

func TestGetRedisKey(t *testing.T) {
	key := config.GetRedisKey("test:key")
	expected := "nofx:test:key"
	if key != expected {
		t.Errorf("Expected '%s', got '%s'", expected, key)
	}
}

