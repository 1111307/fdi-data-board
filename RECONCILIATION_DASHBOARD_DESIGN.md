# FIS 对账监控看板设计

> 在独立服务上实现只读查询接口，对接 Doris 三张对账表，提供对账监控看板。本文档含表结构、接口设计、查询 SQL、前端看板建议。

---

## 一、三张对账表结构

### 1. fis_decode_record（解码主表，每 md5 一行）

```sql
CREATE TABLE fis_decode_record (
    dt                  DATE            NOT NULL   COMMENT '日期分区键，取上游消息的 dt 字段(非 time.Now())',
    md5                 VARCHAR(64)     NOT NULL   COMMENT 'bag文件唯一标识，取上游消息的 md5 字段',
    vin                 VARCHAR(64)     NOT NULL DEFAULT ""  COMMENT '车辆标识（脱敏后）',
    fdi_type            VARCHAR(64)     NOT NULL DEFAULT ""  COMMENT 'FDI类型',
    project             VARCHAR(128)    NOT NULL DEFAULT ""  COMMENT '项目山头',
    status              TINYINT         NOT NULL DEFAULT "0" COMMENT '解码状态 0:pending 1:success 2:failed 3:partial',
    stage               TINYINT         NOT NULL DEFAULT "0" COMMENT '失败阶段 0:success 1:下载解包失败 2:内容为空 3:模块转换失败 4:无结果 5:发送下游失败',
    error_msg           VARCHAR(256)    NOT NULL DEFAULT ""  COMMENT '失败原因 格式:错误码:动态信息',
    status_line_count   INT             NOT NULL DEFAULT "0" COMMENT '原始状态行数',
    skip_line_count     INT             NOT NULL DEFAULT "0" COMMENT 'json解析失败跳过的行数',
    parsed_line_count   INT             NOT NULL DEFAULT "0" COMMENT '实际参与转换的行数',
    created_at          DATETIME        NOT NULL   COMMENT '创建时间',
    updated_at          DATETIME        NOT NULL   COMMENT '最后更新时间'
)
UNIQUE KEY(dt, md5)
PARTITION BY RANGE(dt) ();
```

### 2. fis_decode_detail（解码明细表，每 event 一行）

```sql
CREATE TABLE fis_decode_detail (
    dt            DATE            NOT NULL   COMMENT '日期分区键，取上游消息的 dt 字段',
    md5           VARCHAR(64)     NOT NULL   COMMENT 'bag文件唯一标识，取上游消息的 md5 字段，关联主表',
    uuid          VARCHAR(128)    NOT NULL DEFAULT ""  COMMENT '行唯一标识，取上游消息的 uuid 字段',
    project       VARCHAR(128)    NOT NULL DEFAULT ""  COMMENT '项目山头',
    module_name   VARCHAR(32)     NOT NULL DEFAULT ""  COMMENT '模块 fdr_status/fcl_status/filter_status/fff_close',
    send_status   TINYINT         NOT NULL DEFAULT "0" COMMENT '发送状态 0:pending 1:success 2:failed',
    err_detail    VARCHAR(256)    NOT NULL DEFAULT ""  COMMENT '发送失败原因 格式:错误码:动态信息',
    created_at    DATETIME        NOT NULL   COMMENT '创建时间',
    updated_at    DATETIME        NOT NULL   COMMENT '最后更新时间'
)
UNIQUE KEY(dt, md5, uuid)
PARTITION BY RANGE(dt) ();
```

### 3. fis_consume_record（消费记录表，每 event 一行）

```sql
CREATE TABLE fis_consume_record (
    dt            DATE            NOT NULL   COMMENT '日期分区键，取上游 fis_channel 消息的 dt 字段',
    md5           VARCHAR(64)     NOT NULL   COMMENT 'bag文件唯一标识，关联 fis_decode_detail.md5',
    uuid          VARCHAR(128)    NOT NULL DEFAULT ""  COMMENT '行唯一标识，与 fis_decode_detail.uuid 对账',
    project       VARCHAR(128)    NOT NULL DEFAULT ""  COMMENT '项目山头',
    module_name   VARCHAR(32)     NOT NULL DEFAULT ""  COMMENT '模块 fdr_status/fcl_status/filter_status/fff_close',
    status        TINYINT         NOT NULL DEFAULT "0" COMMENT '消费状态 0:pending 1:success 2:failed',
    err_detail    VARCHAR(256)    NOT NULL DEFAULT ""  COMMENT '失败原因 格式:错误码:动态信息',
    created_at    DATETIME        NOT NULL   COMMENT '创建时间',
    updated_at    DATETIME        NOT NULL   COMMENT '最后更新时间'
)
UNIQUE KEY(dt, md5, uuid)
PARTITION BY RANGE(dt) ();
```

