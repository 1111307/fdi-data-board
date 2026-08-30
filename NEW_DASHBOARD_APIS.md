# Dashboard 待接入指标 — 后端接口实现文档

> 对应前端标注「待接入」的 5 个指标。数据源：Doris `*_daily_summary` 汇总表（已回刷 2026-06-01 ~ 2026-07-28）。
> 风格对齐 `FO_DO_DASHBOARD_API.md`。实现位于 `internal/data/dashboard_do.go` / `dashboard_fo.go`。
>
> ⚠️ **P95 聚合注意**：`PERCENTILE` 类指标**不可跨天/跨维度直接相加**。汇总表的 `*_p95` 是每个（天 × 粒度 × 维度）分桶内的 P95。
> - **单日 + 单一维度**：P95 精确。
> - **多天 / 全维度**：需 `MAX(*_p95)`（保守上界）或按 `*_count` 加权（近似），或退化明细表 `PERCENTILE(col, 0.95)` 精确算（慢）。
> 本文默认：KPI 单值用 `MAX(*_p95)`；趋势图按天展示每日 P95。

## 通用 WHERE（与现有汇总表接口一致）

```sql
dt BETWEEN :start_dt AND :end_dt
AND summary_grain = 'overview'
AND event_name = '__ALL__'
[AND (:project_name='' OR project_name=:project_name)]
[AND (:car_types_empty=1 OR car_type IN (:car_types))]
```

---

## 1. FDR 质量 P95 — `GET /dashboard/v1/do/fdr_quality`

前端 FDR 块 3 个 KPI：TD 磁盘 P95、TM 内存 P95、落盘耗时 P95。

**表**：`fdi.dwd_basic_fdr_trigger_daily_summary`（字段 `td_mb_p95` / `tm_mb_p95` / `time_cost_ms_p95` 全有）

```sql
SELECT
  MAX(td_mb_p95)        AS td_mb_p95,
  MAX(tm_mb_p95)        AS tm_mb_p95,
  MAX(time_cost_ms_p95) AS time_cost_ms_p95,
  SUM(event_count)      AS fdr_total,
  SUM(success_count)    AS fdr_success
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  [AND (:project_name='' OR project_name=:project_name)]
  [AND (:car_types_empty=1 OR car_type IN (:car_types))]
```

可选按天趋势：

```sql
SELECT dt, MAX(td_mb_p95) td_mb_p95, MAX(tm_mb_p95) tm_mb_p95, MAX(time_cost_ms_p95) time_cost_ms_p95
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE ... GROUP BY dt ORDER BY dt
```

**返回结构**：

```json
{
  "code": 0, "message": "OK",
  "td_mb_p95": 512.3,
  "tm_mb_p95": 256.1,
  "time_cost_ms_p95": 8300,
  "fdr_total": 1391656,
  "fdr_success": 1380000
}
```

---

## 2. 筛选器运行健康概览 — `GET /dashboard/v1/fo/running/overview`

主链路 Running/Switch 节点 + FFF 块「运行筛选器数量/运行车辆数」。

**表**：`fdi.dwd_cfdi_basic_fff_running_daily_summary`（字段 `running_count` / `vehicle_count` / `switch_on_count` / `switch_off_count` / `running_success_count` / `running_failed_count`）

```sql
SELECT
  SUM(running_count)         AS running_total,
  SUM(vehicle_count)         AS vehicle_total,
  SUM(switch_on_count)       AS switch_on_total,
  SUM(switch_off_count)      AS switch_off_total,
  SUM(running_success_count) AS running_success,
  SUM(running_failed_count)  AS running_failed,
  COUNT(DISTINCT filter_name) AS filter_count
FROM fdi.dwd_cfdi_basic_fff_running_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter'          -- filter 粒度才能 COUNT(DISTINCT filter_name)
  AND filter_name != '__ALL__' AND filter_name != ''
  [AND (:project_name='' OR project_name=:project_name)]
  [AND (:car_types_empty=1 OR car_type IN (:car_types))]
```

> 占比在应用层算：`switch_on_ratio = switch_on_total / (switch_on_total + switch_off_total)`。
> `vehicle_total` 多天为车辆日口径（偏大），精确唯一车辆需明细 `COUNT(DISTINCT anonymous_id)`。

**返回结构**：

```json
{
  "code": 0, "message": "OK",
  "running_total": 14695042,
  "vehicle_total": 490889,
  "switch_on_total": 8000000,
  "switch_off_total": 6695042,
  "switch_on_ratio": 54.4,
  "running_success": 14000000,
  "running_failed": 695042,
  "filter_count": 120
}
```

---

## 3. FFF 触发概览 — `GET /dashboard/v1/fo/fff/overview`

FFF 块「触发总数 / 触发成功数 / 成功率」。

**表**：`fdi.dwd_cfdi_basic_fff_trigger_daily_summary`（`event_count` / `success_count` / `failed_count`）

```sql
SELECT
  SUM(event_count)   AS trigger_total,
  SUM(success_count) AS trigger_success,
  SUM(failed_count)  AS trigger_failed
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  [AND (:filter_name='' OR filter_name=:filter_name)]   -- 若按筛选器过滤需 filter 粒度
  [AND (:project_name='' OR project_name=:project_name)]
  [AND (:car_types_empty=1 OR car_type IN (:car_types))]
```

