# FIS 对账看板 API 接口文档

> 供其他服务实现对账看板后端接口时参考。只读查询 Doris 三张对账表，本文档在
> `RECONCILIATION_DASHBOARD_DESIGN.md` 的基础上修正了 JOIN 条件（补充 `module_name`），
> 并给出实际表名/字段/连接信息，可直接对照实现。

## 对账范围与前提

本批接口做**三张对账表的全链路、全状态对账**，不直接对比业务表本身。数据分三层，
每层都有自己的成败状态，对账就是把三层串起来看每一层丢了什么、失败在哪：

```
L1  tar/bag 级   fis_decode_record     每个 md5 一行        解码成败（整包）
                       │ 解出 N 个 event
                       ▼
L2  event 级     fis_decode_detail     每个 event 一行      解析发送成败（上游）
                   (dt+md5+uuid+module)
                       │ 发往 kafka，下游消费
                       ▼
L3  event 级     fis_consume_record    每个 event 一行      落库成败（下游，埋点）
                   (dt+md5+uuid+module)
```

### 三层各自的字段与状态

**L1 `fis_decode_record`（bag 级，UNIQUE KEY `dt,md5`）**
- `status`：0=pending / 1=success / 2=failed / 3=partial（部分 event 失败）
- `stage`（失败阶段）：0=success / 1=下载解包失败 / 2=内容为空 / 3=模块转换失败 / 4=无结果 / 5=发送下游失败
- `error_msg` 格式 `错误码:动态信息`（错误码同 stage）；`parsed_line_count`/`skip_line_count`/`status_line_count`

**L2 `fis_decode_detail`（event 级，UNIQUE KEY `dt,md5,uuid,module_name`）**
- `send_status`：0=pending / 1=success（解析成功且已发往下游 kafka）/ 2=failed（发送下游失败）
- `err_detail`（send_status=2 时）：格式 `5:<原因>`（错误码 5=发送下游失败，Kafka 抖动）

**L3 `fis_consume_record`（event 级，UNIQUE KEY `dt,md5,uuid,module_name`）—— 落库埋点**
- `status`：0=pending / 1=success（trigger 业务表真正落库成功）/ 2=failed
- `err_detail`（status=2 时）错误码区分两种失败：
  - `3:<原因>` = **转换失败**：下游把消息转成 trigger 结构体时失败（`trigger==nil`），根本没进写入队列，永远不会落库。由 event 循环里的 `recordConsume(false)` 同步写入。
  - `6:<原因>` = **落库失败**：转换成功、进了 `AsyncTableWriter`，但批量写 Doris 时永久失败（进 DLQ / 熔断 OPEN 下 DLQ 成功 / 停机丢弃），数据没落进业务表。由 `OnBatchFailure` 回调写入，`<原因>` 是截断后的 Doris 错误信息。

### 核心：event 级对账矩阵（L2 ↔ L3，按 `dt+md5+uuid+module_name` 四段 JOIN）

对一条 `fis_decode_detail.send_status=1`（应落库）的 event，下游 `fis_consume_record` 可能有 4 种结果：

| L2 send_status | L3 consume | 含义 | 看板归类 |
|---|---|---|---|
| 1 | status=1 | 解析成功 → 真正落库 | ✅ matched |
| 1 | status=2, err_detail `3:` | 解析成功 → 下游转换失败 | ⚠️ convert_failed |
| 1 | status=2, err_detail `6:` | 解析成功 → 落库失败（DLQ/丢弃） | ⚠️ landing_failed |
| 1 | 无 consume 行 | 解析成功且发送成功，但 consume 还没有对应行 | ❓ missing（待落库或丢库） |

另外三类 event：
- `fis_decode_detail.send_status=2`：解析成功但发送下游 kafka 失败，根本没到 L3 → **send_failed**（上游发送问题，consume 不应有行）。
- `fis_decode_detail.send_status=3`：trigger 消息里未解析出任何可对账 event（events 数组为空或无 reconcile_id）→ **parse_failed**（解析层未产生 event，不存在"应落库"的概念，consume 不应有行，**不算漏落库**）。
- `fis_consume_record` 有行但 `fis_decode_detail` 无对应行：**extra**（落库了但上游无解析记录，数据来源存疑）。

