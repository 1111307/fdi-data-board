# FO / DO Dashboard 接口文档（聚合表版）

> 实现版本：迁移到 `*_daily_summary` 汇总表后的当前工作区代码。
> 数据源：Doris（GORM + MySQL 协议）。路由：`internal/route/dashboard.go`；实现：`internal/data/dashboard_do.go`、`dashboard_fo.go`。
>
> ⚠️ 全局注意：
> 1. Doris 字符串比较**大小写敏感**，`forever_log` 实际为小写。
> 2. 汇总表同时写入真实 `event_name` 和 `__ALL__` 汇总行；按事件展开的查询排除 `__ALL__`，跨事件聚合的查询只查 `__ALL__`。
> 3. 多天查询时 `SUM(vehicle_count)` 为**车辆日累加**口径，非全周期 `COUNT(DISTINCT)`。
> 4. `reason`/`stage_reason` 粒度无 `filter_name`，相关接口按**是否传 filter_name 分流**（见下文）。

## 汇总表与粒度约定

| 汇总表 | 可用粒度 summary_grain |
|---|---|
| `fdi.dwd_cfdi_status_monitor_analysis_daily_summary` | `overview` / `filter` / `stage_status` / `stage_reason` |
| `fdi.dwd_cfdi_basic_fff_trigger_daily_summary` | `overview` / `filter` / `status` / `reason` |
| `fdi.dwd_cfdi_basic_fff_close_daily_summary` | `overview` / `filter` / `reason` |
| `fdi.dwd_cfdi_basic_fff_running_daily_summary` | `overview` / `filter` / `status` |
| `fdi.dwd_basic_fdr_trigger_daily_summary` | `overview` / `status` / `reason` |
| `fdi.dwd_cfdi_basic_fcl_trigger_daily_summary` | `overview` / `status` / `reason` / `source` |
| `fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary` | `overview` / `source` |
| `fdi.ads_cfdi_vehicle_daily_summary` | （ADS，无 grain） |

**粒度选择规则**（status_monitor 类）：传了 `filter_name` 用 `filter` 粒度，否则用 `overview`。

```sql
summary_grain = '<overview|filter>'   -- 由 grainForFilter(filterName) 决定
[AND filter_name = ?]                  -- 仅传了 filter_name 时
```

---

# 一、DO 模块（`/dashboard/v1/do/*`）

公共入参：`filter_name, event_names, project_name, car_types, start_dt, end_dt`（日期不传默认近 7 天）。

## 1. GET /do/overview — 事件横向对比 ✅ 汇总表

表 `status_monitor_summary`，按事件展开（排除 `__ALL__`/`forever_log`）：

```sql
SELECT event_name,
  SUM(event_count)       AS trigger_count,
  SUM(fff_success_count) AS fff_count,
  SUM(fdr_success_count) AS fdr_count,
  SUM(fcl_success_count) AS fcl_count,
  SUM(vehicle_count)     AS vehicle_count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN ? AND ?
  AND summary_grain = '<overview|filter>'
  AND event_name != '__ALL__' AND event_name != 'forever_log'
  [AND event_name IN (...)] [AND filter_name=?] [AND project_name=?] [AND car_type...]
GROUP BY event_name ORDER BY trigger_count DESC
```

## 2. GET /do/trend — 数据总览趋势 ✅ 汇总表

```sql
SELECT dt, SUM(event_count) AS total, SUM(fcl_success_count) AS success
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='<overview|filter>' AND event_name='__ALL__' [+ 维度]
GROUP BY dt ORDER BY dt ASC
```

## 3. GET /do/fail_reason — 失败原因分析 ⚠️ 条件分支

- **未传 filter_name** → 汇总表 `stage_reason` 粒度（三段 UNION，fff/fdr/fcl_failed_count）
- **传 filter_name** → 回退 `ads_do_cfdi_daily` 明细版三阶段 UNION

```sql
-- 汇总表路径（filter_name=''）
SELECT stage, detail_tag, SUM(cnt) FROM (
  SELECT 'FFF' stage, fff_detail_tag detail_tag, SUM(fff_failed_count) cnt
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN ? AND ? AND summary_grain='stage_reason'
    AND fff_detail_tag NOT IN ('__ALL__','') [+ 维度] GROUP BY fff_detail_tag
  UNION ALL
  SELECT 'FDR', fdr_detail_tag, SUM(fdr_failed_count) ... GROUP BY fdr_detail_tag
  UNION ALL
  SELECT 'FCL', fcl_detail_tag, SUM(fcl_failed_count) ... GROUP BY fcl_detail_tag
) t GROUP BY stage, detail_tag ORDER BY stage, cnt DESC
```

