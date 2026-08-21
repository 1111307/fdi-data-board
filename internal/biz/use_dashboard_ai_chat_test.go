package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// chatLlmFake 按调用序返回预设 Message 的假网关,并捕获每轮收到的请求参数
type chatLlmFake struct {
	rounds  []string // 每轮返回的原始 JSON(message content 场景)
	calls   int
	lastReq anthropic.MessageNewParams
}

func mustBlock(raw string) anthropic.ContentBlockUnion {
	var u anthropic.ContentBlockUnion
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		panic(err)
	}
	return u
}

func mustMessage(raw string) *anthropic.Message {
	var m anthropic.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		panic(err)
	}
	return &m
}

func (f *chatLlmFake) Enabled() bool { return true }
func (f *chatLlmFake) Model() string { return "kimi-k3-test" }
func (f *chatLlmFake) ChatStream(ctx context.Context, s1, s2 string, cb func(string)) error {
	return fmt.Errorf("unused")
}

func (f *chatLlmFake) ChatStreamEx(_ context.Context, params anthropic.MessageNewParams, onEvent func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error) {
	f.lastReq = params
	if f.calls >= len(f.rounds) {
		return nil, fmt.Errorf("script exhausted")
	}
	raw := f.rounds[f.calls]
	f.calls++
	// 触发一次文本 delta 事件,验证 onEvent 转发链路
	if onEvent != nil {
		var ev anthropic.MessageStreamEventUnion
		_ = json.Unmarshal([]byte(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"."}}`), &ev)
		onEvent(ev)
	}
	return mustMessage(raw), nil
}

func collectEvents(t *testing.T, uc *AiDashboardUseCase, question string, history []AiChatHistoryMessage) []ChatEvent {
	t.Helper()
	var events []ChatEvent
	err := uc.StreamChat(context.Background(), question, history, func(ev ChatEvent) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat error: %v", err)
	}
	return events
}

func eventTypes(events []ChatEvent) string {
	parts := make([]string, 0, len(events))
	for _, e := range events {
		parts = append(parts, e.Type)
	}
	return strings.Join(parts, ",")
}

// 用例1:模型调用数据工具 → 后端进程内执行(fake 仓储)→ tool_result 回填 → 二轮 end_turn
func TestStreamChatToolLoopExecutesAndFeedsResult(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m1","type":"message","role":"assistant","model":"k","stop_reason":"tool_use","content":[
			{"type":"text","text":"我先查一下失败原因。"},
			{"type":"tool_use","id":"t1","name":"get_fail_reason","input":{"stage":"fff"}}]}`,
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"近7天失败 468 次,cooldown 占比最高。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	events := collectEvents(t, uc, "失败原因有哪些?", nil)

	// 事件序列:delta(一轮文本) tool_call tool_result delta(二轮) delta(伪事件".") done
	got := eventTypes(events)
	if !strings.Contains(got, "tool_call") || !strings.Contains(got, "tool_result") {
		t.Fatalf("missing tool events, got: %s", got)
	}
	last := events[len(events)-1]
	if last.Type != "done" || last.Stop != "end_turn" {
		t.Fatalf("last event = %+v, want done/end_turn", last)
	}

	// 二轮请求应包含 assistant(tool_use) 与 user(tool_result) 两条消息
	fake.calls = 0 // 防再次误用
	msgs := fake.lastReq.Messages
	if len(msgs) < 3 {
		t.Fatalf("second round messages = %d, want >= 3 (user/assistant+tooluse/user+toolresult)", len(msgs))
	}
	// param union 无 As* 访问器,直接看序列化产物
	msgsJSON, _ := json.Marshal(msgs)
	if !strings.Contains(string(msgsJSON), `"tool_use_id":"t1"`) {
		t.Fatalf("no tool_result block fed back: %s", string(msgsJSON))
	}
	if !strings.Contains(string(msgsJSON), "FFF-cooldown") {
		t.Fatalf("tool_result content missing fake data: %s", string(msgsJSON))
	}
	// 工具定义应随请求下发
	if len(fake.lastReq.Tools) == 0 {
		t.Fatalf("tools not sent to gateway")
	}
}

// 用例2:模型调 clarify → 事件流以 clarify + done(clarify) 暂停,不执行任何数据工具
func TestStreamChatClarifyPausesStream(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m1","type":"message","role":"assistant","model":"k","stop_reason":"tool_use","content":[
			{"type":"tool_use","id":"c1","name":"clarify","input":{"kind":"missing_param","question":"要查哪个阶段?","param":"stage","candidates":["fff","fdr","fcl"]}}]}`,
	}}
	uc := newAiUcForTest(fake)

	events := collectEvents(t, uc, "失败原因有哪些?", nil)

	var clarifyEv *ChatEvent
	for i := range events {
		if events[i].Type == "clarify" {
			clarifyEv = &events[i]
		}
	}
	if clarifyEv == nil {
		t.Fatalf("no clarify event, got: %s", eventTypes(events))
	}
	if clarifyEv.ID != "c1" || clarifyEv.Args["param"] != "stage" {
		t.Fatalf("clarify event = %+v", clarifyEv)
	}
	last := events[len(events)-1]
	if last.Type != "done" || last.Stop != "clarify" {
		t.Fatalf("last event = %+v, want done/clarify", last)
	}
	if strings.Contains(eventTypes(events), "tool_result") {
		t.Fatalf("clarify should not execute data tools: %s", eventTypes(events))
	}
}

// 用例3:历史回传(assistant tool_use + user tool_result)正确重建,续跑后正常 end_turn
func TestStreamChatHistoryResume(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"FFF 阶段失败 468 次。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	history := []AiChatHistoryMessage{
		{Role: "user", Text: "失败原因有哪些?"},
		{Role: "assistant", ToolCalls: []AiChatToolCall{
			{ID: "c1", Name: "clarify", Args: map[string]any{"kind": "missing_param", "question": "哪个阶段?"}},
		}},
		{Role: "user", ToolResults: []AiChatToolResult{
			{ID: "c1", Content: "用户选择: stage=fff"},
		}},
	}

	events := collectEvents(t, uc, "继续", history)
	last := events[len(events)-1]
	if last.Type != "done" || last.Stop != "end_turn" {
		t.Fatalf("last event = %+v", eventTypes(events))
	}

	// 重建后的消息:3 条历史 + 本轮提问 = 4
	if len(fake.lastReq.Messages) != 4 {
		t.Fatalf("messages = %d, want 4", len(fake.lastReq.Messages))
	}
}

// 用例4:非法历史/空提问被拒
func TestStreamChatInvalidInput(t *testing.T) {
	uc := newAiUcForTest(&chatLlmFake{})
	if err := uc.StreamChat(context.Background(), "  ", nil, func(ChatEvent) error { return nil }); err == nil {
		t.Fatal("empty question should error")
	}
	if err := uc.StreamChat(context.Background(), "q", []AiChatHistoryMessage{{Role: "admin"}}, func(ChatEvent) error { return nil }); err == nil {
		t.Fatal("invalid role should error")
	}
}

// 用例5(线上bug回归):历史里 assistant(tool_use) 缺配对 tool_result 时,
// backfillToolResults 必须自动补占位结果,否则网关 400
func TestStreamChatBackfillsMissingToolResult(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"接上文继续。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	// 历史里 assistant 带两个 tool_use,但下一条 user 只有一个 tool_result(另一个缺失)
	history := []AiChatHistoryMessage{
		{Role: "user", Text: "失败原因?"},
		{Role: "assistant", ToolCalls: []AiChatToolCall{
			{ID: "t1", Name: "get_fail_reason", Args: map[string]any{"stage": "fff"}},
			{ID: "t2", Name: "get_top", Args: map[string]any{"kind": "trigger"}},
		}},
		{Role: "user", ToolResults: []AiChatToolResult{
			{ID: "t1", Content: "结果略"},
		}},
	}

	events := collectEvents(t, uc, "继续", history)
	last := events[len(events)-1]
	if last.Type != "done" || last.Stop != "end_turn" {
		t.Fatalf("stream failed: %s", eventTypes(events))
	}

	// 校验重建后的消息序列:缺失的 t2 获得占位 tool_result,t1 原结果保留
	msgsJSON, _ := json.Marshal(fake.lastReq.Messages)
	if !strings.Contains(string(msgsJSON), `"tool_use_id":"t2"`) {
		t.Fatalf("missing tool_use t2 not backfilled: %s", string(msgsJSON))
	}
	if !strings.Contains(string(msgsJSON), "结果未随历史回传") {
		t.Fatalf("placeholder tool_result missing: %s", string(msgsJSON))
	}
}

// 用例6:纯续跑——question 为空但历史以 tool_result 结尾时允许,且不追加用户文本
func TestStreamChatResumeWithEmptyQuestion(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"已按你的选择查询完成。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	history := []AiChatHistoryMessage{
		{Role: "user", Text: "失败原因?"},
		{Role: "assistant", ToolCalls: []AiChatToolCall{
			{ID: "c1", Name: "clarify", Args: map[string]any{"kind": "missing_param", "question": "哪个阶段?"}},
		}},
		{Role: "user", ToolResults: []AiChatToolResult{
			{ID: "c1", Content: "用户选择: stage=fff"},
		}},
	}

	events := collectEvents(t, uc, "", history)
	if last := events[len(events)-1]; last.Type != "done" || last.Stop != "end_turn" {
		t.Fatalf("resume failed: %s", eventTypes(events))
	}
	// 历史重建后消息数 = 3(不再追加第 4 条用户文本)
	if len(fake.lastReq.Messages) != 3 {
		t.Fatalf("messages = %d, want 3 (no extra user text)", len(fake.lastReq.Messages))
	}

	// 空 question 且历史不以 tool_result 结尾 → 仍应报错
	if err := uc.StreamChat(context.Background(), "", nil, func(ChatEvent) error { return nil }); err == nil {
		t.Fatal("empty question without pending tool_result should error")
	}
}

// 用例7(线上bug回归):前端回传 tool_calls.args(含嵌套结构)重建后,
// input 必须是 JSON 对象而非 base64/字符串(此前 Marshal []byte 导致网关 400)
func TestStreamChatToolUseArgsStayObject(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"好的。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	history := []AiChatHistoryMessage{
		{Role: "user", Text: "fff"},
		{Role: "assistant", ToolCalls: []AiChatToolCall{
			{ID: "c1", Name: "clarify", Args: map[string]any{
				"kind":     "ambiguous_tool",
				"question": "查什么?",
				"options":  []any{map[string]any{"tool": "get_overview", "reason": "概览"}},
			}},
		}},
		{Role: "user", ToolResults: []AiChatToolResult{
			{ID: "c1", Content: "用户选择: 使用 get_overview"},
		}},
	}

	events := collectEvents(t, uc, "继续", history)
	if last := events[len(events)-1]; last.Type != "done" {
		t.Fatalf("stream failed: %s", eventTypes(events))
	}

	msgsJSON, _ := json.Marshal(fake.lastReq.Messages)
	// input 是对象:出现 "input":{"kind" 形态;且不能是 base64 字符串("input":"eyJ…")
	if !strings.Contains(string(msgsJSON), `"input":{"kind"`) {
		t.Fatalf("tool_use input not marshaled as object: %s", string(msgsJSON))
	}
	if strings.Contains(string(msgsJSON), `"input":"eyJ`) {
		t.Fatalf("tool_use input marshaled as base64 string: %s", string(msgsJSON))
	}
}

