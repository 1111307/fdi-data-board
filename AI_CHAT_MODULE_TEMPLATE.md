# AI 问答模块框架模板(可仿写)

> 从数采看板 AI 问答实战中提炼的**通用 Agent 问答模块骨架**:Go(gin/kratos 分层)+ Anthropic Messages 协议网关 + SSE 流式前端。移植到新场景时,只改"知识包、工具表、前端组件",骨架代码照抄。
> 对应实现:fdi_data_board@feat/ai_summary(后端)、fdi_cloud_unified_web@feat/ai_summary(前端)。

## 0. 模块总览

```
┌─ 前端 (Vue) ──────────────────────────────────────────────────────┐
│ AiChatPanel.vue   对话UI:消息流/工具卡/确认卡/计划卡/思考折叠/停止    │
│ ai.js             consumeSse 通用SSE解析 + streamAiChat(POST SSE)   │
│   消息模型: {role, text, thinking, toolCalls[], toolResults[],      │
│              clarify, plan, hidden}  ← 前端是会话状态的唯一持有者     │
└────────────┬──────────────────────────────────────────────────────┘
             │ POST /xxx/ai/chat {question, messages[]}   (SSE 下行)
┌────────────▼─ 后端 (Go) ───────────────────────────────────────────┐
│ route    EmptyHandlerFunc 注册(裸handler才能流式写)                  │
│ service  ShouldBindJSON → SSE头 → 帧转发(json.Marshal保转义)         │
│ biz      ┌──────────────────────────────────────────────┐          │
│          │ Agent循环 StreamChat(≤N轮):                    │          │
│          │  历史重建→孤儿清理→配对补齐→LLM流式→事件翻译     │          │
│          │  →工具进程内执行→回填续轮 / clarify/plan暂停    │          │
│          ├──────────────────────────────────────────────┤          │
│          │ 工具注册表:clarify+submit_plan(保留)            │          │
│          │           + N个业务工具(直调biz用例)           │          │
│          │ 知识包:knowledge/*.md (go:embed注入system)     │          │
│          ├──────────────────────────────────────────────┤          │
│          │ LlmRepo 接口(Enabled/Model/ChatStream/        │          │
│          │   ChatStreamEx) ← biz只依赖接口                │          │
│ data     └──────────────────────────────────────────────┘          │
│          LlmRepo实现:anthropic-sdk-go + 流事件聚合器                 │
│ conf     data.llm 配置段(proto) → NewData统一建client               │
└────────────────────────────────────────────────────────────────────┘
```

**七种 SSE 下行帧**(前后端契约的核心,仿写时原样保留):

| 帧 | 载荷 | 前端行为 |
|---|---|---|
| `delta` | `{text}` | 正文流式追加(打字机) |
| `thinking` | `{text}` | 思考折叠块追加(纯展示,不回传) |
| `tool_call` | `{id,name,args}` | 工具卡(调用中) |
| `tool_result` | `{id,name,summary,data}` | 工具卡完成 + data 渲染图表 |
| `clarify` | `{id,name,args:{kind,question,candidates/options}}` | 确认卡(暂停点) |
| `plan` | `{id,name,args:{summary,steps[]}}` | 计划卡(暂停点) |
| `done` | `{stop: end_turn/clarify/plan/max_rounds}` | 收尾;stop≠end_turn 时保持卡片 |
| `error` | `{code,message}` | 错误条 |

---

## 1. 知识包层(biz/knowledge/)

**要点**:领域字典(名词、口径、枚举、陷阱)+ 工具选用规则 + 行为纪律。`go:embed` 进二进制,每次请求注入 system,`{{TODAY}}` 运行时替换(模型才能解析"上周")。

```go
//go:embed knowledge/ai_dict.md
var aiKnowledgeFS embed.FS

func knowledgePrompt() string {
    raw, _ := aiKnowledgeFS.ReadFile("knowledge/ai_dict.md")
    return strings.ReplaceAll(string(raw), "{{TODAY}}", time.Now().Format("2006-01-02"))
}
```

**ai_dict.md 骨架(六段式)**:
```markdown
# {领域}字典与工具使用指南
## 一、名词与口径          ← 领域黑话、统计口径、禁算规则
## 二、数据源分层          ← 工具背后的表/服务全景(让模型知道数据从哪来)
## 三、关键枚举值          ← 实测枚举(带中文含义),模型解释数据的依据
## 四、字段陷阱            ← 单位/哨兵值/不可聚合列
## 五、工具清单与选用规则    ← "问题形态→工具"映射表 + 计划纪律(多步必须先出计划)
## 六、行为纪律(最高优先级)  ← ①数字只能来自工具结果 ②不确定必须clarify严禁猜
                              ③思考过程用中文 ④回答结构
```