> `send_status=2`（发送失败）与 `send_status=3`（解析失败）是两个独立信号：前者是"解析出了 event，但发往 kafka 时失败"；后者是"trigger 消息里根本没解析出任何 event"，没有对账 event 产生，不存在应不应落库的问题，因此 parse_failed **不属于** missing（漏落库）。

> **missing 的准确定义**：`fis_decode_detail.send_status=1`（解析成功且发送成功，即"应落库"）但 `fis_consume_record` 还没有对应行。由于 consume 表是异步写入，从 detail 写入到 consume 写入存在消费时延，所以 missing 里包含"正在落库中（时延导致尚未消费到）"和"真的丢库"两种情况。

> ⚠️ 注意：旧的 `matched = COUNT(c.uuid)` 写法会把 `convert_failed`/`landing_failed`（consume 有行但 status=2）误算成 matched。**正确写法是 `matched = SUM(CASE WHEN c.status=1 THEN 1 ELSE 0 END)`**，下方所有 SQL 已按此修正。`missing` 仍用 `c.uuid IS NULL`（consume 完全没行），且限定 `WHERE d.send_status=1`（只算真正应落库的），failed 的 consume 行不算 missing，parse_failed（send_status=3）也不算 missing。

### 实现状态（落库埋点改造已完成）

> ✅ `AsyncTableWriter` 新增可选的 `OnBatchSuccess` / `OnBatchFailure` 回调
> （`internal/biz/table_writer.go`），在一批 records 写 Doris 成功（`err==nil`）或
> 确定无法落库（永久错误进 DLQ、熔断 OPEN 下 DLQ 成功、停机丢弃）时分别触发。三个
> trigger writer（fff/fdr/fcl）的泛型类型从 `orm.FFFTrigger` 等改为 biz 层包装体
> `FFFTriggerWithMeta` 等（`internal/biz/trigger_wrapper.go`），携带
> `md5/module_name/project/dt` 一路到落库回调。成功回填 `consume(status=1)`，
> 失败回填 `consume(status=2, err_detail="6:<原因>")`；转换失败仍由 event 循环的
> `recordConsume(false)` 写 `consume(status=2, err_detail="3:...")`。业务表写入逻辑
> （Stream Load/重试/DLQ/熔断）、orm、repo 签名均未改动。
>
> ⚠️ **dev 测试态**：当前埋点回调未按 env 门控。dev 下业务表是 noOp（`noOpInsert`
> 恒返回 nil = "默认成功"但未真落库），回调会跟着默认成功触发、写出 success consume 行
> （非真实落库），仅用于 dev 跑通对账链路。prd 无 noOp，回调在真实落库时触发，语义正确。
> 代码里 `noOpInsert` 和注册处均有 `TODO(测试态)` 标注，上 prd 前需确认是否恢复
> "仅 env != dev 注册"。

---

## 一、数据源

### 1.1 连接信息（dev 环境）

| 项 | 值 |
|---|---|
| 协议 | MySQL 协议（Doris FE Query Port） |
| Host | `10.18.77.243`（另有 `10.18.65.11`、`10.18.72.52` 可做负载均衡） |
| Port | `9030` |
| Database | `fdi_dev` |
| 账号 | 建议单独申请只读账号，仅 `SELECT` 权限，限 `fdi_dev` 库 |

生产环境地址、账号请向 FDI 团队单独申请，不要复用写入账号。

### 1.2 三张表（实际线上结构，含 module_name 联合唯一键）

> 注意：与设计文档里的初版 DDL 不同，三张表的 `UNIQUE KEY` 都已改为包含
> `module_name`，原因是同一个 `uuid` 在极少数情况下会出现在不同 `module_name` 下
> （不同模块各自生成的行恰好撞了 uuid）。**所有跨表 JOIN 必须带上 `module_name`，
> 只用 `dt+md5+uuid` 三段 JOIN 会把这类行错误地关联到一起，产生假的"匹配"或假的"不一致"。**

