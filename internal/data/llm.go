package data

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/conf"
)

// LlmRepo 大模型流式网关客户端(Anthropic Messages 协议),实现 biz.LlmRepo
type LlmRepo struct {
	client    *anthropic.Client
	model     string
	maxTokens int64
	enabled   bool
}

var _ biz.LlmRepo = (*LlmRepo)(nil)

// NewLlmRepo 按 conf.Data.llm 构建;base_url/api_key/model 任一未配置时禁用,
// AI 总结接口自动退化为本地统计模式
func NewLlmRepo(c *conf.Data, logger log.Logger) *LlmRepo {
	lc := c.GetLlm()
	if lc == nil || lc.GetBaseUrl() == "" || lc.GetApiKey() == "" || lc.GetModel() == "" {
		log.NewHelper(logger).Warn("[llm] base_url/api_key/model 未配置,AI 总结将使用本地统计模式")
		return &LlmRepo{enabled: false}
	}

	maxTokens := lc.GetMaxTokens()
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	timeout := 120 * time.Second
	if lc.GetTimeout() != nil && lc.GetTimeout().AsDuration() > 0 {
		timeout = lc.GetTimeout().AsDuration()
	}

	client := anthropic.NewClient(
		option.WithBaseURL(lc.GetBaseUrl()),
		option.WithAPIKey(lc.GetApiKey()),
		option.WithRequestTimeout(timeout),
	)
	return &LlmRepo{client: &client, model: lc.GetModel(), maxTokens: maxTokens, enabled: true}
}

// Enabled 网关是否已配置可用
func (r *LlmRepo) Enabled() bool { return r != nil && r.enabled }

// ChatStream 发起流式对话,每收到一段增量文本调用一次 onDelta
func (r *LlmRepo) ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string)) error {
	if !r.Enabled() {
		return fmt.Errorf("llm repo disabled")
	}

	stream := r.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     r.model,
		MaxTokens: r.maxTokens,
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
		if text := event.AsContentBlockDelta().Delta.AsTextDelta().Text; text != "" {
			onDelta(text)
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("llm stream: %w", err)
	}
	return nil
}
