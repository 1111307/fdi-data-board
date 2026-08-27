package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"

	"fdi_data_board/internal/biz"
)

// LlmRepo 大模型流式网关(Anthropic Messages 协议),实现 biz.LlmRepo。
// client 由 data.NewData 统一构建(见 data.llmClient),本 repo 只负责调用。
type LlmRepo struct {
	data *Data
}

var _ biz.LlmRepo = (*LlmRepo)(nil)

func NewLlmRepo(data *Data) biz.LlmRepo {
	return &LlmRepo{data: data}
}

// Enabled 网关是否已配置可用(client 未构建或 model 为空即视为未配置)
func (r *LlmRepo) Enabled() bool {
	if r == nil || r.data == nil || r.data.llmClient == nil {
		return false
	}
	return r.data.conf.GetLlm().GetModel() != ""
}

// Model 网关配置的模型名
func (r *LlmRepo) Model() string {
	if !r.Enabled() {
		return ""
	}
	return r.data.conf.GetLlm().GetModel()
}

// MaxTokens 单次输出上限(未配置/非法时默认 16384)
func (r *LlmRepo) MaxTokens() int64 {
	if !r.Enabled() {
		return 16384
	}
	if v := r.data.conf.GetLlm().GetMaxTokens(); v > 0 {
		return v
	}
	return 16384
}

// ChatStream 发起流式对话,每收到一段增量文本调用一次 onDelta
func (r *LlmRepo) ChatStream(ctx context.Context, systemPrompt, userPrompt string, onDelta func(string)) error {
	if !r.Enabled() {
		return fmt.Errorf("llm repo disabled")
	}

	stream := r.newStream(ctx, systemPrompt, userPrompt)
	defer stream.Close()
	return drainStream(stream, onDelta)
}

// ChatStreamEx 完整参数流式对话(带 tools/多轮历史),onEvent 逐事件回调,
// 返回聚合完成的最终 Message。Agent 循环使用。
func (r *LlmRepo) ChatStreamEx(ctx context.Context, params anthropic.MessageNewParams, onEvent func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("llm repo disabled")
	}
	if params.Model == "" {
		params.Model = r.Model()
	}
	if params.MaxTokens == 0 {
		params.MaxTokens = r.MaxTokens()
	}

	stream := r.data.llmClient.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	aggregator := newMessageAggregator()
	for stream.Next() {
		event := stream.Current()
		aggregator.feed(event)
		if onEvent != nil {
			onEvent(event)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("llm stream: %w", err)
	}
	return aggregator.message(), nil
}

func (r *LlmRepo) newStream(ctx context.Context, systemPrompt, userPrompt string) *ssestream.Stream[anthropic.MessageStreamEventUnion] {
	lc := r.data.conf.GetLlm()
	maxTokens := lc.GetMaxTokens()
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return r.data.llmClient.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     lc.GetModel(),
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
}

// ---------- 流式事件聚合:重建完整 Message(含 tool_use input JSON 拼装) ----------

type messageAggregator struct {
	msg       *anthropic.Message
	texts     map[int64]*strings.Builder
	toolUses  map[int64]*anthropic.ToolUseBlock
	inputJSON map[int64]*strings.Builder
	order     []int64
}

func newMessageAggregator() *messageAggregator {
	return &messageAggregator{
		msg:       &anthropic.Message{},
		texts:     map[int64]*strings.Builder{},
		toolUses:  map[int64]*anthropic.ToolUseBlock{},
		inputJSON: map[int64]*strings.Builder{},
	}
}

func (a *messageAggregator) feed(ev anthropic.MessageStreamEventUnion) {
	switch ev.Type {
	case "message_start":
		m := ev.AsMessageStart()
		*a.msg = m.Message
	case "content_block_start":
		start := ev.AsContentBlockStart()
		if tu := start.ContentBlock.AsToolUse(); tu.Name != "" {
			toolUse := tu
			a.toolUses[start.Index] = &toolUse
			a.inputJSON[start.Index] = &strings.Builder{}
			a.order = append(a.order, start.Index)
		} else if tb := start.ContentBlock.AsText(); tb.Text != "" || start.ContentBlock.Type == "text" {
			a.texts[start.Index] = &strings.Builder{}
			a.order = append(a.order, start.Index)
		}
	case "content_block_delta":
		delta := ev.AsContentBlockDelta()
		if d := delta.Delta.AsTextDelta(); d.Text != "" {
			if b, ok := a.texts[delta.Index]; ok {
				b.WriteString(d.Text)
			}
		} else if d := delta.Delta.AsInputJSONDelta(); d.PartialJSON != "" {
			if b, ok := a.inputJSON[delta.Index]; ok {
				b.WriteString(d.PartialJSON)
			}
		}
	case "message_delta":
		md := ev.AsMessageDelta()
		if md.Delta.StopReason != "" {
			a.msg.StopReason = md.Delta.StopReason
		}
	}
}

func (a *messageAggregator) message() *anthropic.Message {
	content := make([]anthropic.ContentBlockUnion, 0, len(a.order))
	for _, idx := range a.order {
		if b, ok := a.texts[idx]; ok {
			block := anthropic.TextBlock{Type: "text", Text: b.String()}
			raw, _ := json.Marshal(block)
			var u anthropic.ContentBlockUnion
			_ = u.UnmarshalJSON(raw)
			content = append(content, u)
		} else if tu, ok := a.toolUses[idx]; ok {
			tu.Input = json.RawMessage(a.inputJSON[idx].String())
			raw, _ := json.Marshal(tu)
			var u anthropic.ContentBlockUnion
			_ = u.UnmarshalJSON(raw)
			content = append(content, u)
		}
	}
	a.msg.Content = content
	return a.msg
}

func drainStream(stream *ssestream.Stream[anthropic.MessageStreamEventUnion], onDelta func(string)) error {
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