```sql
-- fdi_dev.fis_decode_record  解码主表，每个 md5 一行
UNIQUE KEY(dt, md5)
-- 字段：dt, md5, vin, fdi_type, project, status, stage, error_msg,
--      status_line_count, skip_line_count, parsed_line_count, created_at, updated_at
-- status: 0=pending 1=success 2=failed 3=partial
-- stage:  0=success 1=下载解包失败 2=内容为空 3=模块转换失败 4=无结果 5=发送下游失败

-- fdi_dev.fis_decode_detail  解码明细表，每条 event 一行
UNIQUE KEY(dt, md5, uuid, module_name)
-- 字段：dt, md5, uuid, module_name, project, send_status, err_detail, created_at, updated_at
-- send_status: 0=pending 1=success 2=failed

-- fdi_dev.fis_consume_record  下游消费记录表（落库埋点），每条 event 一行
UNIQUE KEY(dt, md5, uuid, module_name)
-- 字段：dt, md5, uuid, module_name, project, status, err_detail, created_at, updated_at
-- status: 0=pending 1=success(落库成功) 2=failed
-- err_detail (status=2): 错误码:动态信息
--   3:xxx = 转换失败（trigger 结构体转换失败，未进写入队列，永不落库）
--   6:xxx = 落库失败（转换成功但 Doris 写入永久失败，进 DLQ / 停机丢弃）
```

`module_name` 取值：`filter_status` / `fdr_status` / `fcl_status` / `fff_close`。

三张表均按 `dt`（DATE）做动态分区，单天查询走分区裁剪很快；范围查询建议限制在 30 天以内。

---

## 二、接口列表

统一前缀：`/dashboard/v1/reconcile/`，全部 `GET`，只读。

| 接口 | 说明 | 参数 |
|---|---|---|
| `overview` | 当日全链路健康度（三层状态：bag解码/event解析/event落库） | `date`（默认今天，格式 `YYYY-MM-DD`） |
| `trend` | 时间趋势 | `start_dt`, `end_dt`（闭区间，跨度 ≤30 天） |
| `module` | 模块维度 | `date`，`project`（可选，交叉钻取到项目内模块分布） |
| `project` | 项目维度 | `date`，`order_by`（`missing`\|`expected`，默认 `missing`），`limit`（默认 20，上限 200） |
| `decode_status` | 上游解码状态分布 | `date` |
| `md5` | 差异 md5 列表 | `date`，`type`（`missing`\|`convert_failed`\|`landing_failed`\|`decode_failed`），`limit`（默认 20，上限 200） |
| `md5_detail` | 单 md5 的 event 级明细 | `md5`（query，必填），`date` |
| `event_list` | event/uuid 级明细列表，不依赖先选 md5 | `date`，`module_name`（可选），`project`（可选），`type`（`missing`\|`extra`\|`send_failed`\|`convert_failed`\|`landing_failed`\|`mismatched`），`page`（默认1），`page_size`（默认20，上限200） |
| `record_consistency` | record↔detail 内部一致性（解码阶段是否丢行） | `date`，`limit`（默认20，上限200） |
| `uuid_source` | uuid 来源细分（real / gen_fallback）匹配率 | `date` |
| `failure_summary` | 失败汇总（四类失败：解码/发送/转换/落库，按 stage/err_detail 聚合） | `date` |

### 通用规则

- `date` 缺省为当天；范围类参数（`start_dt`/`end_dt`）缺省为最近 7 天。
- 非法参数（如 `type` 不在枚举内、`start_dt`~`end_dt` 超过 30 天、`limit`/`page_size` 超过上限）应返回业务错误码，**不要**用 HTTP 5xx。响应是 `BaseResponse` 内嵌的扁平结构：HTTP 200 + `{"code": 0, "message": "ok", ...其余字段与 code/message 同级...}`，`code != 0` 表示业务错误，具体字段见下方各接口示例。
- 所有涉及 detail↔consume 对账的查询，JOIN 条件统一为 `dt + md5 + uuid + module_name` 四段（下方 SQL 已修正）。

---

## 三、返回结构与 SQL（已修正 JOIN 条件）

### 3.1 `GET /dashboard/v1/reconcile/overview?date=2026-07-08`

当日全链路健康度，三层状态一次看全。

```json
{
  "date": "2026-07-08",
  "bag": {
    "total": 5000,
    "decode_success": 4980,
    "decode_failed": 15,
    "decode_partial": 5,
    "decode_success_rate": 0.9960
  },
  "event_parse": {
    "expected": 12329,
    "send_failed": 7,
    "send_failed_rate": 0.0006
  },
  "event_land": {
    "matched": 12320,
    "convert_failed": 3,
    "landing_failed": 2,
    "missing": 4,
    "match_rate": 0.9993
  },
  "extra_consume": 0
}
```