## 4. GET /do/cool_top — 冷却 Top20 ❌ 保留明细

`reason` 粒度无 `filter_name`，无法按筛选器排名 + cooldown 组合，仍查 `ads_do_cfdi_daily`（待新增 `filter_reason` 粒度）。

## 5. GET /do/trigger_rank — 触发频次 Top10 ✅ 汇总表

表 `fff_trigger_summary` **filter 粒度**（按 filter_name 排名）：

```sql
SELECT filter_name, SUM(event_count) AS cnt
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='filter'
  AND filter_name != '__ALL__' AND filter_name != '' [+ 维度]
GROUP BY filter_name ORDER BY cnt DESC LIMIT 10
```

## 6. GET /do/sw_version — 软件版本分布 ✅ 汇总表

```sql
SELECT fff_sw_version AS sw_version, SUM(event_count) AS cnt
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='<overview|filter>'
  AND fff_sw_version IS NOT NULL AND fff_sw_version != '' [+ 维度]
GROUP BY fff_sw_version ORDER BY cnt DESC LIMIT 20
```

## 7. GET /do/project_car — 项目×车型分布 ✅ 汇总表

```sql
SELECT project_name, car_type, SUM(event_count) AS cnt
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='<overview|filter>'
  AND project_name != '' AND car_type != '' [+ 维度]
GROUP BY project_name, car_type ORDER BY project_name, car_type
```

## 8. GET /do/mem_top — FDR 内存不足 Top20 ✅ 汇总表

表 `fdr_trigger_summary` reason 粒度：

```sql
SELECT event_name, SUM(failed_count) AS cnt
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='reason'
  AND detail_tag IN ('full_gc','mem_pool_water_line')
  AND event_name != '__ALL__' AND event_name != '' [+ 维度]
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 9. GET /do/disk_top — FDR 磁盘不足 Top20 ✅ 汇总表

同上，`detail_tag IN ('disk_overrun','max_files_exceeded')`。

## 10. GET /do/close_top — 关闭次数 Top 筛选器 ✅ 汇总表

```sql
SELECT filter_name, SUM(close_count) AS cnt
FROM fdi.dwd_cfdi_basic_fff_close_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='filter'
  AND filter_name NOT IN ('__ALL__','') [+ 维度]
GROUP BY filter_name ORDER BY cnt DESC LIMIT 20
```

## 11. GET /do/quota_top — FCL Quota 超限 Top20 ✅ 汇总表

```sql
SELECT event_name, SUM(failed_count) AS cnt
FROM fdi.dwd_cfdi_basic_fcl_trigger_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='reason'
  AND detail_tag='quota_exceeded'
  AND event_name != '__ALL__' AND event_name != '' [+ 维度]
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 12. GET /do/project_event — 项目触发回流事件总数 ✅ 汇总表

```sql
SELECT project_name,
  COUNT(DISTINCT CASE WHEN fcl_success_count > 0 THEN event_name END) AS event_count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='<overview|filter>'
  AND event_name != '__ALL__' AND event_name != 'forever_log'
  AND project_name != '' [+ 维度]
GROUP BY project_name ORDER BY event_count DESC
```

## 13. GET /do/net_speed — 各车型平均上传带宽 ✅ 汇总表

表 `fcl_upload_summary`，`AVG` 改为 `SUM(sum)/SUM(count)`：

```sql
SELECT dt, car_type,
  ROUND(IFNULL(SUM(upload_bandwidth_sum)/NULLIF(SUM(upload_bandwidth_count),0),0),2) AS avg_bw
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='overview' AND event_name='__ALL__'
  AND car_type != '' [+ 维度]
GROUP BY dt, car_type ORDER BY dt, car_type
```

## 14. GET /do/fcl_bw — FCL 整体平均上传带宽 ✅ 汇总表

同 13，去掉 `car_type` 分组。

## 15. GET /do/top_vehicles — Top20 活跃车辆 ❌ 保留明细

需 `anonymous_id` 行级字段，仍查 `dwd_cfdi_status_monitor_analysis`（含每车主失败原因补充查询）。

## 16. GET /do/anomaly_vehicles — 异常车辆 ❌ 保留明细

