# AI 问答模块设计文档

> 范围:数采链路看板「✨ AI 问答」功能,横跨 fdi_cloud_unified_web(前端)与 fdi_data_board(看板后端),AI 供应商为 Anthropic Messages 协议网关(模型 kimi-k3)。

## 1. 背景

数采链路看板已有 FFF/FDR/FCL 三阶段的指标、趋势、失败原因等可视化页面,但存在两个使用痛点:

1. **数据要人找**:用户带着问题(“最近失败变多了吗?”“哪个车型落盘最慢?”)进来,需要自己切 tab、选筛选器、看图表、脑内拼接结论;
2. **口径门槛高**:三阶段独立口径、`__ALL__` 哨兵、P95 不可再聚合、vehicle_count 不可跨天 SUM 等数仓口径细节,非数据团队用户容易误读。

因此做 AI 问答:用户用自然语言提问,后端 Agent 根据数仓字典选择合适的看板聚合接口取数,流式生成带数字、带口径、带来源的解释;参数不确定时向用户发起确认(human-in-the-loop),确认后继续执行。

## 2. 目的

| 目的 | 说明 |
|---|---|
| 自然语言取数 | "近7天 FFF 失败原因分布"→ 自动调用 get_fail_reason → 出结论 |
| 口径可靠 | AI 只能调用白名单内的看板聚合方法(口径与页面一致),数字 100% 来自工具结果,禁止编造 |
| 人工确认(HITL) | 参数缺失/工具歧义时 AI 主动发 clarify,用户点选后从断点续跑,严禁猜参数 |
| 结果可溯源 | 每次工具调用在前端渲染成工具卡(接口名+参数+结果摘要),图表标注数据来源 |
| 架构可演进 | Anthropic 原生 tool_use/tool_result 协议 + 前端持历史无状态后端,同构 LangGraph interrupt/resume,后续可平滑升级服务端会话/上下文压缩 |

## 3. 时序图

三角色:前端(浏览器)、board 服务(fdi_data_board)、AI 供应商(llm-gateway,Anthropic Messages 协议)。

### 3.1 主流程:提问 → 工具执行 → 流式回答

```
 用户                前端(AiChatPanel)           board服务(fdi_data_board)         AI供应商(kimi-k3)
  │                        │                              │                              │
  │ 提问"近7天失败原因?"    │                              │                              │
  ├───────────────────────>│                              │                              │
  │                        │ POST /dashboard/v1/ai/chat   │                              │
  │                        │ {question, messages:历史}    │                              │
  │                        ├─────────────────────────────>│ keycloak 鉴权(同看板接口)     │
  │                        │                              │ POST /v1/messages (stream)   │
  │                        │                              │ system=数仓字典+工具schema     │
  │                        │                              ├─────────────────────────────>│
  │                        │                              │                              │
  │                        │                              │<─ text 增量(content_block_delta)
  │                        │<─ event: delta {text} ───────┤                              │
  │  (流式渲染文本)         │                              │                              │
  │                        │                              │<─ tool_use: get_fail_reason   │
  │                        │                              │   (stage 缺失,无法推断)       │
  │                        │                              │                              │
  │                        │                              │ (本轮结束,带上 tool_use)      │
  │                        │<─ event: clarify ────────────┤                              │
  │                        │   {kind,question,candidates} │                              │
  │                        │<─ event: done {stop:clarify} ┤                              │
  │  (渲染确认卡:fff/fdr/fcl 候选按钮)                      │                              │
  │                        │  ‖ 流结束,等待用户            │                              │
  │                        │                              │                              │
  │ 点击"fff"              │                              │                              │
  ├───────────────────────>│ 历史追加:                     │                              │
  │                        │  assistant(tool_use:clarify) │                              │
  │                        │  user(tool_result:用户选择)   │                              │
  │                        │ POST /ai/chat                │                              │
  │                        │ {question:"", messages}      │                              │
  │                        ├─────────────────────────────>│ POST /v1/messages            │
  │                        │                              │ (历史以tool_result结尾=续跑)  │
  │                        │                              ├─────────────────────────────>│
  │                        │                              │                              │
  │                        │                              │<─ tool_use: get_fail_reason(stage=fff)
  │                        │<─ event: tool_call ──────────┤                              │
  │                        │  {name,args}                 │──┐ 进程内执行 biz 方法         │
  │                        │                              │   │ 查 Doris 汇总表(口径=页面) │
  │                        │                              │<──┘                            │
  │                        │<─ event: tool_result ────────┤                               │
  │                        │  {summary,data}              │ POST /v1/messages(下一轮)     │
  │  (渲染工具卡+图表)       │                              │ messages 追加 tool_result     │
  │                        │                              ├─────────────────────────────>│
  │                        │                              │<─ 结论文本流                   │
  │                        │<─ event: delta... ───────────┤                               │
  │                        │<─ event: done{end_turn} ─────┤                               │
  │  (得到最终回答)          │                              │                              │
```

### 3.2 分支:参数齐全,无需确认