字段含义（对应「对账范围与前提」的状态矩阵）：
- `bag.*`：L1 bag 级，来自 `fis_decode_record`。`decode_success_rate = decode_success / total`。
- `event_parse.expected`：L2 解析成功应落库的 event 数（`send_status=1`）；`send_failed`：发送下游失败（`send_status=2`）。
- `event_land.*`：L3 落库结果（对 expected 的 event）：
  - `matched`：consume `status=1`（真正落库成功）
  - `convert_failed`：consume `status=2` 且 `err_detail LIKE '3:%'`（下游转换失败）
  - `landing_failed`：consume `status=2` 且 `err_detail LIKE '6:%'`（落库失败 DLQ/丢弃）
  - `missing`：detail `send_status=1` 但 consume 完全无行（待落库/丢库）
  - `match_rate = matched / expected`
- `extra_consume`：consume 有行但 detail 无对应解析记录（反向 JOIN，单独查）。

```sql
-- L1 bag 级
SELECT
  COUNT(*) AS total,
  SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) AS decode_success,
  SUM(CASE WHEN status=2 THEN 1 ELSE 0 END) AS decode_failed,
  SUM(CASE WHEN status=3 THEN 1 ELSE 0 END) AS decode_partial
FROM fdi_dev.fis_decode_record
WHERE dt = '2026-07-08';

-- L2 + L3 event 级（detail send_status=1 LEFT JOIN consume）
SELECT
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed,
  SUM(CASE WHEN d.send_status=1 AND c.status=1 THEN 1 ELSE 0 END) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08';

-- extra_consume（反向 JOIN，单独查避免大表全量扫描）
SELECT COUNT(*) AS extra_consume
FROM fdi_dev.fis_consume_record c
LEFT JOIN fdi_dev.fis_decode_detail d
  ON c.dt = d.dt AND c.md5 = d.md5 AND c.uuid = d.uuid AND c.module_name = d.module_name
WHERE c.dt = '2026-07-08' AND d.uuid IS NULL;
```

### 3.2 `GET /dashboard/v1/reconcile/trend?start_dt=2026-07-01&end_dt=2026-07-08`

```json
{
  "start": "2026-07-01",
  "end": "2026-07-08",
  "points": [
    {"dt": "2026-07-01", "expected": 12000, "matched": 12000, "convert_failed": 0, "landing_failed": 0, "missing": 0, "match_rate": 1.0}
  ]
}
```

```sql
SELECT
  d.dt,
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  SUM(CASE WHEN d.send_status=1 AND c.status=1 THEN 1 ELSE 0 END) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt BETWEEN '2026-07-01' AND '2026-07-08'
GROUP BY d.dt
ORDER BY d.dt;
```

### 3.3 `GET /dashboard/v1/reconcile/module?date=2026-07-08[&project=proj-a]`

```json
{
  "date": "2026-07-08",
  "project": "proj-a",
  "modules": [
    {"module_name": "filter_status", "expected": 5000, "matched": 4995, "convert_failed": 1, "landing_failed": 2, "missing": 2, "send_failed": 0, "match_rate": 0.9990}
  ]
}
```

```sql
SELECT
  d.module_name,
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  SUM(CASE WHEN d.send_status=1 AND c.status=1 THEN 1 ELSE 0 END) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing,
  SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08'
  AND d.project = 'proj-a'   -- 按参数条件性拼接：传了 project 才加这一行，交叉钻取到项目内模块分布
GROUP BY d.module_name
ORDER BY missing DESC;
```

### 3.4 `GET /dashboard/v1/reconcile/project?date=2026-07-08&order_by=missing&limit=20`

```json
{
  "date": "2026-07-08",
  "projects": [
    {"project": "proj-a", "expected": 8000, "matched": 7950, "convert_failed": 1, "landing_failed": 2, "missing": 47, "match_rate": 0.9938}
  ]
}
```

```sql
SELECT
  d.project,
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  SUM(CASE WHEN d.send_status=1 AND c.status=1 THEN 1 ELSE 0 END) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
  SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08'
GROUP BY d.project
ORDER BY missing DESC   -- order_by=expected 时改为 ORDER BY expected DESC，白名单映射，不要拼接用户输入
LIMIT 20;
```

`order_by` 只接受 `missing`/`expected` 两个取值，实现时用白名单映射成固定 SQL 片段，禁止把参数原文拼进 `ORDER BY`。

