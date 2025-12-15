package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/internal/utils"
)

// Provider AI提供商类型
type Provider string

const (
	ProviderDeepSeek Provider = "deepseek"
	ProviderOpenAI   Provider = "openai"
	ProviderGemini   Provider = "gemini"
)

// AIProvider AI提供商接口
type AIProvider interface {
	// ChatCompletion 调用AI API进行对话
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	
	// GetProvider 获取提供商类型
	GetProvider() Provider
	
	// GetModel 获取当前使用的模型
	GetModel() string
}

// ChatRequest AI对话请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message 消息
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatResponse AI对话响应
type ChatResponse struct {
	Content   string `json:"content"`
	LatencyMs int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// GetAIProvider 获取AI提供商实例
func GetAIProvider() (AIProvider, error) {
	cfg := config.Get()
	logger := utils.GetLogger("ai_provider")

	providerName := strings.ToLower(cfg.AIProvider)
	
	switch providerName {
	case "deepseek":
		if !cfg.DeepSeekEnabled || cfg.DeepSeekAPIKey == "" {
			return nil, fmt.Errorf("DeepSeek未启用或API Key未配置")
		}
		return NewDeepSeekProvider(cfg), nil
	case "openai":
		if !cfg.OpenAIEnabled || cfg.OpenAIAPIKey == "" {
			return nil, fmt.Errorf("OpenAI未启用或API Key未配置")
		}
		return NewOpenAIProvider(cfg), nil
	case "gemini":
		if !cfg.GeminiEnabled || cfg.GeminiAPIKey == "" {
			return nil, fmt.Errorf("Gemini未启用或API Key未配置")
		}
		return NewGeminiProvider(cfg), nil
	default:
		logger.Warnw("未知的AI提供商，尝试使用DeepSeek",
			"provider", providerName,
		)
		if cfg.DeepSeekEnabled && cfg.DeepSeekAPIKey != "" {
			return NewDeepSeekProvider(cfg), nil
		}
		return nil, fmt.Errorf("未配置有效的AI提供商")
	}
}

// GetAIModel 获取当前使用的模型名称
func GetAIModel(provider AIProvider) string {
	if provider == nil {
		return "unknown"
	}
	return provider.GetModel()
}

