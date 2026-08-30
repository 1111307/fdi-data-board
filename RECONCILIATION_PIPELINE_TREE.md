# 全链路树形可视化设计（tar包 → cdi解析 → kafka → ETL消费 → 落库）

> 在 [RECONCILIATION_API_SPEC.md](RECONCILIATION_API_SPEC.md) 已有的三表对账模型基础上，新增一个
> **树形聚合接口**，把 L1/L2/L3 的扁平计数重组成前端可以直接渲染的父子节点结构，用于"这条链路每一步
> 分流去哪了"的可视化。不引入新表、不改写入逻辑，纯查询侧重组。

---

## 一、树的层级设计

按用户给的分流方式，映射到三张表已有的字段：

```
收到的tar包                                    ← fis_decode_record 总数（L1）
├── 解析成功tar包  (status IN (1,3) success+partial)
│     │  meta: status_line_count / skip_line_count / parsed_line_count 聚合值（信息性，不参与子节点求和）
│     │
│     ├── 解析成功event  (fis_decode_detail.send_status=1)   ← L2
│     │     ├── 落库成功    (fis_consume_record.status=1)             ← L3
│     │     ├── 转换失败    (status=2, err_detail LIKE '3:%')
│     │     ├── 落库失败    (status=2, err_detail LIKE '6:%')
│     │     └── 待落库/丢库 (consume 无对应行, missing)
│     │
│     └── 解析失败event  (fis_decode_detail.send_status=2，即上游发下游 Kafka 失败)
│           failure_reasons: 按 err_detail 聚合
│
└── 解析失败tar包  (fis_decode_record.status=2)
      failure_reasons: 按 stage 聚合
```

**关键说明（避免和用户口语描述产生歧义）**：

- 用户描述里的"event 解析成功/失败"对应表里的 `send_status`；这张表的 `send_status` 本身就是"解析
  出 event 并尝试发到下游 Kafka"这一个原子动作的结果（1=两步都成功，2=Kafka 发送失败），当前实现
  **不单独区分"解析"和"发送"两个子步骤**——所以树上只有一层节点，不是"解析"下面再分"发送"。
- 用户描述里"发送成功/发送失败节点等等"，对应的是 L3 落库结果（`fis_consume_record`），因为"发到
  Kafka 之后、下游真正处理完"才是这条链路的下一个可观测节点，"等等"覆盖的就是 `convert_failed` /
  `landing_failed` / `missing` 这三种非"完全成功"的落库结果。
- `skip_line_count`（json 解析失败跳过的原始行数）是目前唯一能反映"tar 包内部有多少行连 event 都
  没解出来"的信号，但它是 **bag 级聚合计数，没有逐行失败原因**，且统计口径覆盖所有 module（含非
  trigger 类），不能和 `fis_decode_detail`（只覆盖 trigger 类 module）的行数精确对应相加。所以把它
  放在"解析成功tar包"节点的 `meta` 里作**信息性展示**，不作为严格的树形子节点参与计数求和，避免
  子节点之和 ≠ 父节点这种误导性的假不一致。如果之后需要逐行失败原因，需要上游在 skip 时额外记录
  一个原因码，这是一个新的埋点需求，当前表结构不支持。

---

## 二、树节点 JSON 结构

```ts
interface PipelineNode {
  key: string;                // 稳定 id，前端用作 React/Vue key 及展开状态缓存键
  label: string;               // 展示名，如"解析成功tar包"
  status: "root" | "success" | "failed" | "partial" | "unknown";
  count: number;
  rate?: number;                // 相对于直接父节点的占比，0~1，root 节点没有
  meta?: Record<string, number>;      // 信息性附加数据，不参与子节点求和校验
  failure_reasons?: FailureReason[];  // 只有失败类叶子节点才有
  children?: PipelineNode[];
}

interface FailureReason {
  code: string;         // stage 编号或 err_detail 前缀码，如 "1" "3" "6"
  desc: string;          // 人话描述，后端映射，前端不猜数字（沿用 API_SPEC 3.11 的枚举表）
  module_name?: string;  // L2/L3 失败原因按 module_name 分桶
  sample: string;        // 该分桶下任取一条 error_msg/err_detail 作样例
  count: number;
}
```

**不变量（前端/QA 都可以拿来做校验）**：同一层的 `children[].count` 之和必须等于父节点的
`count`（`meta` 字段不算在内）。后端实现时可以直接拿这个不变量写单测，比人工核对 SQL 更可靠。

---

## 三、新增接口

```
GET /api/reconcile/pipeline-tree?date=2026-07-08[&project=proj-a][&module_name=fff_close][&md5=abc123]
```

