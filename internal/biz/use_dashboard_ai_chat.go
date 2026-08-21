package biz

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

//go:embed knowledge/ai_dict.md
var aiKnowledgeFS embed.FS

// aiMaxRounds 单次请求内 agent 循环轮数上限(防成本失控)
const aiMaxRounds = 6

// aiMaxHistory 历史消息条数上限(防 prompt 膨胀)
const aiMaxHistory = 40

// ChatEvent SSE 下行事件(前端据此渲染:delta 文本/tool_call 工具卡/clarify 确认卡/tool_result 结果与图表数据)
type ChatEvent struct {
	Type    string          `json:"type"` // delta / tool_call / tool_result / clarify / done / error
	Text    string          `json:"text,omitempty"`
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Args    map[string]any  `json:"args,omitempty"`
	Summary string          `json:"summary,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"` // tool_result 的完整 JSON(前端渲染图表)
	Stop    string          `json:"stop,omitempty"` // done 帧的结束原因: end_turn / clarify / max_rounds
	Error   string          `json:"error,omitempty"`
}

// AiChatHistoryMessage 前端回传的会话历史(轻量结构,后端重建为 Anthropic content blocks)
type AiChatHistoryMessage struct {
	Role        string             `json:"role"`                   // user / assistant
	Text        string             `json:"text,omitempty"`         //ai或者用户说过的话
	ToolCalls   []AiChatToolCall   `json:"tool_calls,omitempty"`   //ai调用的工具
	ToolResults []AiChatToolResult `json:"tool_results,omitempty"` //工具调用结果
}

type AiChatToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type AiChatToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// knowledgePrompt 加载知识包并注入当天日期
func knowledgePrompt() string {
	raw, err := aiKnowledgeFS.ReadFile("knowledge/ai_dict.md")
	if err != nil {
		return "今天是 " + time.Now().Format("2006-01-02") + "。"
	}
	return strings.ReplaceAll(string(raw), "{{TODAY}}", time.Now().Format("2006-01-02"))
}

