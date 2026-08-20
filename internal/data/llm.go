package data

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"

	"fdi_data_board/internal/biz"
)

// LlmRepo 大模型流式网关(Anthropic Messages 协议),实现 biz.LlmRepo。
// client 由 data.NewData 统一构建(见 data.llmClient),本 repo 只负责调用。
type LlmRepo struct {
	data *Data
}

var _ biz.LlmRepo = (*LlmRepo)(nil)

func NewLlmRepo(data *Data) *LlmRepo {
	return &LlmRepo{data: data}
}

// Enabled 网关是否已配置可用(client 未构建或 model 为空即视为未配置)
func (r *LlmRepo) Enabled() bool {
	if r == nil || r.data == nil || r.data.llmClient == nil {
		return false
	}
	return r.data.conf.GetLlm().GetModel() != ""
}

// ChatStream 发起流式对话,每收到一段增量文本调用一次 onDelta
func (r *LlmRepo) ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string)) error {
	if !r.Enabled() {
		return fmt.Errorf("llm repo disabled")
	}

	lc := r.data.conf.GetLlm()
	maxTokens := lc.GetMaxTokens()
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	stream := r.data.llmClient.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     lc.GetModel(),
		MaxTokens: maxTokens,
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