| 参数 | 说明 |
|---|---|
| `date` | 必填，默认当天，格式 `YYYY-MM-DD` |
| `project` | 可选，过滤到单个项目山头；作用于 L1（`fis_decode_record.project`）和 L2/L3 |
| `module_name` | 可选，过滤到单个模块；**只作用于 L2/L3**（`fis_decode_record` 是 bag 级，一个 bag 可能含多个 module，不做过滤，只影响其下 event 层的分支） |
| `md5` | 可选，**单包下钻模式**：把树收窄到某一个具体 tar 包的完整生命周期，`tar_received.count` 恒为 1，用于从 `/md5` 差异排行或 `/md5/{md5}` 明细页跳转过来时"看这一个包完整走了哪条路" |

> `module_name`/`project`/`md5` 均走参数化查询（prepared statement 或安全的字符串校验+转义），
> 禁止拼接进 SQL；`date` 用 `time.Parse("2006-01-02", ...)` 校验后再使用——和 API_SPEC 里其余
> 接口的安全约定一致。

### 返回示例

```json
{
  "date": "2026-07-08",
  "filters": {"project": null, "module_name": null, "md5": null},
  "tree": {
    "key": "tar_received",
    "label": "收到的tar包",
    "status": "root",
    "count": 5000,
    "children": [
      {
        "key": "tar_parse_success",
        "label": "解析成功tar包",
        "status": "success",
        "count": 4985,
        "rate": 0.9970,
        "meta": {
          "decode_success": 4980,
          "decode_partial": 5,
          "status_line_count": 130000,
          "skip_line_count": 42,
          "parsed_line_count": 129958
        },
        "children": [
          {
            "key": "event_parse_success",
            "label": "解析成功event",
            "status": "success",
            "count": 12322,
            "rate": 0.9994,
            "children": [
              {"key": "event_land_matched",        "label": "落库成功",     "status": "success", "count": 12312, "rate": 0.9992},
              {"key": "event_land_convert_failed",  "label": "转换失败",     "status": "failed",  "count": 3,     "rate": 0.0002,
                "failure_reasons": [{"code": "3", "desc": "下游转换失败", "module_name": "fdr_status", "sample": "3:trigger struct convert failed", "count": 2}]},
              {"key": "event_land_landing_failed",  "label": "落库失败",     "status": "failed",  "count": 2,     "rate": 0.0002,
                "failure_reasons": [{"code": "6", "desc": "落库失败(DLQ/丢弃)", "module_name": "fff_close", "sample": "6:no partition for this tuple", "count": 2}]},
              {"key": "event_land_missing",         "label": "待落库/丢库",   "status": "unknown", "count": 5,     "rate": 0.0004}
            ]
          },
          {
            "key": "event_parse_failed",
            "label": "解析失败event",
            "status": "failed",
            "count": 7,
            "rate": 0.0006,
            "failure_reasons": [
              {"code": "5", "desc": "发送下游Kafka失败", "module_name": "fff_close", "sample": "5:broker unavailable:192.168.1.1:9092", "count": 3}
            ]
          }
        ]
      },
      {
        "key": "tar_parse_failed",
        "label": "解析失败tar包",
        "status": "failed",
        "count": 15,
        "rate": 0.0030,
        "failure_reasons": [
          {"code": "1", "desc": "下载解包失败", "sample": "1:connection timeout", "count": 5},
          {"code": "2", "desc": "内容为空",     "sample": "2:statusList is empty", "count": 10}
        ]
      }
    ]
  }
}
```

单包下钻模式（`&md5=abc123`）返回结构完全一致，只是 `tar_received.count=1`，且 event 层节点直接
是该 md5 下的 event 列表聚合（通常 event 数量很小，前端可以把 `failure_reasons` 直接当明细列表展示，
不用再跳转）。

---

## 四、SQL（复用 API_SPEC 已有查询，只是拆成按层取数）

### 4.1 L1 bag 层

```sql
SELECT
  COUNT(*) AS total,
  SUM(CASE WHEN status IN (1,3) THEN 1 ELSE 0 END) AS parse_success,
  SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) AS decode_success,
  SUM(CASE WHEN status=3 THEN 1 ELSE 0 END) AS decode_partial,
  SUM(CASE WHEN status=2 THEN 1 ELSE 0 END) AS decode_failed,
  SUM(status_line_count) AS status_line_count,
  SUM(skip_line_count)   AS skip_line_count,
  SUM(parsed_line_count) AS parsed_line_count
FROM fdi_dev.fis_decode_record
WHERE dt = '2026-07-08'
  -- AND project = 'proj-a'
  -- AND md5 = 'abc123'
;
```

```sql
-- tar_parse_failed 的 failure_reasons（仅 decode_failed>0 时才查）
SELECT stage, COUNT(*) AS count, ANY_VALUE(error_msg) AS sample
FROM fdi_dev.fis_decode_record
WHERE dt = '2026-07-08' AND status = 2
  -- AND project = 'proj-a'  -- AND md5 = 'abc123'
GROUP BY stage
ORDER BY count DESC;
```

