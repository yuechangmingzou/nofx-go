package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/utils"
)

// GeminiProvider Google Gemini提供商实现
type GeminiProvider struct {
	cfg    *config.Config
	client *http.Client
}

// NewGeminiProvider 创建Gemini提供商实例
func NewGeminiProvider(cfg *config.Config) *GeminiProvider {
	return &GeminiProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProvider 获取提供商类型
func (p *GeminiProvider) GetProvider() Provider {
	return ProviderGemini
}

// GetModel 获取当前使用的模型
func (p *GeminiProvider) GetModel() string {
	if p.cfg.GeminiModel != "" {
		return p.cfg.GeminiModel
	}
	return "gemini-pro"
}

// ChatCompletion 调用Gemini API
func (p *GeminiProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	logger := utils.GetLogger("ai_gemini")

	// Gemini API格式略有不同
	model := p.GetModel()
	if req.Model != "" {
		model = req.Model
	}

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, p.cfg.GeminiAPIKey)

	// 转换消息格式
	var contents []map[string]interface{}
	for _, msg := range req.Messages {
		contents = append(contents, map[string]interface{}{
			"role": msg.Role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		})
	}

	requestBody := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature": req.Temperature,
			"maxOutputTokens": req.MaxTokens,
		},
	}

	jsonData, _ := json.Marshal(requestBody)
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	startTime := time.Now()

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("读取响应失败: %v", err),
		}, err
	}

	if resp.StatusCode != http.StatusOK {
		logger.Warnw("Gemini API返回错误",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("API错误: HTTP %d", resp.StatusCode),
		}, fmt.Errorf("API错误: HTTP %d", resp.StatusCode)
	}

	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
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

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return &ChatResponse{
			Content:   "",
			LatencyMs: latencyMs,
			Error:     "响应中没有内容",
		}, fmt.Errorf("响应中没有内容")
	}

	content := apiResp.Candidates[0].Content.Parts[0].Text

	return &ChatResponse{
		Content:   content,
		LatencyMs: latencyMs,
		Error:     "",
	}, nil
}