---

## 二、Go ORM 结构体

```go
package orm

import "time"

// FisDecodeRecord 对应 fis_decode_record 表，每个 md5 一行
type FisDecodeRecord struct {
    Dt              string    `json:"dt" db:"dt"`               // 日期分区键，取上游消息的 dt 字段
    Md5             string    `json:"md5" db:"md5"`             // bag 文件唯一标识
    Vin             string    `json:"vin" db:"vin"`             // 车辆标识（脱敏后）
    FdiType         string    `json:"fdi_type" db:"fdi_type"`   // FDI 类型
    Project         string    `json:"project" db:"project"`     // 项目山头
    Status          int8      `json:"status" db:"status"`       // 0:pending 1:success 2:failed 3:partial
    Stage           int8      `json:"stage" db:"stage"`         // 失败阶段错误码
    ErrorMsg        string    `json:"error_msg" db:"error_msg"` // 失败原因
    StatusLineCount int       `json:"status_line_count" db:"status_line_count"` // 原始状态行数
    SkipLineCount   int       `json:"skip_line_count" db:"skip_line_count"`     // 跳过行数
    ParsedLineCount int       `json:"parsed_line_count" db:"parsed_line_count"` // 实际参与转换行数
    CreatedAt       time.Time `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// FisDecodeDetail 对应 fis_decode_detail 表，每条 event 一行