---

## 2. 工具注册表(biz/ai_tools.go)

**要点**:白名单工具 = Anthropic ToolParam schema + 进程内执行函数;`clarify`/`submit_plan` 是保留工具(exec=nil,触发人工暂停);业务工具直调 biz 用例(继承入口鉴权、口径与页面一致);结果截断防 prompt 膨胀。

```go
type aiToolDef struct {
    def  anthropic.ToolParam
    exec func(ctx context.Context, args map[string]any) (string, error)
}
type aiToolRegistry struct {
    tools map[string]*aiToolDef
    order []string
}
const aiClarifyToolName = "clarify"        // 保留:参数/意图不确定时问用户
const aiSubmitPlanToolName = "submit_plan" // 保留:多步查询先出计划

func newAiToolRegistry(/* 业务usecase依赖 */) *aiToolRegistry {
    r := &aiToolRegistry{tools: map[string]*aiToolDef{}}
    add := func(def anthropic.ToolParam, exec func(context.Context, map[string]any) (string, error)) {
        r.tools[def.Name] = &aiToolDef{def: def, exec: exec}
        r.order = append(r.order, def.Name)
    }

    // ── 保留工具:clarify(仿写原样抄)──
    add(anthropic.ToolParam{
        Name: aiClarifyToolName,
        Description: param.NewOpt("当无法确定调用哪个工具、或关键参数缺失/有歧义时调用…"),
        InputSchema: aiSchema(map[string]any{
            "kind":       map[string]any{"type": "string", "enum": []string{"missing_param", "ambiguous_tool", "confirm_params"}},
            "question":   map[string]any{"type": "string"},
            "candidates": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
            // options(工具歧义)/args(参数确认)…
        }, "kind", "question"),
    }, nil)

    // ── 业务工具:每个工具一个 add(仿写时替换 exec 内容)──
    add(anthropic.ToolParam{
        Name:        "get_fail_reason",
        Description: param.NewOpt("查某阶段失败原因分布。适用:失败原因/为什么失败。"),
        InputSchema: aiSchemaWith(commonProps(), map[string]any{
            "stage": map[string]any{"type": "string", "enum": []string{"fff", "fdr", "fcl"}},
        }, "stage"),
    }, execStageQuery(/*fo, do, "fail_reason"*/)) // 内部:参数归一→直调usecase→marshalToolResult(截断)
    return r
}

// 导出给请求:注意包 ToolUnionParam
func (r *aiToolRegistry) Params() []anthropic.ToolUnionParam {
    out := make([]anthropic.ToolUnionParam, 0, len(r.order))
    for _, name := range r.order {
        def := r.tools[name].def
        out = append(out, anthropic.ToolUnionParam{OfTool: &def})
    }
    return out
}
```

**参数归一化惯例**:日期缺省近 7 天后端补;枚举校验失败返回 error(给模型的 tool_result,它会自纠)。

---

## 3. Agent 循环(biz/use_dashboard_ai_chat.go)——模块心脏