### 4.2 L2 event 解析层

```sql
SELECT
  SUM(CASE WHEN send_status=1 THEN 1 ELSE 0 END) AS parse_success,
  SUM(CASE WHEN send_status=2 THEN 1 ELSE 0 END) AS parse_failed
FROM fdi_dev.fis_decode_detail
WHERE dt = '2026-07-08'
  -- AND project = 'proj-a'  -- AND module_name = 'fff_close'  -- AND md5 = 'abc123'
;
```

```sql
-- event_parse_failed 的 failure_reasons（仅 parse_failed>0 时才查）
SELECT module_name, err_detail, COUNT(*) AS count
FROM fdi_dev.fis_decode_detail
WHERE dt = '2026-07-08' AND send_status = 2
  -- AND project = 'proj-a'  -- AND module_name = 'fff_close'  -- AND md5 = 'abc123'
GROUP BY module_name, err_detail
ORDER BY count DESC
LIMIT 50;
```

### 4.3 L3 event 落库层（只看 send_status=1 的 event，即 event_parse_success 分支）

```sql
SELECT
  SUM(CASE WHEN c.status=1 THEN 1 ELSE 0 END) AS matched,
  SUM(CASE WHEN c.status=2 AND c.err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
  SUM(CASE WHEN c.status=2 AND c.err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
  SUM(CASE WHEN c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08' AND d.send_status = 1
  -- AND d.project = 'proj-a'  -- AND d.module_name = 'fff_close'  -- AND d.md5 = 'abc123'
;
```

```sql
-- convert_failed / landing_failed 的 failure_reasons，同 API_SPEC 3.11 的两条查询，加同样的过滤条件
SELECT module_name, err_detail, COUNT(*) AS count
FROM fdi_dev.fis_consume_record
WHERE dt = '2026-07-08' AND status = 2 AND err_detail LIKE '3:%'   -- 落库失败换成 '6:%'
  -- AND project = 'proj-a'  -- AND module_name = 'fff_close'  -- AND md5 = 'abc123'
GROUP BY module_name, err_detail
ORDER BY count DESC
LIMIT 50;
```

一共 **6 条查询**（3 条计数 + 3 条失败原因明细，失败原因明细可以在计数=0 时跳过，减少一半查询），
后端组装成上面的嵌套 JSON。`stage`/`err_detail` 前缀码 → `desc` 的映射直接复用 API_SPEC 3.11 已有
的两张枚举表，不要再定义一套新的。

### 4.4 缓存

和 `/overview` 一样的量级、一样的刷新频率，1~5 分钟内存/Redis 缓存即可；`md5` 下钻模式数据量极小，
可以不缓存或缓存更短时间（比如 30s），因为通常是排查问题时高频刷新看最新状态。

---

## 五、前端实现

### 5.1 数据消费方式

接口一次性返回整棵树，前端**不需要**为每层单独发请求、不需要自己做聚合/求和——直接拿 `tree` 字段
渲染。展开/收起是纯前端状态（哪些 `key` 处于展开），不触发新请求。只有点击"待落库/丢库"、"转换
失败"等叶子节点想看**明细列表**时，才调用已有的 `/api/reconcile/event-list?type=...` 或
`/api/reconcile/md5/{md5}` 做二次请求（这两个接口本来就有分页），树形接口只负责"分流去哪了、去了
多少"，不负责"具体是哪几条"。

### 5.2 可视化方案

三种实现复杂度递增，按团队现状选：

**方案 A（推荐，成本最低）：递归可折叠节点树 + 自定义 CSS，不依赖图表库。**
每个 `PipelineNode` 渲染成一个卡片：`label` + `count` + `rate` 进度条 + 失败节点标红/加警示图标，
`children` 递归渲染，用一条竖线/横线连接（纯 CSS `border-left` 画连接线即可，不需要 canvas/svg 布局
引擎）。这是目前大多数内部数据看板（包括这个仓库已有的 `RECONCILIATION_DASHBOARD_DESIGN.md` 里
的表格+饼图布局）采用的复杂度级别，用 React/Vue 的组件递归 + 一份数据结构就能做，不引入新依赖，
维护成本最低，且已经能满足"点开看分流、点失败节点看原因"的核心需求。