```
 用户       前端                      board服务                     AI供应商
  │           │                          │                           │
  │           │ POST /ai/chat            │                           │
  │           │ {question:"近7天FFF失败原因?", messages:[]}          │
  │           ├─────────────────────────>│ POST /v1/messages        │
  │           │                          ├─────────────────────────>│
  │           │                          │<─ tool_use: get_fail_reason{stage:"fff"}
  │           │<─ tool_call ─────────────┤                           │
  │           │                          │──┐ 进程内执行              │
  │           │                          │<─┘ (FO/DO用例 → Doris)   │
  │           │<─ tool_result{summary,data}                          │
  │           │                          │ POST /v1/messages        │
  │           │                          ├─────────────────────────>│
  │           │                          │<─ 结论文本流              │
  │           │<─ delta... → done{end_turn}                          │
```

### 3.3 分支:用户点「停止」/连接中断

```
 用户       前端                      board服务                     AI供应商
  │           │                          │                           │
  │           │ POST /ai/chat (SSE进行中) │                           │
  │           ├─────────────────────────>│ ── 流式转发中 ──>          │
  │           │                          │                           │
  │ 点「停止」 │                          │                           │
  ├──────────>│ AbortController.abort()  │                           │
  │           │ ── 掐断 TCP ─────────────>│ req ctx 取消              │
  │           │                          │ ── 上游 LLM 请求随之取消 ──>│ (不白烧 token)
  │           │                          │ Agent 循环退出,静默收尾    │
```

## 4. 流程

### 4.1 前端流程(AiChatPanel.vue)

```
用户提问 / 点选确认卡
   │
   ├─ 组装请求体 {question, messages}
   │    messages 来自本地会话数组,规则:
   │    · user.Text → 用户提问
   │    · assistant.Text + tool_calls[] → AI 回答与工具调用
   │    · user.tool_results[] → 工具结果回执(隐形消息) / clarify 确认回执
   │    · 每轮 done 后自动落一条隐形 user 消息保存本轮工具结果(与 tool_use 配对)
   │
   ├─ fetch POST /ai/chat (SSE 读流,带 keycloak bearer)
   │    逐帧分发:
   │    delta      → 追加文本,markdown 流式渲染
   │    tool_call  → 渲染工具卡(调用中)
   │    tool_result→ 卡片转完成 + 按 data 渲染 ECharts
   │                 (get_fail_reason→饼图 / get_top→柱状 / get_stage_trend→折线)
   │    clarify    → 渲染确认卡(缺参候选/工具歧义/参数确认三种形态)
   │    done       → 结束;stop=clarify 保持确认卡等待点选
   │    error      → 显示错误条
   │
   ├─ clarify 点选 → 历史追加回执 → 空 question 重新请求(纯续跑)
   └─ 停止 → AbortController → 中断读流(连接销毁不复用)
```

### 4.2 后端流程(fdi_data_board)

```
POST /dashboard/v1/ai/chat (route: EmptyHandlerFunc 裸 handler,可流式写)
   │
   ├─ service: ShouldBindJSON → SSE 响应头(text/event-stream)→ 转发事件帧
   │
   ├─ biz StreamChat: Agent 循环(上限 6 轮)
   │    ① buildAnthropicMessages:
   │         历史校验(≤40条,role白名单)→ 重建 Anthropic content blocks
   │         空 question 且历史以 tool_result 结尾 → 纯续跑(不追加文本)
   │         backfillToolResults: 兜底为缺失配对的 tool_use 补占位 tool_result(协议保合法)
   │    ② 调 LLM(ChatStreamEx):
   │         system = ai_dict.md 数仓字典(go:embed,{{TODAY}} 运行时替换)
   │         tools  = 7 个工具 schema(clarify + 6 个数据工具)
   │         流式聚合: text_delta 直发 delta 帧;input_json_delta 拼装 tool_use
   │    ③ 按 stop_reason 分流:
   │         end_turn   → done 帧收尾
   │         tool_use   → 逐个处理:
   │            · clarify  → emit clarify 帧 → done{stop:clarify} → 本流结束(暂停点)
   │            · 数据工具 → emit tool_call → 进程内直调 FO/DO 用例(继承入口鉴权,
   │              口径与看板一致,Top 截断 10~15 行)→ emit tool_result{summary,data}
   │              → tool_result 回填 messages → 继续下一轮
   │
   └─ data LlmRepo(anthropic-sdk-go):
        client 在 NewData 统一构建(conf.proto data.llm 段)
        三项(base_url/api_key/model)未配置 → 降级:总结走本地统计模式
```

### 4.3 关键设计决策

| 决策 | 理由 |
|---|---|
| 工具=进程内直调 biz 方法,非 HTTP 回环 | 无 token 二次传递问题;入口 keycloak 鉴权一次,授权自然继承;AI 能查的数据 = 用户页面能看的数据 |
| 历史由前端持有,后端无状态 | 与 Anthropic/Vercel AI SDK 官方模式一致;天然支持水平扩展;升级服务端会话只加存储层,协议不变 |
| clarify 走"流结束+新请求续跑",不长挂 SSE | SSE 无回写通道;同构 LangGraph interrupt/Command(resume);不受代理空闲超时影响 |
| 数仓字典随每次请求注入 system | LLM 无状态,每次决策都有完整口径;字典为实测 DDL+枚举值整理 |
| 数字纪律:只能引用工具结果 | 口径可靠性的根;字典附单位换算与聚合陷阱,防误读 |