// 用例8(线上bug回归):孤儿 tool_result(前一条 assistant 无对应 tool_use)必须被清理
func TestStreamChatDropsOrphanToolResult(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"好的。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	// assistant 只有 t1 的 tool_use,但 user 回执里有 t1 和孤儿 c1
	history := []AiChatHistoryMessage{
		{Role: "user", Text: "fff"},
		{Role: "assistant", ToolCalls: []AiChatToolCall{
			{ID: "t1", Name: "get_fail_reason", Args: map[string]any{"stage": "fff"}},
		}},
		{Role: "user", ToolResults: []AiChatToolResult{
			{ID: "t1", Content: "结果略"},
			{ID: "c1", Content: "用户选择: 概览"},
		}},
	}

	events := collectEvents(t, uc, "继续", history)
	if last := events[len(events)-1]; last.Type != "done" {
		t.Fatalf("stream failed: %s", eventTypes(events))
	}

	msgsJSON, _ := json.Marshal(fake.lastReq.Messages)
	if strings.Contains(string(msgsJSON), `"tool_use_id":"c1"`) {
		t.Fatalf("orphan tool_result c1 not dropped: %s", string(msgsJSON))
	}
	if !strings.Contains(string(msgsJSON), `"tool_use_id":"t1"`) {
		t.Fatalf("legit tool_result t1 missing: %s", string(msgsJSON))
	}
}