```tsx
// React 示例，框架无关的思路：拿到 tree 后递归渲染，antd 版本用 Tree 组件也可以直接吃这个数据结构
function PipelineTreeNode({ node }: { node: PipelineNode }) {
  const [expanded, setExpanded] = useState(true);
  const isFailure = node.status === "failed";
  return (
    <div className="pipeline-node">
      <div
        className={`pipeline-node__card ${isFailure ? "is-failure" : ""}`}
        onClick={() => node.children && setExpanded(!expanded)}
      >
        <span className="pipeline-node__label">{node.label}</span>
        <span className="pipeline-node__count">{node.count.toLocaleString()}</span>
        {node.rate != null && <span className="pipeline-node__rate">{(node.rate * 100).toFixed(2)}%</span>}
        {node.failure_reasons && (
          <FailureReasonPopover reasons={node.failure_reasons} />
        )}
      </div>
      {expanded && node.children && (
        <div className="pipeline-node__children">
          {node.children.map(c => <PipelineTreeNode key={c.key} node={c} />)}
        </div>
      )}
    </div>
  );
}
```

`FailureReasonPopover` 点击失败节点弹出 `failure_reasons` 列表（code/desc/module_name/sample/count），
每一行可以再加一个"查看明细"链接跳到 `/event-list?type=convert_failed&module_name=...`。

**方案 B：ECharts `tree` 系列。** 如果团队已经在用 ECharts（大概率，因为已有的对账看板里趋势图/
饼图很可能也是 ECharts），可以直接把这棵树喂给 `series: [{type: 'tree', data: [...]}]`，好处是自带
布局算法（横向树状图）和缩放/拖拽，坏处是节点内容自定义程度不如方案 A 高（想在节点内画进度条、
失败原因图标要用 `tooltip.formatter` 或 `label.rich`，比较绕）。适合"先出效果图/demo"，长期看方案 A
更好维护。

**方案 C：AntV G6 / X6。** 如果需要更复杂的横向 DAG 布局（比如以后要把 kafka 分区、consumer group
延迟这些运维指标也画进同一张图，变成真正的"链路拓扑图"而不只是"计数分流树"），上专门的图可视化
引擎更合适，但学习成本和开发量明显高于 A/B，现阶段这个需求（分层计数 + 失败原因）不需要上到这一步，
先按方案 A 做，以后如果要扩展成拓扑图再迁移。

**结论：先用方案 A 落地，不引入新依赖；如果已经有 ECharts 基建想统一图表风格，用方案 B 也可以。**

### 5.3 交互设计

- **默认展开到 event 落库层**（4 层全展开），因为这正是排查问题时最想一眼看到的粒度；`tar_parse_failed`
  这类失败节点默认展开显示 `failure_reasons`（因为数量通常不大，展开成本低）。
- **失败节点视觉强调**：红色/警示色 + 数字前加 ⚠️，`rate` 超过某个阈值（比如 1%）时额外加粗，帮助
  一眼定位"链路在哪一步开始出问题"。
- **节点点击钻取**：叶子节点（`event_land_convert_failed` 等）点击后跳转/侧滑打开
  `/event-list?date=...&type=convert_failed&module_name=...`（已有接口，见 API_SPEC 3.8），复用现有
  分页明细表组件，不用为树形视图单独做一套明细 UI。
- **顶部保留日期选择器 + project/module_name 筛选器**，改变筛选条件时整棵树重新请求（一次请求换一棵
  新树，不是逐层请求）。
- **单包下钻入口**：在已有的 `/md5` 差异排行表格每一行加一个"查看链路"按钮，带上该行的 `md5` 跳转到
  树形视图并自动带上 `&md5=xxx`，这样从"哪个包有问题"到"这个包完整走了哪条路"是一条连续的排查路径。

### 5.4 与现有看板的关系

树形视图是**现有对账看板的一个新 Tab**，不是替代品。总览的四个大数字（匹配率/漏消费等）、时间趋势
折线图、模块/项目维度表格仍然是"今天整体健康吗"的第一入口；树形视图是"某一天/某个包，问题具体卡在
哪一环"的下钻工具，两者共享同一批底层 SQL 语义（`send_status`/`status`/`err_detail` 的枚举定义完全
一致），只是聚合形状不同（扁平 vs 嵌套）。

---

## 六、相关文档

- [RECONCILIATION_API_SPEC.md](RECONCILIATION_API_SPEC.md) — 三表结构、已有扁平接口、状态矩阵与错误码枚举，本文档的字段/SQL 均以此为准
- [RECONCILIATION_DASHBOARD_DESIGN.md](RECONCILIATION_DASHBOARD_DESIGN.md) — 原始看板设计（总览/趋势/模块/项目维度）
- [RECONCILIATION_ISSUES_SUMMARY.md](RECONCILIATION_ISSUES_SUMMARY.md) — 对账问题排查历史，含 Kafka retention、Doris 分区、DLQ 熔断等真实案例
- [PIPELINE_FLOW.md](PIPELINE_FLOW.md) — Kafka→Doris 代码调用链（ETL 服务内部实现，非对账接口）