### 3.5 `GET /dashboard/v1/reconcile/decode_status?date=2026-07-08`

```json
{
  "date": "2026-07-08",
  "items": [
    {"status": 1, "stage": 0, "count": 11800},
    {"status": 2, "stage": 1, "count": 5}
  ]
}
```

```sql
SELECT status, stage, COUNT(*) AS cnt
FROM fdi_dev.fis_decode_record
WHERE dt = '2026-07-08'
GROUP BY status, stage
ORDER BY status, stage;
```

### 3.6 `GET /dashboard/v1/reconcile/md5?date=2026-07-08&type=missing&limit=20`

按 md5 聚合差异，定位问题 bag。`type` 取值：
- `missing`：漏落库最多的 md5（detail `send_status=1` 但 consume 无行）
- `convert_failed`：转换失败最多的 md5（consume `err_detail LIKE '3:%'`）
- `landing_failed`：落库失败最多的 md5（consume `err_detail LIKE '6:%'`）
- `decode_failed`：解码失败的 md5（来自 `fis_decode_record` `status IN (2,3)`）

```json
{
  "date": "2026-07-08",
  "type": "missing",
  "items": [
    {"md5": "abc123", "module_name": "fff_close", "count": 50}
  ]
}
```

```sql
-- type=missing：漏落库最多的 md5
SELECT d.md5,
       ANY_VALUE(d.module_name) AS module_name,
       SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS count
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08'
GROUP BY d.md5
HAVING count > 0
ORDER BY count DESC
LIMIT 20;

-- type=convert_failed：转换失败最多的 md5
SELECT c.md5,
       ANY_VALUE(c.module_name) AS module_name,
       COUNT(*) AS count
FROM fdi_dev.fis_consume_record c
WHERE c.dt = '2026-07-08' AND c.status=2 AND c.err_detail LIKE '3:%'
GROUP BY c.md5
ORDER BY count DESC
LIMIT 20;

-- type=landing_failed：落库失败最多的 md5
SELECT c.md5,
       ANY_VALUE(c.module_name) AS module_name,
       COUNT(*) AS count
FROM fdi_dev.fis_consume_record c
WHERE c.dt = '2026-07-08' AND c.status=2 AND c.err_detail LIKE '6:%'
GROUP BY c.md5
ORDER BY count DESC
LIMIT 20;

-- type=decode_failed：解码失败的 md5（bag 级）
SELECT md5, status, stage, error_msg, parsed_line_count
FROM fdi_dev.fis_decode_record
WHERE dt = '2026-07-08' AND status IN (2, 3)
ORDER BY updated_at DESC
LIMIT 20;
```

### 3.7 `GET /dashboard/v1/reconcile/md5_detail?date=2026-07-08&md5=abc123`

```json
{
  "date": "2026-07-08",
  "md5": "abc123",
  "items": [
    {"uuid": "real-uuid-1", "module_name": "fff_close", "send_status": 1, "err_detail": "",
     "consumed": true, "consume_status": 1, "consume_err": ""}
  ]
}
```

```sql
SELECT
  d.uuid, d.module_name, d.send_status, d.err_detail,
  CASE WHEN c.uuid IS NOT NULL THEN 1 ELSE 0 END AS consumed,
  c.status AS consume_status, c.err_detail AS consume_err
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08' AND d.md5 = 'abc123'
ORDER BY d.module_name, d.uuid;
```

### 3.8 `GET /dashboard/v1/reconcile/event_list?date=2026-07-08&type=missing[&module_name=fff_close][&project=proj-a]&page=1&page_size=20`

event/uuid 级明细列表，不依赖先选 md5。`type` 取值（对应「对账范围与前提」的状态矩阵）：
- `missing`：detail `send_status=1` 但 consume 无行（待落库/丢库，包含消费时延中尚未处理到和真正丢库）
- `extra`：consume 有行但 detail 无对应解析记录（数据来源存疑）
- `send_failed`：detail `send_status=2`（解析成功但发送下游失败，Kafka 抖动）
- `parse_failed`：detail `send_status=3`（trigger 消息未解析出任何可对账 event，不是漏落库）
- `convert_failed`：consume `status=2` 且 `err_detail LIKE '3:%'`（下游转换失败）
- `landing_failed`：consume `status=2` 且 `err_detail LIKE '6:%'`（落库失败 DLQ/丢弃）
- `mismatched`：matched 但 `project` 不一致（内容级差异，见 3.10）