同 15，需单车成功率，查明细表。

## 17. GET /do/active_trend — 活跃车辆趋势 ✅ 汇总表

```sql
SELECT dt, SUM(vehicle_count) AS active_count
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt BETWEEN ? AND ? [+ event/project/car_type]
GROUP BY dt ORDER BY dt
```

## 18. GET /do/funnel — DO 漏斗 ⚠️ 条件分支

- **节点统计**（任何情况）→ 汇总表 `overview|filter` 粒度 + `event_name='__ALL__'`
- **失败原因 Top10**：未传 filter_name → 汇总表 `stage_reason`；传了 → 返回空（待 `filter_stage_reason` 粒度）

```sql
SELECT SUM(event_count) fff_total, SUM(fff_success_count) fff_allow,
  SUM(fdr_success_count) fdr_success, SUM(fdr_failed_count) fdr_fail,
  SUM(fcl_success_count) fcl_success, SUM(fcl_failed_count) fcl_fail
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='<overview|filter>' AND event_name='__ALL__' [+ 维度]
```

---

# 二、FO 模块

## 1. GET /dashboard/v1/dimensions — 下拉维度 ✅ 汇总表（60min 缓存 + singleflight）

```sql
SELECT DISTINCT filter_name FROM fdi.dwd_cfdi_basic_fff_running_daily_summary
WHERE dt>=DATE_SUB(CURDATE(),INTERVAL 7 DAY) AND summary_grain='filter' AND filter_name NOT IN ('__ALL__','');

SELECT DISTINCT project_name FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt>=DATE_SUB(CURDATE(),INTERVAL 7 DAY) AND project_name != '';

SELECT DISTINCT event_name FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt>=DATE_SUB(CURDATE(),INTERVAL 7 DAY) AND event_name NOT IN ('__ALL__','');

SELECT DISTINCT car_type FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt>=DATE_SUB(CURDATE(),INTERVAL 7 DAY) AND car_type != '';
```

## 2. GET /dashboard/v1/diag/funnel — 数采全链路 ⚠️ 条件分支

与 DO #18 完全相同（同一实现）。

## 3–7. 明细分页接口 ❌ 保留明细（行级字段）

| 接口 | 表 |
|---|---|
| GET /fo/detail/running | `dwd_cfdi_basic_fff_running` |
| GET /fo/detail/trigger | `dwd_cfdi_basic_fff_trigger` |
| GET /fo/detail/close | `dwd_cfdi_basic_fff_close` |
| GET /fo/detail/fdr | `dwd_basic_fdr_trigger` |
| GET /fo/detail/fcl | `dwd_cfdi_basic_fcl_trigger` |

均为 COUNT + 数据双 SQL 并发，`LIMIT/OFFSET` 分页。

## 8. GET /fo/detail/uuid — 全链路明细 ❌ 保留明细

查 `dwd_cfdi_status_monitor_analysis`，支持 `only_fail`/`stage_filter`。

## 9. GET /fo/diag/stage_trend — 三阶段触发趋势 ⚠️ 条件分支

- **未传 filter_name** → 汇总表：成功量查 `overview` 粒度，失败拆分查 `stage_reason` 粒度（按 detail_tag 分序列）
- **传 filter_name** → 回退 `ads_do_cfdi_daily`（明细版 26 列 CASE WHEN）

## 10. GET /fo/diag/close_reason — 算子关闭原因 ⚠️ 条件分支

- **未传 filter_name** → 汇总表 `fff_close` reason 粒度（`close_reason_tag`，注意：仅 3 类，明细版的 `not enough memory`/`with error` 被并入 `other`）
- **传 filter_name** → 回退 `dwd_cfdi_basic_fff_close` INSTR 归类

## 11. GET /fo/running/trend — 算子活跃车辆趋势 ✅ 汇总表

入参必填 `filter_name, start_dt, end_dt`。表 `fff_running_summary` filter 粒度：

```sql
SELECT dt, SUM(vehicle_count) AS vehicle_count
FROM fdi.dwd_cfdi_basic_fff_running_daily_summary
WHERE summary_grain='filter' AND switch_on=1 AND filter_name=? AND dt BETWEEN ? AND ?
  [AND project_name=?]
GROUP BY dt ORDER BY dt
```

---

# 三、新增质量/概览指标接口（2026-08-13）

