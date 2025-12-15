package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yuechangmingzou/nofx-go/internal/config"
	"github.com/yuechangmingzou/nofx-go/internal/utils"
)

// DeepSeekProvider DeepSeek AI提供商实现
type DeepSeekProvider struct {
	cfg    *config.Config
	client *http.Client
}

// NewDeepSeekProvider 创建DeepSeek提供商实例
func NewDeepSeekProvider(cfg *config.Config) *DeepSeekProvider {
	return &DeepSeekProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProvider 获取提供商类型
func (p *DeepSeekProvider) GetProvider() Provider {
	return ProviderDeepSeek
}

// GetModel 获取当前使用的模型
func (p *DeepSeekProvider) GetModel() string {
	if p.cfg.DeepSeekModel != "" {
		return p.cfg.DeepSeekModel
	}
	return "deepseek-chat"
}

// ChatCompletion 调用DeepSeek API
func (p *DeepSeekProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	logger := utils.GetLogger("ai_deepseek")

	// 构建请求
	apiURL := fmt.Sprintf("%s/v1/chat/completions", p.cfg.DeepSeekBaseURL)

	// 使用配置中的模型，如果没有则使用请求中的模型
	model := p.GetModel()
	if req.Model != "" {
		model = req.Model
	}

	requestBody := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}

	// 如果请求包含JSON格式要求，添加response_format
	jsonData, _ := json.Marshal(requestBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.DeepSeekAPIKey)

	// 记录开始时间
	startTime := time.Now()

	// 发送请求
	resp, err := p.client.Do(httpReq)
	if err != nil {
		latencyMs := int(time.Since(startTime).Milliseconds())
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("请求失败: %v", err),
		}, err
	}
	defer resp.Body.Close()

	latencyMs := int(time.Since(startTime).Milliseconds())

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("读取响应失败: %v", err),
		}, err
	}

	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		errorMsg := fmt.Sprintf("API错误: HTTP %d", resp.StatusCode)
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			errorMsg = fmt.Sprintf("API错误: %s (type: %s, code: %s)", errorResp.Error.Message, errorResp.Error.Type, errorResp.Error.Code)
		}
		
		logger.Warnw("DeepSeek API返回错误",
			"status", resp.StatusCode,
			"error", errorMsg,
			"body", string(body),
		)
		
		// 如果是速率限制，返回特殊错误
		if resp.StatusCode == http.StatusTooManyRequests {
			return &ChatResponse{
				Content:   "",
				LatencyMs: latencyMs,
				Error:     "速率限制: 请求过于频繁，请稍后重试",
			}, fmt.Errorf("速率限制: %s", errorMsg)
		}
		
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     errorMsg,
		}, fmt.Errorf("%s", errorMsg)
	}

	// 解析响应
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("解析响应失败: %v", err),
		}, err
	}

	if len(apiResp.Choices) == 0 {
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     "响应中没有choices",
		}, fmt.Errorf("响应中没有choices")
	}

	content := apiResp.Choices[0].Message.Content

	return &ChatResponse{
		Content:   content,
		LatencyMs: latencyMs,
		Error:     "",
	}, nil
}