```json
{
  "date": "2026-07-08",
  "type": "missing",
  "total": 3,
  "page": 1,
  "page_size": 20,
  "items": [
    {"md5": "abc123", "uuid": "real-uuid-1", "module_name": "fff_close", "project": "proj-a", "send_status": 1, "err_detail": ""}
  ]
}
```

```sql
-- type=missing：detail 有、consume 没有
SELECT d.dt, d.md5, d.uuid, d.module_name, d.project, d.send_status, d.err_detail
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08' AND d.send_status = 1 AND c.uuid IS NULL
  -- AND d.module_name = 'fff_close'
  -- AND d.project = 'proj-a'
ORDER BY d.md5, d.uuid
LIMIT 20 OFFSET 0;

-- type=extra：consume 有、detail 没有
SELECT c.dt, c.md5, c.uuid, c.module_name, c.project, c.status, c.err_detail
FROM fdi_dev.fis_consume_record c
LEFT JOIN fdi_dev.fis_decode_detail d
  ON c.dt = d.dt AND c.md5 = d.md5 AND c.uuid = d.uuid AND c.module_name = d.module_name
WHERE c.dt = '2026-07-08' AND d.uuid IS NULL
  -- AND c.module_name = 'fff_close'
  -- AND c.project = 'proj-a'
ORDER BY c.md5, c.uuid
LIMIT 20 OFFSET 0;

-- type=send_failed：上游发送下游失败（detail 侧）
SELECT dt, md5, uuid, module_name, project, send_status, err_detail
FROM fdi_dev.fis_decode_detail
WHERE dt = '2026-07-08' AND send_status = 2
  -- AND module_name = 'fff_close'  AND project = 'proj-a'
ORDER BY md5, uuid
LIMIT 20 OFFSET 0;

-- type=convert_failed：下游转换失败（consume 侧，err_detail 3:）
SELECT dt, md5, uuid, module_name, project, status, err_detail
FROM fdi_dev.fis_consume_record
WHERE dt = '2026-07-08' AND status = 2 AND err_detail LIKE '3:%'
  -- AND module_name = 'fff_close'  AND project = 'proj-a'
ORDER BY md5, uuid
LIMIT 20 OFFSET 0;

-- type=landing_failed：落库失败（consume 侧，err_detail 6:）
SELECT dt, md5, uuid, module_name, project, status, err_detail
FROM fdi_dev.fis_consume_record
WHERE dt = '2026-07-08' AND status = 2 AND err_detail LIKE '6:%'
  -- AND module_name = 'fff_close'  AND project = 'proj-a'
ORDER BY md5, uuid
LIMIT 20 OFFSET 0;

-- type=mismatched：见 3.10（内容级差异），同样加分页
```

`total` 用对应分支去掉 `LIMIT/OFFSET` 后套一层 `SELECT COUNT(*) FROM (...) t` 得到。`page`/`page_size` 转换为 `LIMIT page_size OFFSET (page-1)*page_size`。

### 3.9 `GET /dashboard/v1/reconcile/record_consistency?date=2026-07-08&limit=20`

```json
{
  "date": "2026-07-08",
  "items": [
    {"md5": "abc123", "parsed_line_count": 50, "detail_count": 47, "diff": 3}
  ]
}
```

```sql
SELECT r.md5, r.parsed_line_count,
       COUNT(d.uuid) AS detail_count,
       r.parsed_line_count - COUNT(d.uuid) AS diff
FROM fdi_dev.fis_decode_record r
LEFT JOIN fdi_dev.fis_decode_detail d ON r.dt = d.dt AND r.md5 = d.md5
WHERE r.dt = '2026-07-08' AND r.status IN (1, 3)   -- 只看 success/partial，status=2 本就无 detail，不算异常
GROUP BY r.md5, r.parsed_line_count
HAVING diff <> 0
ORDER BY ABS(diff) DESC
LIMIT 20;
```

> 这条 JOIN 只到 `dt+md5`（`fis_decode_record` 主键不含 uuid/module_name），不需要也不能加 module_name。

### 3.10 `GET /dashboard/v1/reconcile/uuid_source?date=2026-07-08`