// StreamChat AI 问答 agent 循环:
// 文本增量 → emit(delta);确定调用工具 → 进程内执行 → tool_result 回填继续循环;
// clarify 工具 → emit(clarify) 后正常结束本流(暂停点),前端确认后带历史重新请求续跑。
func (uc *AiDashboardUseCase) StreamChat(ctx context.Context, question string, history []AiChatHistoryMessage, emit func(ChatEvent) error) error {
	msgs, err := buildAnthropicMessages(question, history)
	if err != nil {
		return err
	}

	for round := 0; round < aiMaxRounds; round++ {
		params := anthropic.MessageNewParams{
			Model:     uc.llmModel(),
			MaxTokens: 4096,
			System:    []anthropic.TextBlockParam{{Text: knowledgePrompt()}},
			Messages:  msgs,
			Tools:     uc.tools.Params(),
		}

		var textBuf strings.Builder
		var toolUses []anthropic.ToolUseBlock
		// 流式回调:网关的每个 SSE 事件都会进来一次,但只有一种事件会往外 emit——
		// ① Type 为 content_block_delta(增量事件)才继续,块的 start/stop、
		//    message_start/delta/stop 等生命周期事件直接忽略;
		// ② 该增量必须是 text_delta(正文碎片)才 emit(delta) 推给前端。
		//    同为 content_block_delta 的另外两种在此静默丢弃:
		//    - input_json_delta(tool_use 参数碎片)→ 由 ChatStreamEx 内的 aggregator 拼装,
		//      流结束后以完整 tool_use 出现在返回值 msg 里,由下方循环翻译成
		//      clarify / tool_call / tool_result 帧(不走 delta 通道);
		//    - thinking_delta(思考碎片)→ 整体过滤,前端不可见。
		msg, err := uc.llm.ChatStreamEx(ctx, params, func(ev anthropic.MessageStreamEventUnion) {
			if ev.Type != "content_block_delta" {
				return
			}
			if text := ev.AsContentBlockDelta().Delta.AsTextDelta().Text; text != "" {
				textBuf.WriteString(text)
				_ = emit(ChatEvent{Type: "delta", Text: text})
			}
		})
		if err != nil {
			return fmt.Errorf("llm stream: %w", err)
		}
		if msg == nil {
			return fmt.Errorf("llm stream: empty message")
		}
		for _, block := range msg.Content {
			if tu := block.AsToolUse(); tu.Name != "" {
				toolUses = append(toolUses, tu)
			}
		}

		// 无工具调用或 end_turn:结束
		if len(toolUses) == 0 || msg.StopReason != anthropic.StopReasonToolUse {
			return emit(ChatEvent{Type: "done", Stop: "end_turn"})
		}

		// 重建 assistant 消息(文本 + tool_use blocks)追加进上下文
		assistantBlocks := []anthropic.ContentBlockParamUnion{}
		if textBuf.Len() > 0 {
			assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(textBuf.String()))
		}
		resultBlocks := []anthropic.ContentBlockParamUnion{}
		clarifyPaused := false

		for _, tu := range toolUses {
			assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(tu.ID, tu.Input, tu.Name))

			// clarify:人工确认暂停点,本流到此结束
			if tu.Name == aiClarifyToolName {
				var payload map[string]any
				if len(tu.Input) > 0 {
					_ = json.Unmarshal(tu.Input, &payload)
				}
				_ = emit(ChatEvent{Type: "clarify", ID: tu.ID, Name: tu.Name, Args: payload})
				clarifyPaused = true
				continue
			}

			// 普通工具:进程内执行(口径与看板一致),结果作为 tool_result 回填
			_ = emit(ChatEvent{Type: "tool_call", ID: tu.ID, Name: tu.Name, Args: rawToArgs(tu.Input)})
			result, execErr := uc.tools.Exec(ctx, tu.Name, rawToArgs(tu.Input))
			if execErr != nil {
				result = "工具执行失败: " + execErr.Error()
			}
			summary := result
			if len(summary) > 120 {
				summary = summary[:120] + "…"
			}
			_ = emit(ChatEvent{Type: "tool_result", ID: tu.ID, Name: tu.Name, Summary: summary, Data: json.RawMessage(result)})
			resultBlocks = append(resultBlocks, anthropic.NewToolResultBlock(tu.ID, result, execErr != nil))
		}

		msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleAssistant, Content: assistantBlocks})

		if clarifyPaused {
			// 暂停点:前端确认后,把 assistant(tool_use) + user(tool_result) 带回历史重新请求
			return emit(ChatEvent{Type: "done", Stop: "clarify"})
		}
		if len(resultBlocks) > 0 {
			msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: resultBlocks})
		}
	}

	return emit(ChatEvent{Type: "done", Stop: "max_rounds"})
}

// buildAnthropicMessages 校验并重建历史 + 当前提问
func buildAnthropicMessages(question string, history []AiChatHistoryMessage) ([]anthropic.MessageParam, error) {
	q := strings.TrimSpace(question)
	// 纯续跑场景(clarify 确认后):question 可为空,但历史必须以 tool_result 结尾,
	// 此时不再追加任何用户文本,由历史尾部的 tool_result 驱动模型继续
	if q == "" && !historyEndsWithToolResult(history) {
		return nil, fmt.Errorf("question 不能为空")
	}
	if len(history) > aiMaxHistory {
		history = history[len(history)-aiMaxHistory:]
	}

	msgs := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, h := range history {
		switch h.Role {
		case "user":
			blocks := []anthropic.ContentBlockParamUnion{}
			for _, tr := range h.ToolResults {
				blocks = append(blocks, anthropic.NewToolResultBlock(tr.ID, tr.Content, tr.IsError))
			}
			if len(blocks) == 0 {
				if strings.TrimSpace(h.Text) == "" {
					continue
				}
				blocks = append(blocks, anthropic.NewTextBlock(h.Text))
			}
			msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: blocks})
		case "assistant":
			blocks := []anthropic.ContentBlockParamUnion{}
			if strings.TrimSpace(h.Text) != "" {
				blocks = append(blocks, anthropic.NewTextBlock(h.Text))
			}
			for _, tc := range h.ToolCalls {
				raw, _ := json.Marshal(tc.Args)
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, raw, tc.Name))
			}
			if len(blocks) == 0 {
				continue
			}
			msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleAssistant, Content: blocks})
		default:
			return nil, fmt.Errorf("历史消息 role 非法: %s", h.Role)
		}
	}
	if q != "" {
		msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(q)}})
	}
	return backfillToolResults(msgs), nil
}