> 成功率应用层算：`trigger_success / trigger_total`。
> 若要按 `filter_name` 过滤，改用 `summary_grain='filter' AND filter_name=:filter_name`（`grainForFilter` 规则）。

**返回结构**：

```json
{
  "code": 0, "message": "OK",
  "trigger_total": 1286565,
  "trigger_success": 1200000,
  "trigger_failed": 86565,
  "trigger_success_rate": 93.3
}
```

---

## 4. FCL Bag 大小 P95 — `GET /dashboard/v1/do/fcl_quality`

FCL 块「Bag 大小 P95」。

**表**：`fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary`（`package_size_p95` / `package_size_avg` / `package_size_max`）

```sql
SELECT
  MAX(package_size_p95) AS bag_size_p95,
  MAX(package_size_max) AS bag_size_max,
  ROUND(IFNULL(SUM(package_size_sum)/NULLIF(SUM(package_size_count),0),0),2) AS bag_size_avg,
  SUM(upload_count) AS upload_total
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  [AND (:project_name='' OR project_name=:project_name)]
  [AND (:car_types_empty=1 OR car_type IN (:car_types))]
```

> 注意：明细版 `net_speed` 有 `package_size > 10485760`(10MB) 的过滤条件。若 Bag 大小也要同样过滤小文件，汇总表无法按行过滤（已预聚合），口径会偏大——需业务确认是否接受，或退化明细表 `PERCENTILE(package_size, 0.95) WHERE package_size > 10485760`。

**返回结构**：

```json
{
  "code": 0, "message": "OK",
  "bag_size_p95": 52428800,
  "bag_size_avg": 31457280,
  "bag_size_max": 104857600,
  "upload_total": 310587
}
```

---

## 5. 碎片率 P95 — `GET /dashboard/v1/do/fdr_fragment`（新增，已确认可做）

**结论更新**：原分析认为碎片率无字段，但实测发现**专用汇总表已存在**，无需补字段。

**表**：`fdi.dwd_cfdi_basic_fdr_fragment_daily_summary`
- `fragment_field` 区分碎片类型，**总碎片率取 `fragment_field='total_fragment'`**
- 指标列：`fragment_value_avg` / `fragment_value_p95` / `fragment_value_max` / `fragment_count`
- 粒度：`overview`（总量）/ `field`（按 fragment_field 细分）

```sql
SELECT
  MAX(fragment_value_p95) AS fragment_p95,
  MAX(fragment_value_max) AS fragment_max,
  ROUND(IFNULL(SUM(fragment_value_sum)/NULLIF(SUM(fragment_value_count),0),0),2) AS fragment_avg
FROM fdi.dwd_cfdi_basic_fdr_fragment_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  AND fragment_field = 'total_fragment'
  [AND (:project_name='' OR project_name=:project_name)]
  [AND (:car_types_empty=1 OR car_type IN (:car_types))]
```

按碎片类型细分（前端若要展示各类碎片占比）：

```sql
SELECT fragment_field, MAX(fragment_value_p95) AS p95, SUM(fragment_count) AS cnt
FROM fdi.dwd_cfdi_basic_fdr_fragment_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'field'
  AND fragment_field != 'total_fragment'
GROUP BY fragment_field ORDER BY p95 DESC
```

> 精确 P95 可退化明细表：`SELECT PERCENTILE(fragment_value, 0.95) FROM dwd_cfdi_basic_fdr_fragment WHERE ...`（明细表 `dwd_cfdi_basic_fdr_fragment` 已确认存在）。

**返回结构**：

```json
{
  "code": 0, "message": "OK",
  "fragment_p95": 27.86,
  "fragment_avg": 21.24,
  "fragment_max": 468.14
}
```

---

## 汇总：5 个接口落地清单

| 接口 | 汇总表 | 关键字段 | 状态 |
|---|---|---|---|
| `GET /do/fdr_quality` | `dwd_basic_fdr_trigger_daily_summary` | `td/tm/time_cost_ms_p95` | ✅ 可做 |
| `GET /fo/running/overview` | `dwd_cfdi_basic_fff_running_daily_summary` | `running/vehicle/switch_on/off_count` | ✅ 可做 |
| `GET /fo/fff/overview` | `dwd_cfdi_basic_fff_trigger_daily_summary` | `event/success/failed_count` | ✅ 可做 |
| `GET /do/fcl_quality` | `dwd_cfdi_basic_fcl_uploadinfo_daily_summary` | `package_size_p95` | ✅ 可做 |
| `GET /do/fdr_fragment` | `dwd_cfdi_basic_fdr_fragment_daily_summary` | `fragment_value_p95` | ✅ **可做（表已存在）** |

## 待业务确认的 2 个口径点

1. **P95 跨天/跨维度**：单值用 `MAX(*_p95)`（保守）还是按 count 加权（近似），还是退化明细 `PERCENTILE`（精确但慢）？建议趋势图用每日 P95、KPI 单值用 MAX。
2. **FCL Bag 大小**：明细版有 `package_size > 10MB` 过滤，汇总表无法按行过滤，会导致 P95 偏大。是否接受？或 Bag 大小也退化明细表精确算。