```json
{
  "date": "2026-07-08",
  "sources": [
    {"uuid_source": "real", "expected": 12329, "matched": 12329, "missing": 0, "match_rate": 1.0},
    {"uuid_source": "gen_fallback", "expected": 0, "matched": 0, "missing": 0, "match_rate": null}
  ]
}
```

```sql
-- matched 已按前述统一修正：用 SUM(c.status=1) 而不是 COUNT(c.uuid)，
-- 否则 consume 侧 status=2（转换失败/落库失败）的行会被误计入 matched
SELECT
  CASE WHEN d.uuid LIKE 'gen:%' THEN 'gen_fallback' ELSE 'real' END AS uuid_source,
  COUNT(*) AS expected,
  SUM(CASE WHEN c.status=1 THEN 1 ELSE 0 END) AS matched,
  SUM(CASE WHEN c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fdi_dev.fis_decode_detail d
LEFT JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08' AND d.send_status = 1
GROUP BY uuid_source;
```

> `gen_fallback` 分支预期恒为 0（上游兜底生成 uuid 的逻辑已下线），一旦非 0 属于回归，需要报警。

内容级差异（`type=mismatched` 复用此查询）：

```sql
SELECT d.dt, d.md5, d.uuid,
       d.module_name AS detail_module, c.module_name AS consume_module,
       d.project AS detail_project, c.project AS consume_project
FROM fdi_dev.fis_decode_detail d
JOIN fdi_dev.fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
WHERE d.dt = '2026-07-08'
  AND (d.project <> c.project)   -- module_name 已经在 JOIN 条件里，两边必然相等，不会再出现在这里
LIMIT 50 OFFSET 0;
```

> 注意：修正 JOIN 后 `module_name` 已经是等值连接条件，两边天然一致，**不会再出现
> `module_name` 不一致的行**——这类"跨模块串号"的脏数据，改成四段 JOIN 后已经被
> 从"内容不一致"变成了"完全找不到匹配"（会体现在 `missing`/`extra` 计数里）。
> 如果业务上仍想探测"同 uuid 不同 module_name 撞车"这种情况本身，需要单独一条
> **不带 module_name** 的 JOIN 查询专门定位这类行，而不是套用上面对账用的四段 JOIN：
> ```sql
> SELECT d.dt, d.md5, d.uuid, d.module_name AS detail_module, c.module_name AS consume_module
> FROM fdi_dev.fis_decode_detail d
> JOIN fdi_dev.fis_consume_record c ON d.dt=c.dt AND d.md5=c.md5 AND d.uuid=c.uuid
> WHERE d.dt = '2026-07-08' AND d.module_name <> c.module_name
> LIMIT 50;
> ```

### 3.11 `GET /dashboard/v1/reconcile/failure_summary?date=2026-07-08`

把四类失败聚合到一张表，一眼看清当天失败的全貌和主导原因，不用分别翻各接口。

```json
{
  "date": "2026-07-08",
  "decode_failed": [
    {"stage": 1, "stage_desc": "下载解包失败", "count": 5, "sample_error_msg": "1:connection timeout"}
  ],
  "send_failed": [
    {"module_name": "fff_close", "err_detail": "5:broker unavailable:192.168.1.1:9092", "count": 3}
  ],
  "convert_failed": [
    {"module_name": "fdr_status", "err_detail": "3:convert failed", "count": 2}
  ],
  "landing_failed": [
    {"module_name": "fff_close", "err_detail": "6:no partition for this tuple...", "count": 4}
  ]
}
```

四类各自的含义（对应三层模型）：
- `decode_failed`：L1 bag 级，来自 `fis_decode_record`，`status IN (2,3)`，按失败阶段 `stage` 聚合，`count` 为失败 md5 数；`stage_desc` 由后端按枚举映射（见下），`sample_error_msg` 取该 stage 下任一条 `error_msg` 作为样例。
- `send_failed`：L2 event 级，来自 `fis_decode_detail`，`send_status=2`（上游发送下游失败，Kafka 抖动），按 `module_name` + `err_detail` 聚合。
- `convert_failed`：L3 event 级，来自 `fis_consume_record`，`status=2` 且 `err_detail LIKE '3:%'`（下游转换失败，`trigger==nil`），按 `module_name` + `err_detail` 聚合。
- `landing_failed`：L3 event 级，来自 `fis_consume_record`，`status=2` 且 `err_detail LIKE '6:%'`（落库失败，Doris 写入永久失败/DLQ/丢弃），按 `module_name` + `err_detail` 聚合。