```go
const aiMaxRounds = 6   // 循环轮数熔断
const aiMaxHistory = 40 // 历史条数上限(压缩方案见文末)

func (uc *AiDashboardUseCase) StreamChat(ctx context.Context, question string,
    history []AiChatHistoryMessage, emit func(ChatEvent) error) error {

    msgs, err := buildAnthropicMessages(question, history) // ① 重建历史(见§3.1)
    if err != nil { return err }

    for round := 0; round < aiMaxRounds; round++ {
        if ctx.Err() != nil { return ctx.Err() }            // ② 客户端断开即停(帧间隙)

        params := anthropic.MessageNewParams{
            Model: uc.llmModel(), MaxTokens: 4096,
            System: []anthropic.TextBlockParam{{Text: knowledgePrompt()}}, // 知识包每轮注入
            Messages: msgs, Tools: uc.tools.Params(),
        }

        var textBuf strings.Builder
        var emitErr error                                     // ③ emit短路(见§3.2)
        safeEmit := func(ev ChatEvent) {
            if emitErr != nil { return }
            emitErr = emit(ev)
        }
        msg, err := uc.llm.ChatStreamEx(ctx, params, func(ev anthropic.MessageStreamEventUnion) {
            if ev.Type != "content_block_delta" { return }    // 只处理增量事件
            if t := ev.AsContentBlockDelta().Delta.AsTextDelta().Text; t != "" {
                textBuf.WriteString(t)
                safeEmit(ChatEvent{Type: "delta", Text: t})   // 正文→打字机
                return
            }
            if th := ev.AsContentBlockDelta().Delta.AsThinkingDelta().Thinking; th != "" {
                safeEmit(ChatEvent{Type: "thinking", Text: th}) // 思考→折叠块(不进上下文)
            }
        })
        if err != nil { return fmt.Errorf("llm stream: %w", err) }
        if emitErr != nil { return emitErr }                  // 客户端已断,止损

        var toolUses []anthropic.ToolUseBlock
        for _, block := range msg.Content {                   // ④ 聚合结果里取tool_use
            if tu := block.AsToolUse(); tu.Name != "" { toolUses = append(toolUses, tu) }
        }
        if len(toolUses) == 0 || msg.StopReason != anthropic.StopReasonToolUse {
            return emit(ChatEvent{Type: "done", Stop: "end_turn"}) // 说完即收工
        }

        assistantBlocks := []anthropic.ContentBlockParamUnion{}
        if textBuf.Len() > 0 { assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(textBuf.String())) }
        var resultBlocks []anthropic.ContentBlockParamUnion
        clarifyPaused, planPaused := false, false

        for _, tu := range toolUses {
            assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(tu.ID, tu.Input, tu.Name))
            if emitErr != nil || ctx.Err() != nil { return emitErr } // 工具前止损

            switch tu.Name {
            case aiClarifyToolName:                            // ⑤a 澄清=暂停点
                safeEmit(ChatEvent{Type: "clarify", ID: tu.ID, Name: tu.Name, Args: rawToArgs(tu.Input)})
                clarifyPaused = true
            case aiSubmitPlanToolName:                         // ⑤b 计划=暂停点
                safeEmit(ChatEvent{Type: "plan", ID: tu.ID, Name: tu.Name, Args: rawToArgs(tu.Input)})
                planPaused = true
            default:                                           // ⑥ 业务工具=进程内执行
                safeEmit(ChatEvent{Type: "tool_call", ID: tu.ID, Name: tu.Name, Args: rawToArgs(tu.Input)})
                result, execErr := uc.tools.Exec(ctx, tu.Name, rawToArgs(tu.Input))
                if execErr != nil { result = "工具执行失败: " + execErr.Error() }
                safeEmit(ChatEvent{Type: "tool_result", ID: tu.ID, Name: tu.Name,
                    Summary: truncateStr(result, 120), Data: json.RawMessage(result)})
                if emitErr != nil { return emitErr }
                resultBlocks = append(resultBlocks, anthropic.NewToolResultBlock(tu.ID, result, execErr != nil))
            }
        }
        msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleAssistant, Content: assistantBlocks})
        if planPaused    { return emit(ChatEvent{Type: "done", Stop: "plan"}) }    // 暂停:前端带回执续跑
        if clarifyPaused { return emit(ChatEvent{Type: "done", Stop: "clarify"}) }
        if len(resultBlocks) > 0 {
            msgs = append(msgs, anthropic.MessageParam{Role: anthropic.MessageParamRoleUser, Content: resultBlocks})
        }
    }
    return emit(ChatEvent{Type: "done", Stop: "max_rounds"})
}
```

### 3.1 历史重建(协议正确性的命门)

前端回传轻量历史 `{role, text, tool_calls[], tool_results[]}`,后端重建为协议消息,**两道兜底缺一不可**:

```go
func buildAnthropicMessages(question string, history []AiChatHistoryMessage) ([]anthropic.MessageParam, error) {
    q := strings.TrimSpace(question)
    if q == "" && !historyEndsWithToolResult(history) {   // 空 question 仅允许"纯续跑"(以回执结尾)
        return nil, fmt.Errorf("question 不能为空")
    }
    // …逐条转 content blocks:assistant→NewTextBlock+NewToolUseBlock(id, Args, name)
    //                        user→NewToolResultBlock / NewTextBlock
    // ★ NewToolUseBlock 的 input 必须传 map(传 Marshal 后的 []byte 会被序列化成 base64 字符串→网关400)
    if q != "" { msgs = append(msgs, user(q)) }
    msgs = dropOrphanToolResults(msgs)   // 兜底①:孤儿清理——user 的 tool_result 配不上前面 assistant 的 tool_use 就丢
    return backfillToolResults(msgs), nil // 兜底②:配对补齐——assistant 的 tool_use 缺紧邻 tool_result 就补占位
}
```

