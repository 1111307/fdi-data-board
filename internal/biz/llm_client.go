package biz

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// LlmConfig 大模型接入配置,来源环境变量(不进 conf.proto,避免改 proto 重新生成)
//   LLM_BASE_URL  OpenAI 兼容网关地址,如 http://llm-gateway:8000/v1
//   LLM_API_KEY   网关密钥
//   LLM_MODEL     模型名
//   LLM_TIMEOUT   单次请求超时,默认 120s
// 全部未配置时客户端处于禁用状态,AI 总结接口退化为本地统计模式。
type LlmConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

func LlmConfigFromEnv() *LlmConfig {
	timeout := 120 * time.Second
	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			timeout = d
		}
	}
	return &LlmConfig{
		BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/"),
		APIKey:  strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		Model:   strings.TrimSpace(os.Getenv("LLM_MODEL")),
		Timeout: timeout,
	}
}

func (c *LlmConfig) Enabled() bool {
	return c != nil && c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

// LlmClient 最小 OpenAI 兼容 Chat Completions 流式客户端
type LlmClient struct {
	cfg    *LlmConfig
	client *http.Client
}

func NewLlmClient(cfg *LlmConfig) *LlmClient {
	return &LlmClient{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Enabled 是否已配置可用的大模型网关
func (c *LlmClient) Enabled() bool { return c.cfg.Enabled() }

// ChatStream 发起流式对话,每收到一段增量文本调用一次 onDelta
func (c *LlmClient) ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string)) error {
	if !c.Enabled() {
		return fmt.Errorf("llm client disabled")
	}

	reqBody := map[string]interface{}{
		"model": c.cfg.Model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("llm gateway status %d: %s", resp.StatusCode, string(body))
	}

	return scanLlmSse(resp.Body, onDelta)
}

// scanLlmSse 解析 SSE 流,提取 choices[0].delta.content
func scanLlmSse(r io.Reader, onDelta func(string)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue // 跳过无法解析的心跳/备注帧
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onDelta(chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}