// 用例9:复杂问题模型提交查询计划 → plan 帧暂停,不执行任何数据工具
func TestStreamChatPlanPausesStream(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m1","type":"message","role":"assistant","model":"k","stop_reason":"tool_use","content":[
			{"type":"tool_use","id":"p1","name":"submit_plan","input":{
				"summary":"对比两周 FFF 失败原因",
				"steps":[
					{"tool":"get_fail_reason","args":{"stage":"fff","start_dt":"2026-08-04"},"purpose":"查上周失败原因"},
					{"tool":"get_fail_reason","args":{"stage":"fff","start_dt":"2026-08-11"},"purpose":"查本周失败原因"}
				]}}]}`,
	}}
	uc := newAiUcForTest(fake)

	events := collectEvents(t, uc, "对比这两周的 FFF 失败原因", nil)

	var planEv *ChatEvent
	for i := range events {
		if events[i].Type == "plan" {
			planEv = &events[i]
		}
	}
	if planEv == nil {
		t.Fatalf("no plan event, got: %s", eventTypes(events))
	}
	if planEv.ID != "p1" || planEv.Args["summary"] != "对比两周 FFF 失败原因" {
		t.Fatalf("plan event = %+v", planEv)
	}
	steps, _ := planEv.Args["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("plan steps = %v", planEv.Args["steps"])
	}
	last := events[len(events)-1]
	if last.Type != "done" || last.Stop != "plan" {
		t.Fatalf("last event = %+v, want done/plan", last)
	}
	if strings.Contains(eventTypes(events), "tool_result") {
		t.Fatalf("plan pause should not execute data tools: %s", eventTypes(events))
	}
}

// 用例10:计划确认回执 → 模型按计划逐步执行数据工具
func TestStreamChatPlanConfirmedExecutesSteps(t *testing.T) {
	fake := &chatLlmFake{rounds: []string{
		`{"id":"m2","type":"message","role":"assistant","model":"k","stop_reason":"tool_use","content":[
			{"type":"tool_use","id":"t1","name":"get_fail_reason","input":{"stage":"fff"}}]}`,
		`{"id":"m3","type":"message","role":"assistant","model":"k","stop_reason":"end_turn","content":[
			{"type":"text","text":"两周对比完成。"}]}`,
	}}
	uc := newAiUcForTest(fake)

	history := []AiChatHistoryMessage{
		{Role: "user", Text: "对比这两周的失败原因"},
		{Role: "assistant", ToolCalls: []AiChatToolCall{
			{ID: "p1", Name: "submit_plan", Args: map[string]any{
				"summary": "对比两周", "steps": []any{map[string]any{"tool": "get_fail_reason"}},
			}},
		}},
		{Role: "user", ToolResults: []AiChatToolResult{
			{ID: "p1", Content: "用户对查询计划的决定: 确认按计划执行"},
		}},
	}

	events := collectEvents(t, uc, "", history)

	got := eventTypes(events)
	if !strings.Contains(got, "tool_call") || !strings.Contains(got, "tool_result") {
		t.Fatalf("plan steps not executed, got: %s", got)
	}
	if last := events[len(events)-1]; last.Type != "done" || last.Stop != "end_turn" {
		t.Fatalf("last event = %+v", events[len(events)-1])
	}
	// 提交的计划作为 tool_use 保留在上下文中(非孤儿)
	msgsJSON, _ := json.Marshal(fake.lastReq.Messages)
	if !strings.Contains(string(msgsJSON), `"name":"submit_plan"`) {
		t.Fatalf("submit_plan tool_use missing from context: %s", string(msgsJSON))
	}
}