type FisDecodeDetail struct {
    Dt         string    `json:"dt" db:"dt"`                   // 日期分区键
    Md5        string    `json:"md5" db:"md5"`                 // bag 文件唯一标识，关联主表
    UUID       string    `json:"uuid" db:"uuid"`               // 行唯一标识
    Project    string    `json:"project" db:"project"`         // 项目山头
    ModuleName string    `json:"module_name" db:"module_name"` // fdr_status / fcl_status / filter_status / fff_close
    SendStatus int8      `json:"send_status" db:"send_status"` // 0:pending 1:success 2:failed
    ErrDetail  string    `json:"err_detail" db:"err_detail"`   // 发送失败原因
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// FisConsumeRecord 对应 fis_consume_record 表，下游消费每条 event 一行
type FisConsumeRecord struct {
    Dt         string    `json:"dt" db:"dt"`                   // 日期分区键，与 fis_decode_detail.dt 同源
    Md5        string    `json:"md5" db:"md5"`                 // bag 文件唯一标识，关联 fis_decode_detail.md5
    UUID       string    `json:"uuid" db:"uuid"`               // 行唯一标识，与 fis_decode_detail.uuid 对账
    Project    string    `json:"project" db:"project"`         // 项目山头
    ModuleName string    `json:"module_name" db:"module_name"` // fdr_status / fcl_status / filter_status / fff_close
    Status     int8      `json:"status" db:"status"`           // 0:pending 1:success 2:failed
    ErrDetail  string    `json:"err_detail" db:"err_detail"`   // 失败原因 格式:错误码:动态信息
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
```

### 枚举值

| 字段 | 值 | 含义 |
|---|---|---|
| fis_decode_record.status | 0/1/2/3 | pending/success/failed/partial |
| fis_decode_record.stage | 0/1/2/3/4/5 | success/下载解包失败/内容为空/转换失败/无结果/发送下游失败 |
| fis_decode_detail.send_status | 0/1/2 | pending/success/failed |
| fis_consume_record.status | 0/1/2 | pending/success/failed |
| module_name | - | fdr_status / fcl_status / filter_status / fff_close |

---

## 三、看板指标设计

### 总览（当日健康度）
回答"今天对账正常吗？"

| 指标 | 含义 |
|---|---|
| 期望收到 (expected) | detail send_status=1 计数 |
| 实际收到 (consumed) | consume_record 计数 |
| 匹配数 (matched) | 两表 JOIN 命中 |
| 匹配率 (match_rate) | matched / expected |
| 漏消费 (missing_consume) | detail 有 consume 没（下游问题） |
| 多消费 (extra_consume) | consume 有 detail 没（上游丢 decode_log） |
| 上游发送失败 (send_failed) | detail send_status=2（Kafka 抖动指标） |
| 解码失败 md5 数 | record status=2/3（上游解码问题） |
| 解码成功率 | record status=1 / 总数 |

### 时间趋势
按 `dt` 分组，看匹配率随时间变化。**曲线突降 = 出问题时刻**，对应上游发版/关停/Kafka 抖动。

- 期望/实际/匹配 三条线（折线图）
- 漏消费/多消费 差异线（柱状图）
- 解码成功率趋势

### 模块维度
按 `module_name` 分组，定位哪个模块对不上。差异通常集中在某模块（如 fff_close）。

| module_name | expected | matched | missing | match_rate |
|---|---|---|---|---|
| filter_status | ... | ... | ... | ... |
| fdr_status | ... | ... | ... | ... |
| fcl_status | ... | ... | ... | ... |
| fff_close | ... | ... | ... | ... |

### 项目维度
按 `project`（项目山头）分组，定位是否有某个项目整体对不上（例如某项目车辆用了旧版本上报协议、某项目单独接入了新 module）。项目数可能比 module 多，默认按 missing 降序 + limit，不做全量展示。

| project | expected | matched | missing | match_rate |
|---|---|---|---|---|
| proj-a | ... | ... | ... | ... |
| proj-b | ... | ... | ... | ... |

支持 `project + module_name` 二级交叉钻取（选中一个 project 后再看该项目内各 module 的匹配率），复用模块维度 SQL 加 `AND d.project = ?` 即可，不必单独开接口。

### md5 明细
差异最大的 md5 列表，点进去看 event 级明细，直接定位问题 bag。

- 差异排行：按 missing 数降序
- 上游解码失败 md5：record status=2/3
- 单 md5 详情：该 bag 下所有 event 的 detail vs consume 逐条对比

### 细粒度对账（event/uuid 级）

上面三层（总览/趋势/模块）都是聚合计数，md5 明细也要先选中一个 md5 才能钻取。以下补细粒度对账能力，不依赖先选 md5：

**1. event 明细列表（独立视图，可全天扫描）**

不先选 md5，直接按 `dt + module_name + 差异类型` 过滤，分页看 uuid 级明细：
- `missing`：detail 有、consume 没有（下游没收到）
- `extra`：consume 有、detail 没有（上游发的 channel 消息没对应 decode_log 记录）
- `failed`：send_status=2 或 consume status=2（明确失败，非缺失）
- `matched`：JOIN 命中但字段不一致（见第 4 点）

**2. record ↔ detail 内部一致性**

detail↔consume_record 对的是"上游发出 vs 下游收到"，但上游解码阶段自己就可能丢行——`fis_decode_record.parsed_line_count`（该 md5 声明的应写入行数）与 `fis_decode_detail` 里该 md5 实际写入的行数不一致，说明问题出在解码阶段，跟下游无关，且不会被 detail↔consume 对账发现（因为两边都是从同一份缺行的 detail 出发）。

**3. uuid 来源细分**

按 uuid 是否为 `gen:` 前缀的兜底生成值分组统计匹配率。历史修复已去掉 `buildFisDecodeDetails` 里的兜底生成逻辑（events 为空时不再生成 `gen:uuid`），此处保留作为**回归检测**：如果某天 `gen_fallback` 类别的数量从 0 变为非 0，说明兜底逻辑被意外重新触发。

**4. event 内容级差异**

目前 detail↔consume_record 的 JOIN 只判断 uuid 是否存在（"有没有"），没检查同一 uuid 两边的 `module_name`、`project` 等字段是否一致（"对不对"）。字段不一致意味着同一事件在上下游被分类/归属到了不同模块或项目，属于隐蔽的数据错乱，不会体现在 missing/extra 计数里。

---

## 四、接口设计（REST）

所有接口只读，连 Doris 9030（MySQL 协议），给只读账号。

| 接口 | 说明 | 关键参数 |
|---|---|---|
| `GET /api/reconcile/overview` | 总览（当日健康度） | date（默认今天） |
| `GET /api/reconcile/trend` | 时间趋势 | start, end（日期范围，≤30天） |
| `GET /api/reconcile/module` | 模块维度 | date, project(可选，交叉钻取) |
| `GET /api/reconcile/project` | 项目维度 | date, order_by(missing/expected,默认missing), limit |
| `GET /api/reconcile/decode-status` | 上游解码状态分布 | date |
| `GET /api/reconcile/md5` | 差异 md5 列表 | date, type(missing/extra/failed), limit |
| `GET /api/reconcile/md5/{md5}` | 单 md5 event 级明细 | md5, date |
| `GET /api/reconcile/event-list` | event/uuid 级明细列表（不依赖先选 md5） | date, module_name, project, type(missing/extra/failed/mismatched), page, page_size |
| `GET /api/reconcile/record-consistency` | record↔detail 内部一致性（解码阶段是否丢行） | date, limit |
| `GET /api/reconcile/uuid-source` | uuid 来源细分（real / gen_fallback）匹配率 | date |

### 返回示例

```json
// GET /api/reconcile/overview?date=2026-07-08
{
  "date": "2026-07-08",
  "expected": 12329,
  "consumed": 12329,
  "matched": 12329,
  "match_rate": 1.0000,
  "missing_consume": 0,
  "extra_consume": 0,
  "send_failed": 0,
  "decode_failed_md5": 0,
  "decode_success_rate": 1.0000
}
```

```json
// GET /api/reconcile/module?date=2026-07-08
{
  "date": "2026-07-08",
  "modules": [
    {"module_name": "filter_status", "expected": 5000, "matched": 5000, "missing": 0, "match_rate": 1.0},
    {"module_name": "fdr_status", "expected": 4000, "matched": 4000, "missing": 0, "match_rate": 1.0}
  ]
}
```

```json
// GET /api/reconcile/project?date=2026-07-08&limit=20
{
  "date": "2026-07-08",
  "projects": [
    {"project": "proj-a", "expected": 8000, "matched": 7950, "missing": 50, "match_rate": 0.9938},
    {"project": "proj-b", "expected": 4329, "matched": 4329, "missing": 0, "match_rate": 1.0}
  ]
}
```

```json
// GET /api/reconcile/module?date=2026-07-08&project=proj-a  （交叉钻取：单项目内的模块分布）
{
  "date": "2026-07-08",
  "project": "proj-a",
  "modules": [
    {"module_name": "fff_close", "expected": 2000, "matched": 1950, "missing": 50, "match_rate": 0.975}
  ]
}
```

```json
// GET /api/reconcile/event-list?date=2026-07-08&type=missing&module_name=fff_close&page=1&page_size=20
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

```json
// GET /api/reconcile/record-consistency?date=2026-07-08
{
  "date": "2026-07-08",
  "items": [
    {"md5": "abc123", "parsed_line_count": 50, "detail_count": 47, "diff": 3}
  ]
}
```

```json
// GET /api/reconcile/uuid-source?date=2026-07-08
{
  "date": "2026-07-08",
  "sources": [
    {"uuid_source": "real", "expected": 12329, "matched": 12329, "missing": 0, "match_rate": 1.0},
    {"uuid_source": "gen_fallback", "expected": 0, "matched": 0, "missing": 0, "match_rate": null}
  ]
}
```

---

## 五、查询 SQL

### 1. 总览
```sql
SELECT
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed,
  COUNT(c.uuid) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing_consume
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid
WHERE d.dt = '2026-07-08';

-- 多消费（反向 JOIN，单独查避免全表扫描）
SELECT COUNT(*) AS extra_consume
FROM fis_consume_record c
LEFT JOIN fis_decode_detail d
  ON c.dt = d.dt AND c.md5 = d.md5 AND c.uuid = d.uuid
WHERE c.dt = '2026-07-08' AND d.uuid IS NULL;

-- 解码状态
SELECT
  SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) AS decode_success,
  SUM(CASE WHEN status IN (2,3) THEN 1 ELSE 0 END) AS decode_failed,
  COUNT(*) AS total
FROM fis_decode_record
WHERE dt = '2026-07-08';
```

### 2. 时间趋势
```sql
SELECT
  d.dt,
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  COUNT(c.uuid) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid
WHERE d.dt BETWEEN '2026-07-01' AND '2026-07-08'
GROUP BY d.dt
ORDER BY d.dt;
```

### 3. 模块维度
```sql
SELECT
  d.module_name,
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  COUNT(c.uuid) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing,
  SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid
WHERE d.dt = '2026-07-08'
  -- AND d.project = 'proj-a'   -- 传了 project 参数则交叉钻取到该项目内的模块分布
GROUP BY d.module_name
ORDER BY missing DESC;
```

### 3b. 项目维度
```sql
SELECT
  d.project,
  SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
  COUNT(c.uuid) AS matched,
  SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid
WHERE d.dt = '2026-07-08'
GROUP BY d.project
ORDER BY missing DESC
LIMIT 20;
```

### 4. 上游解码状态分布
```sql
SELECT status, stage, COUNT(*) AS cnt
FROM fis_decode_record
WHERE dt = '2026-07-08'
GROUP BY status, stage
ORDER BY status, stage;
```

### 5. 差异 md5 排行
```sql
-- 漏消费最多的 md5
SELECT d.md5,
       ANY_VALUE(d.module_name) AS module_name,
       SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid
WHERE d.dt = '2026-07-08'
GROUP BY d.md5
HAVING missing > 0
ORDER BY missing DESC
LIMIT 20;

-- 解码失败的 md5
SELECT md5, status, stage, error_msg, parsed_line_count
FROM fis_decode_record
WHERE dt = '2026-07-08' AND status IN (2, 3)
ORDER BY updated_at DESC
LIMIT 20;
```

### 6. 单 md5 event 级明细
```sql
SELECT
  d.uuid, d.module_name, d.send_status, d.err_detail,
  CASE WHEN c.uuid IS NOT NULL THEN 1 ELSE 0 END AS consumed,
  c.status AS consume_status, c.err_detail AS consume_err
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c
  ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid
WHERE d.dt = '2026-07-08' AND d.md5 = 'xxxxx'
ORDER BY d.module_name, d.uuid;
```

### 7. event 明细列表（独立视图，不依赖先选 md5）

```sql
-- type=missing：detail 有、consume 没有
SELECT d.dt, d.md5, d.uuid, d.module_name, d.project, d.send_status, d.err_detail
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c ON d.dt=c.dt AND d.md5=c.md5 AND d.uuid=c.uuid
WHERE d.dt = '2026-07-08' AND d.send_status = 1 AND c.uuid IS NULL
  -- AND d.module_name = 'fff_close'
ORDER BY d.md5, d.uuid
LIMIT 20 OFFSET 0;

-- type=extra：consume 有、detail 没有
SELECT c.dt, c.md5, c.uuid, c.module_name, c.project, c.status, c.err_detail
FROM fis_consume_record c
LEFT JOIN fis_decode_detail d ON c.dt=d.dt AND c.md5=d.md5 AND c.uuid=d.uuid
WHERE c.dt = '2026-07-08' AND d.uuid IS NULL
  -- AND c.module_name = 'fff_close'
ORDER BY c.md5, c.uuid
LIMIT 20 OFFSET 0;

-- type=failed：上游发送失败或下游消费失败
SELECT dt, md5, uuid, module_name, project, send_status AS status, err_detail
FROM fis_decode_detail
WHERE dt = '2026-07-08' AND send_status = 2
UNION ALL
SELECT dt, md5, uuid, module_name, project, status, err_detail
FROM fis_consume_record
WHERE dt = '2026-07-08' AND status = 2
ORDER BY md5, uuid
LIMIT 20 OFFSET 0;

-- type=mismatched：见第 10 条（内容级差异）
```

### 8. record ↔ detail 内部一致性

```sql
SELECT r.md5, r.parsed_line_count,
       COUNT(d.uuid) AS detail_count,
       r.parsed_line_count - COUNT(d.uuid) AS diff
FROM fis_decode_record r
LEFT JOIN fis_decode_detail d ON r.dt = d.dt AND r.md5 = d.md5
WHERE r.dt = '2026-07-08'
GROUP BY r.md5, r.parsed_line_count
HAVING diff <> 0
ORDER BY ABS(diff) DESC
LIMIT 20;
```

> 只统计 `status IN (1,3)`（success/partial）的 record，status=2（完全失败，本就无 detail）不算异常。

### 9. uuid 来源细分（real vs gen_fallback）

```sql
SELECT
  CASE WHEN d.uuid LIKE 'gen:%' THEN 'gen_fallback' ELSE 'real' END AS uuid_source,
  COUNT(*) AS expected,
  COUNT(c.uuid) AS matched,
  SUM(CASE WHEN c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
FROM fis_decode_detail d
LEFT JOIN fis_consume_record c ON d.dt=c.dt AND d.md5=c.md5 AND d.uuid=c.uuid
WHERE d.dt = '2026-07-08' AND d.send_status = 1
GROUP BY uuid_source;
```

> `gen_fallback` 分支预期恒为 0（兜底逻辑已在上游去掉）。一旦非 0，说明代码被回退或重新引入了兜底生成，需要报警级关注。

### 10. event 内容级差异（uuid 匹配但字段不一致）

```sql
SELECT d.dt, d.md5, d.uuid,
       d.module_name AS detail_module, c.module_name AS consume_module,
       d.project AS detail_project, c.project AS consume_project
FROM fis_decode_detail d
JOIN fis_consume_record c ON d.dt=c.dt AND d.md5=c.md5 AND d.uuid=c.uuid
WHERE d.dt = '2026-07-08'
  AND (d.module_name <> c.module_name OR d.project <> c.project)
LIMIT 50;
```

---

## 六、实现建议

### 性能
1. **限定时间范围**：三表按 dt 分区，查单天走分区裁剪很快；趋势查询 ≤30 天
2. **多消费单独查**：反向 LEFT JOIN 数据量大时慢，单独接口 + limit
3. **缓存**：总览和趋势 1-5 分钟刷新一次即可（对账表是分钟级攒批写入），可用 Redis 或内存缓存

### 权限
- Doris 账号给只读（SELECT）权限，限 fdi_dev 库
- 看板服务不写入任何数据

### 前端看板布局
```
┌─────────────────────────────────────────────────┐
│  日期选择器 [2026-07-08]                          │
├──────────────┬──────────────┬──────────────────┤
│  匹配率       │  漏消费       │  多消费           │
│   100%       │    0         │    0             │
├──────────────┴──────────────┴──────────────────┤
│  时间趋势（折线图：期望/实际/匹配）                 │
│                                                   │
├───────────────────────┬─────────────────────────┤
│  模块维度（表格）       │  上游解码状态（饼图）      │
│  filter 5000 100%     │   success 95%           │
│  fdr     4000 100%    │   failed  5%            │
├───────────────────────┴─────────────────────────┤
│  项目维度（表格，按 missing 降序，点击行钻取该项目内模块分布）│
│  proj-a  8000  99.4%  [查看模块分布]              │
│  proj-b  4329  100%                              │
├───────────────────────┴─────────────────────────┤
│  差异 md5 排行（表格，可点击查看明细）              │
│  md5-xxx  missing 50  [查看]                     │
├───────────────────────┬─────────────────────────┤
│  record↔detail 一致性  │  uuid 来源细分            │
│  md5-yyy diff=3 [查看] │  real 100%  gen_fallback 0│
├───────────────────────┴─────────────────────────┤
│  event 明细列表（独立 Tab，不依赖先选 md5）          │
│  筛选：模块 [全部▾] 类型 [漏消费/多消费/失败/字段不一致▾]│
│  dt        md5      uuid      module   状态       │
│  07-08   abc123   real-uu..  fff_close  漏消费     │
└─────────────────────────────────────────────────┘
```

---

## 七、相关文档

- [DATA_FLOW_AND_RECONCILIATION.md](DATA_FLOW_AND_RECONCILIATION.md) — 数据流向与对账链路
- [RECONCILIATION_TEST_REPORT.md](RECONCILIATION_TEST_REPORT.md) — 对账结果报告
- [RECONCILIATION_ISSUES_SUMMARY.md](RECONCILIATION_ISSUES_SUMMARY.md) — 对账问题排查总结
- [UPSTREAM_DATA_LOSS_RISKS.md](UPSTREAM_DATA_LOSS_RISKS.md) — 上游丢数据风险
