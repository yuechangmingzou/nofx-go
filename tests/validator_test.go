package tests

import (
	"os"
	"testing"

	"github.com/yuechangmingzou/nofx-go/internal/config"
)

func TestValidateConfig_Success(t *testing.T) {
	// 设置有效的配置
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("WEB_BASIC_AUTH_USER", "admin")
	os.Setenv("WEB_BASIC_AUTH_PASS", "strongpassword123")
	os.Setenv("DRY_RUN", "true")
	os.Setenv("STRATEGY_FILE", "strategies/test.txt")

	config.Load()
	err := config.ValidateConfig()
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

func TestValidateConfig_MissingRedisHost(t *testing.T) {
	os.Setenv("REDIS_HOST", "")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("WEB_BASIC_AUTH_USER", "admin")
	os.Setenv("WEB_BASIC_AUTH_PASS", "strongpassword123")
	os.Setenv("DRY_RUN", "true")

	config.Load()
	err := config.ValidateConfig()
	if err == nil {
		t.Error("Expected validation to fail for missing REDIS_HOST")
	}
}

func TestValidateConfig_WeakPassword(t *testing.T) {
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("WEB_BASIC_AUTH_USER", "admin")
	os.Setenv("WEB_BASIC_AUTH_PASS", "weak") // 太短
	os.Setenv("DRY_RUN", "true")

	config.Load()
	err := config.ValidateConfig()
	if err == nil {
		t.Error("Expected validation to fail for weak password")
	}
}

func TestValidateConfig_DefaultPassword(t *testing.T) {
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("WEB_BASIC_AUTH_USER", "admin")
	os.Setenv("WEB_BASIC_AUTH_PASS", "change_me") // 默认密码
	os.Setenv("DRY_RUN", "true")

	config.Load()
	err := config.ValidateConfig()
	if err == nil {
		t.Error("Expected validation to fail for default password")
	}
}

func TestValidateConfig_MissingBinanceKey(t *testing.T) {
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("WEB_BASIC_AUTH_USER", "admin")
	os.Setenv("WEB_BASIC_AUTH_PASS", "strongpassword123")
	os.Setenv("DRY_RUN", "false") // 非DRY_RUN模式
	os.Setenv("BINANCE_API_KEY", "") // 缺少API Key

	config.Load()
	err := config.ValidateConfig()
	if err == nil {
		t.Error("Expected validation to fail for missing BINANCE_API_KEY when DRY_RUN=false")
	}
}

