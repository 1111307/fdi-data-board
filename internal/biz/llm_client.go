package biz

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// LlmConfig 大模型接入配置,来源环境变量(不进 conf.proto,避免改 proto 重新生成)
//   LLM_BASE_URL  Anthropic 协议网关地址,如 http://llm-gateway:8000
//   LLM_API_KEY   网关密钥(请求头 x-api-key)
//   LLM_MODEL     模型名,如 kimi-k3
//   LLM_MAX_TOKENS 单次生成上限,默认 4096
//   LLM_TIMEOUT   单次请求超时,默认 120s
// 全部未配置时客户端处于禁用状态,AI 总结接口退化为本地统计模式。
type LlmConfig struct {
	BaseURL   string
	APIKey    string
	Model     string
	MaxTokens int64
	Timeout   time.Duration
}

func LlmConfigFromEnv() *LlmConfig {
	timeout := 120 * time.Second
	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			timeout = d
		}
	}
	maxTokens := int64(4096)
	if raw := strings.TrimSpace(os.Getenv("LLM_MAX_TOKENS")); raw != "" {
		var v int64
		if _, err := fmt.Sscanf(raw, "%d", &v); err == nil && v > 0 {
			maxTokens = v
		}
	}
	return &LlmConfig{
		BaseURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/"),
		APIKey:    strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		Model:     strings.TrimSpace(os.Getenv("LLM_MODEL")),
		MaxTokens: maxTokens,
		Timeout:   timeout,
	}
}

func (c *LlmConfig) Enabled() bool {
	return c != nil && c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

// LlmClient 基于 anthropic-sdk-go 的流式客户端(供应商为 Anthropic Messages 协议)
type LlmClient struct {
	cfg    *LlmConfig
	client *anthropic.Client
}

func NewLlmClient(cfg *LlmConfig) *LlmClient {
	if !cfg.Enabled() {
		return &LlmClient{cfg: cfg}
	}
	client := anthropic.NewClient(
		option.WithBaseURL(cfg.BaseURL),
		option.WithAPIKey(cfg.APIKey),
		option.WithRequestTimeout(cfg.Timeout),
	)
	return &LlmClient{cfg: cfg, client: &client}
}

// Enabled 是否已配置可用的大模型网关
func (c *LlmClient) Enabled() bool { return c.cfg.Enabled() }

// ChatStream 发起流式对话,每收到一段增量文本调用一次 onDelta
func (c *LlmClient) ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string)) error {
	if !c.Enabled() {
		return fmt.Errorf("llm client disabled")
	}

	stream := c.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     c.cfg.Model,
		MaxTokens: c.cfg.MaxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	defer stream.Close()

	for stream.Next() {
		event := stream.Current()
		if event.Type != "content_block_delta" {
			continue
		}
		delta := event.AsContentBlockDelta()
		if text := delta.Delta.AsTextDelta().Text; text != "" {
			onDelta(text)
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("llm stream: %w", err)
	}
	return nil
}