> 为什么必须两道:Anthropic 协议硬性要求 **assistant.tool_use 与紧邻下一条 user.tool_result 一一配对**,前端任何组装失误都会 400。drop 处理"多出的 result",backfill 处理"缺失的 result"。

### 3.2 emit 短路(止损第二腿)

停止的完整链:`abort → TCP①断 → req ctx cancel → ChatStreamEx 报错(腿①)` + `写死连接 Flush 报错 → emitErr(腿②)`。任何一条腿先到,循环立即终止,不再执行工具/续轮。

---

## 4. LLM 仓储(biz 接口 / data 实现)

```go
// biz 定义接口(业务层不依赖 SDK)
type LlmRepo interface {
    Enabled() bool
    Model() string
    ChatStream(ctx context.Context, system, user string, onDelta func(string)) error // 简单场景(总结)
    ChatStreamEx(ctx context.Context, params anthropic.MessageNewParams,
        onEvent func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error)  // agent场景
}
```

```go
// data 实现:client 在 NewData 统一构建(和 mysql/doris 同款)
func newLlmClient(c *conf.Data, logger log.Logger) *anthropic.Client {
    lc := c.GetLlm()
    if lc == nil || lc.GetBaseUrl() == "" || lc.GetApiKey() == "" || lc.GetModel() == "" {
        log.NewHelper(logger).Warn("[llm] 未配置,降级") // 未配置→nil→上层走本地兜底
        return nil
    }
    client := anthropic.NewClient(option.WithBaseURL(lc.GetBaseUrl()),
        option.WithAPIKey(lc.GetApiKey()), option.WithRequestTimeout(timeout))
    return &client
}
```

**ChatStreamEx 核心 = 边直播边聚合**:

```go
func (r *LlmRepo) ChatStreamEx(ctx context.Context, params anthropic.MessageNewParams,
    onEvent func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error) {
    stream := r.data.llmClient.Messages.NewStreaming(ctx, params) // ctx绑外呼:cancel即断下游TCP
    defer stream.Close()
    aggregator := newMessageAggregator()          // message_start→骨架; text_delta→拼正文;
    for stream.Next() {                           // input_json_delta→拼tool参数; message_delta→stop_reason
        event := stream.Current()
        aggregator.feed(event)
        if onEvent != nil { onEvent(event) }      // 实时直播(同一事件,两个消费者)
    }
    if err := stream.Err(); err != nil { return nil, fmt.Errorf("llm stream: %w", err) }
    return aggregator.message(), nil              // 碎片拼回完整Message(给循环做决策)
}
```

---

## 5. SSE 服务层(service + route)

```go
// route:必须用 EmptyHandlerFunc(裸 gin handler)——JsonHandlerFunc 会被包装成一次性 c.JSON
{EmptyHandlerFunc: s.StreamChat, Path: "chat", Method: POST},

// service
func (s *AiDashboardService) StreamChat(c *gin.Context) {
    var req dashboard_api.AiChatRequest
    if err := c.ShouldBindJSON(&req); err != nil { writeSseError(c.Writer, 400, err.Error()); return }
    c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("X-Accel-Buffering", "no") // 防nginx缓冲
    c.Writer.WriteHeader(http.StatusOK); flushWriter(c.Writer)

    err := s.uc.StreamChat(c.Request.Context(), req.Question, toBizHistory(req), func(ev biz.ChatEvent) error {
        payload, _ := json.Marshal(ev)               // ★ json.Marshal 转义(勿用 fmt %q——非严格JSON)
        _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Type, payload)
        flushWriter(c.Writer)                         // 每帧Flush:活着=推流;死了=暴露写错误(emit短路)
        return err
    })
    if err != nil {
        log.Errorf("StreamChat error: %v, question: %s", err, req.Question) // ★ 必须记日志(排障命门)
        writeSseError(c.Writer, 500, "AI 问答失败: "+err.Error())
    }
}
```

**配置**(conf.proto + yaml,环境变量可覆盖):
```yaml
data:
  llm:
    base_url: ${LLM_BASE_URL:https://your-gateway}   # Anthropic Messages 协议网关
    api_key:  ${LLM_API_KEY:}
    model:    ${LLM_MODEL:glm-5.3}
    max_tokens: ${LLM_MAX_TOKENS:8192}               # thinking 模型给思考留余量
    timeout:  ${LLM_TIMEOUT:360s}
server:
  gin:
    timeout: 360s   # ★ SSE 长流需放宽(kratos Timeout 是 server 级,无法按路由豁免)
```