```sql
-- decode_failed：按 stage 聚合解码失败的 md5 数
SELECT stage,
       COUNT(*) AS count,
       ANY_VALUE(error_msg) AS sample_error_msg
FROM fdi_dev.fis_decode_record
WHERE dt = '2026-07-08' AND status IN (2, 3)
GROUP BY stage
ORDER BY count DESC;

-- send_failed：按 module_name + err_detail 聚合上游发送失败
SELECT module_name, err_detail, COUNT(*) AS count
FROM fdi_dev.fis_decode_detail
WHERE dt = '2026-07-08' AND send_status = 2
GROUP BY module_name, err_detail
ORDER BY count DESC
LIMIT 50;

-- convert_failed：按 module_name + err_detail 聚合下游转换失败（err_detail 3:）
SELECT module_name, err_detail, COUNT(*) AS count
FROM fdi_dev.fis_consume_record
WHERE dt = '2026-07-08' AND status = 2 AND err_detail LIKE '3:%'
GROUP BY module_name, err_detail
ORDER BY count DESC
LIMIT 50;

-- landing_failed：按 module_name + err_detail 聚合落库失败（err_detail 6:）
SELECT module_name, err_detail, COUNT(*) AS count
FROM fdi_dev.fis_consume_record
WHERE dt = '2026-07-08' AND status = 2 AND err_detail LIKE '6:%'
GROUP BY module_name, err_detail
ORDER BY count DESC
LIMIT 50;
```

`stage` → `stage_desc` 枚举映射（后端实现时硬编码，不要让前端自己猜数字）：

| stage | 含义 |
|---|---|
| 0 | success（不会出现在本接口结果里） |
| 1 | 下载解包失败 |
| 2 | 文件内容为空 |
| 3 | 模块转换失败 |
| 4 | 转换后无结果 |
| 5 | 发送下游失败 |

`err_detail` 错误码映射（同样建议后端硬编码 desc，前端不猜数字）：

| 前缀码 | 出现表 | 含义 |
|---|---|---|
| `1:`~`5:` | fis_decode_record.error_msg | 对应 stage（解码阶段失败） |
| `5:` | fis_decode_detail.err_detail | 发送下游失败（Kafka 抖动） |
| `3:` | fis_consume_record.err_detail | 转换失败（trigger 结构体转换失败） |
| `6:` | fis_consume_record.err_detail | 落库失败（Doris 写入永久失败/DLQ/丢弃） |

> `err_detail` 的格式是 `错误码:动态信息`，动态信息里可能带变化的 IP/timeout 等，
> 按 `err_detail` 整串分组会产生较多细行。如果实现方觉得太碎，可改成只取冒号前的
> 错误码部分做分组键（`SUBSTRING_INDEX(err_detail, ':', 1)`），把同类失败合并到一行，
> 再附 `ANY_VALUE(err_detail)` 给一个样例——两种粒度都可接受，按看板展示效果选。
> `landing_failed` 的动态信息尤其碎（每条 Doris 错误不同），强烈建议按错误码+错误类别
> 归并，否则一个"no partition"能产生几十行。

---

## 四、实现建议（给实现方）

1. **只读账号**：Doris 侧建议单独开一个只 `SELECT` 权限的账号，限定 `fdi_dev` 库，不要用写入账号。
2. **缓存**：对账表是分钟级批写入，`overview`/`trend`/`module`/`project` 建议做 1~5 分钟的内存或 Redis 缓存，避免看板高频轮询打满 Doris FE 查询连接。
3. **反向 JOIN 单独查**：`extra_consume`（多消费）用 `fis_consume_record` 反向 LEFT JOIN，数据量大时较慢，务必加 `LIMIT`/单独接口，不要和总览的正向查询合并成一次大查询。
4. **参数防注入**：`order_by`、`type` 等枚举参数一律用白名单映射成固定 SQL 片段，禁止拼接用户输入到 SQL 中；`date`/`start`/`end` 建议用 `time.Parse("2006-01-02", ...)` 校验格式后再拼 SQL。
5. **分区裁剪**：所有查询都必须带 `dt =` 或 `dt BETWEEN` 条件，不要走全表扫描；范围查询限制在 30 天以内。