// historyEndsWithToolResult 历史末尾是否为带 tool_result 的 user 消息(即挂起的确认回执)
func historyEndsWithToolResult(history []AiChatHistoryMessage) bool {
	if len(history) == 0 || history[len(history)-1].Role != "user" {
		return false
	}
	return len(history[len(history)-1].ToolResults) > 0
}

// backfillToolResults 协议兜底:assistant 里的每个 tool_use 必须在紧邻的下一条 user 消息里
// 有对应 tool_result,否则网关 400。历史回传缺失时(前端未带、clarify 未回答、暂停丢结果)
// 自动补占位 tool_result,保证请求永远合法。
func backfillToolResults(msgs []anthropic.MessageParam) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs)+4)
	for i := 0; i < len(msgs); i++ {
		m := msgs[i]
		if m.Role != anthropic.MessageParamRoleAssistant {
			out = append(out, m)
			continue
		}
		missing := missingToolResultIDs(m, msgs, i)
		if len(missing) == 0 {
			out = append(out, m)
			continue
		}
		// 缺失的 tool_result 处理:
		// 下一条是 user 且带 tool_result → 把缺失块并入该消息开头;
		// 否则在 assistant 后插入一条补齐的 user 消息。
		next := []anthropic.ContentBlockParamUnion{}
		for _, id := range missing {
			next = append(next, anthropic.NewToolResultBlock(id, "(该工具调用在上一轮已完成,结果未随历史回传)", false))
		}
		if i+1 < len(msgs) && msgs[i+1].Role == anthropic.MessageParamRoleUser && hasToolResult(msgs[i+1]) {
			merged := append(next, msgs[i+1].Content...)
			out = append(out, m, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: merged})
			i++ // 原下一条已并入
		} else if i+1 < len(msgs) && msgs[i+1].Role == anthropic.MessageParamRoleUser {
			// 下一条是纯文本 user → tool_result 块拼在其前(协议允许同一 user 消息混合)
			merged := append(next, msgs[i+1].Content...)
			out = append(out, m, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: merged})
			i++
		} else {
			out = append(out, m, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: next})
		}
	}
	return out
}

func missingToolResultIDs(assistant anthropic.MessageParam, msgs []anthropic.MessageParam, i int) []string {
	var useIDs []string
	for _, b := range assistant.Content {
		if tu := b.OfToolUse; tu != nil {
			useIDs = append(useIDs, tu.ID)
		}
	}
	if len(useIDs) == 0 {
		return nil
	}
	covered := map[string]bool{}
	if i+1 < len(msgs) && msgs[i+1].Role == anthropic.MessageParamRoleUser {
		for _, b := range msgs[i+1].Content {
			if tr := b.OfToolResult; tr != nil {
				covered[tr.ToolUseID] = true
			}
		}
	}
	missing := []string{}
	for _, id := range useIDs {
		if !covered[id] {
			missing = append(missing, id)
		}
	}
	return missing
}

func hasToolResult(m anthropic.MessageParam) bool {
	for _, b := range m.Content {
		if b.OfToolResult != nil {
			return true
		}
	}
	return false
}

func rawToArgs(raw json.RawMessage) map[string]any {
	args := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &args)
	}
	return args
}