对应前端「待接入」指标，全部走汇总表。
**P95 聚合口径**：汇总表每个（天×维度）分桶各存一个 P95，跨桶**不能 `MAX`**（小样本脏桶会放大离群值，曾导致碎片率 999%、落盘 948738s），改用 **count 加权均值** `SUM(p95*count)/SUM(count)` —— 快且接近明细精确 P95。

| 指标 | 加权 P95（汇总表） | 明细精确 P95 | MAX（错误） |
|---|---|---|---|
| 碎片率 | 171.5% | 277% | 999% ❌ |
| 落盘耗时 | 12.2 s | 13.8 s | 948738 s ❌ |
| Bag 大小 | 233.7 MB | 309 MB | 1645 MB ❌ |

## 19. GET /do/fdr_quality — FDR 质量 P95 ✅ 汇总表

表 `fdr_trigger_summary`（overview 粒度）。返回 TD 磁盘/TM 内存/落盘耗时 P95 + 落盘总数/成功数。

```sql
SELECT ROUND(SUM(td_mb_p95*td_mb_count)/NULLIF(SUM(td_mb_count),0),2) td_mb_p95,
  ROUND(SUM(tm_mb_p95*tm_mb_count)/NULLIF(SUM(tm_mb_count),0),2) tm_mb_p95,
  ROUND(SUM(time_cost_ms_p95*time_cost_ms_count)/NULLIF(SUM(time_cost_ms_count),0),2) time_cost_ms_p95,
  SUM(event_count) fdr_total, SUM(success_count) fdr_success
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='overview' AND event_name='__ALL__' [+ 维度]
```

## 20. GET /do/fcl_quality — FCL Bag 大小质量 ✅ 汇总表

表 `fcl_upload_summary`（overview 粒度）。返回 Bag 大小 P95 + 上传总数（avg/max 已移除，易受离群值影响）。

```sql
SELECT ROUND(SUM(package_size_p95*package_size_count)/NULLIF(SUM(package_size_count),0),2) bag_size_p95,
  SUM(upload_count) upload_total
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='overview' AND event_name='__ALL__' [+ 维度]
```

## 21. GET /do/fdr_fragment — FDR 碎片率 ✅ 汇总表

表 `fdr_fragment_summary`（overview 粒度 + `fragment_field='total_fragment'`）。返回碎片率 P95（avg/max 已移除）。

```sql
SELECT ROUND(SUM(fragment_value_p95*fragment_value_count)/NULLIF(SUM(fragment_value_count),0),2) fragment_p95
FROM fdi.dwd_cfdi_basic_fdr_fragment_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='overview' AND event_name='__ALL__'
  AND fragment_field='total_fragment' [+ 维度]
```

## 12(FO). GET /fo/running/overview — 筛选器运行健康概览 ✅ 汇总表

表 `fff_running_summary`（**filter 粒度**，才能 `COUNT(DISTINCT filter_name)`）。占比/成功率在应用层算。

```sql
SELECT SUM(running_count) running_total, SUM(vehicle_count) vehicle_total,
  SUM(switch_on_count) switch_on_total, SUM(switch_off_count) switch_off_total,
  SUM(running_success_count) running_success, SUM(running_failed_count) running_failed,
  COUNT(DISTINCT filter_name) filter_count
FROM fdi.dwd_cfdi_basic_fff_running_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='filter' AND filter_name NOT IN ('__ALL__','') [+ 维度]
```

## 13(FO). GET /fo/fff/overview — FFF 触发概览 ✅ 汇总表

表 `fff_trigger_summary`（粒度按 `grainForFilter(filter_name)`）。成功率应用层算。

```sql
SELECT SUM(event_count) trigger_total, SUM(success_count) trigger_success, SUM(failed_count) trigger_failed
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN ? AND ? AND summary_grain='<overview|filter>' AND event_name='__ALL__' [+ 维度]
```

---

## 汇总：接口 → 数据源对照

| 状态 | 接口 |
|---|---|
| ✅ 纯汇总表 | overview, trend, sw_version, project_car, mem_top, disk_top, close_top, quota_top, project_event, net_speed, fcl_bw, active_trend, trigger_rank, dimensions, running/trend, **fdr_quality, fcl_quality, fdr_fragment, running/overview, fff/overview** |
| ⚠️ 条件分支（未传 filter 用汇总，传了回退明细） | fail_reason, funnel(DO/FO), stage_trend, close_reason |
| ❌ 保留明细表 | cool_top, top_vehicles, anomaly_vehicles, 5 个 detail 分页, uuid |