---

## 6. 前端(api/dashboard/ai.js)

```js
// 通用 SSE 消费:fetch + ReadableStream(axios 0.18 不支持流式)
export async function streamAiChat(body, handlers = {}, signal) {
  const headers = { Accept: 'text/event-stream', 'Content-Type': 'application/json' }
  if (store.getters.bearer_token) headers.authorization = store.getters.bearer_token
  const resp = await fetch(url, { method: 'POST', headers, body: JSON.stringify(body), signal })
  const reader = resp.body.getReader(), decoder = new TextDecoder()
  let buffer = ''
  const consumeFrames = () => { /* 按\n\n分帧→提取event:/data:→JSON.parse
                                    →onFrame(event,payload); delta/done/error 分发 */ }
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true }); consumeFrames()
  }
}
// 中断:AbortController.abort() → TCP撕毁 → 服务端ctx取消 → 下游TCP同断(止损全链)
```

**消息模型与历史组装规则**(组件内,仿写的核心逻辑):

```js
// 发送铁律:先快照后 push —— 否则本轮提问同时进 messages 和 question(双份)
const history = this.historyForBackend()      // ① 快照
this.messages.push({ role: 'user', text: q }) // ② 本轮入列
this.send(q, history)                         // ③ 用快照发送

// historyForBackend 输出(与后端对齐):
// user:      text 或 tool_results(二选一)
// assistant: text + tool_calls[]
// ★ clarify/plan 无论是否已回答,都要作为 tool_use 回传(回执依赖它配对,漏传=孤儿400)
// ★ 相邻同文本 user 去重(脏数据自愈) + thinking 永不回传(纯展示)
```

---

## 7. 移植 checklist(仿写时改哪里)

| 步骤 | 动作 | 不变的 |
|---|---|---|
| 1 | 写领域知识包(六段式 md) | embed 机制、{{TODAY}} |
| 2 | 替换业务工具表(schema+exec 直调你的 usecase) | clarify/submit_plan、Params()/Exec() 骨架 |
| 3 | 挂路由+service(帧转发原样) | EmptyHandlerFunc、七帧协议 |
| 4 | conf.proto 加 llm 段 + NewData 建 client | 降级逻辑 |
| 5 | 前端组件改名接入(消息模型照搬) | ai.js、历史组装规则、abort 链 |
| 6 | 联调:单步查询→clarify 闭环→计划卡→停止→追问(五条路径全过) | — |

## 8. 实战踩坑清单(全部真实踩过,仿写必防)

1. **tool_use.input 传了 `json.Marshal` 的 []byte → SDK 序列化成 base64 字符串 → 网关 400**。input 直接传 map;
2. **孤儿 tool_result**(前端漏传 assistant.tool_use)→ 400。`dropOrphanToolResults` 兜底 + 前端"clarify/plan 必回传";
3. **tool_result.content 类型**:后端用 `json.RawMessage` 兼容 string/object,前端归一为字符串;
4. **gin 全局 35s 超时掐 SSE**:kratos Timeout 是 server 级 ctx,响应头已 flush 无法接管,表现为 LLM 流 `context deadline exceeded`——放宽到 ≥360s;
5. **emit 错误勿忽略**:写死连接失败 = 客户端已走,立即短路(不烧 token 不查库);
6. **前端发送时序**:历史快照必须先于 push;
7. **中文输入法回车双发**:`keydown.enter + isComposing/keyCode 229` 检查 + 同文本去重;
8. **渲染 v-if 必须限定 role**:`v-if="msg.text"` 会让 user 消息同时命中插值和 markdown 两个渲染点(显示两遍);
9. **停止链验证**:abort → 后端日志 `context canceled` 即刻出现 = ctx 传导正常;
10. **知识包纪律 > 模型能力**:数字只来自工具/不确定必须问/思考中文——写进 system 比换模型有效。

## 9. 已知边界与升级路线(设计时预留)

- **上下文压缩**:现为 40 条砍头;升级两步走——①模板式 tool_result 瘦身(保轮对齐,禁切 tool_use/result 配对中间)②LLM 摘要(保数字结论/用户意图/未决事项,摘要缓存前端);
- **会话持久化**:现为前端持有(刷新即丢);加服务端存储只需把 messages 序列化入库,协议零改动;
- **prompt caching**:需先确认网关是否透传 Anthropic cache_control,否则白做;
- **多模型 fallback**:网关侧能力,模块不感知(换 model 只改配置)。
