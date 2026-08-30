# DO / FO Dashboard 接口与 SQL 查询文档

> 数据源：Doris（OLAP），通过 GORM + MySQL 协议（9030 端口）查询。
> 路由注册：`internal/route/dashboard.go`；SQL 实现：`internal/data/dashboard_do.go`、`internal/data/dashboard_fo.go`。

## 涉及的 Doris 表

| 表名 | 说明 |
|------|------|
| `ads_do_cfdi_daily` | DO 侧按天聚合表（cnt + 三阶段状态/明细 tag），DO 看板大部分接口使用 |
| `dwd_cfdi_status_monitor_analysis` | 全链路状态明细表（uuid 维度），FO/DO 共用 |
| `dwd_cfdi_basic_fff_running` | 筛选器（算子）运行明细 |
| `dwd_cfdi_basic_fff_trigger` | 筛选器触发明细 |
| `dwd_cfdi_basic_fff_close` | 筛选器关闭明细 |
| `dwd_basic_fdr_trigger` | FDR 落盘明细 |
| `dwd_cfdi_basic_fcl_trigger` | FCL 上传明细 |
| `dwd_cfdi_basic_fcl_uploadinfo` | FCL 上传带宽信息 |

## 公共 WHERE 条件构建

`buildDoCommonWhere`（`internal/data/dashboard_do.go:789`）— DO 聚合接口共用：

```sql
WHERE dt BETWEEN ? AND ?            -- 或不传日期时：dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND filter_name = ?               -- 可选
  AND event_name = ? / IN (...)     -- 可选，多选
  AND project_name = ?              -- 可选
  AND car_type = ? / IN (...)       -- 可选，多选
```

`buildVehicleWhere`（`:739`）— 车辆维度（`dwd_cfdi_status_monitor_analysis`），同上但无 `filter_name`。

FO 明细接口的 `buildXxxWhere` 类似，日期不传时默认 `dt = CURDATE()`。

---

# 一、DO 模块（`/dashboard/v1/do/*`）

## 1. GET /dashboard/v1/do/overview — 事件横向对比

入参：`filter_name, event_names, project_name, car_types, start_dt, end_dt`

**并发执行 2 条 SQL：**

SQL①（触发/三阶段计数，表 `ads_do_cfdi_daily`）：

```sql
SELECT event_name,
  SUM(cnt) AS trigger_count,
  SUM(CASE WHEN fff_status != 'discard' THEN cnt ELSE 0 END) AS fff_count,
  SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') THEN cnt ELSE 0 END) AS fdr_count,
  SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status != 'discard' THEN cnt ELSE 0 END) AS fcl_count
FROM ads_do_cfdi_daily
WHERE <公共条件> AND event_name != 'forever_log'
GROUP BY event_name ORDER BY trigger_count DESC
```

SQL②（车辆数，表 `dwd_cfdi_status_monitor_analysis`）：

```sql
SELECT event_name, COUNT(DISTINCT anonymous_id) AS vehicle_count
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件> AND event_name != 'forever_log'
GROUP BY event_name
```

## 2. GET /dashboard/v1/do/trend — 数据总览趋势

```sql
SELECT dt,
  SUM(cnt) AS total,
  SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status != 'discard' THEN cnt ELSE 0 END) AS success
FROM ads_do_cfdi_daily
WHERE <公共条件>
GROUP BY dt ORDER BY dt ASC
```

## 3. GET /dashboard/v1/do/fail_reason — 失败原因分析（三阶段归因）

```sql
SELECT stage, detail_tag, SUM(cnt) AS cnt
FROM (
  SELECT 'FFF' AS stage, fff_detail_tag AS detail_tag, cnt
  FROM ads_do_cfdi_daily
  WHERE <公共条件> AND fff_status = 'discard' AND fff_detail_tag IS NOT NULL AND fff_detail_tag != ''
  UNION ALL
  SELECT 'FDR', fdr_detail_tag, cnt
  FROM ads_do_cfdi_daily
  WHERE <公共条件> AND fff_status != 'discard' AND fdr_status != 'success' AND fcl_status = '' AND fdr_detail_tag IS NOT NULL AND fdr_detail_tag != ''
  UNION ALL
  SELECT 'FCL', fcl_detail_tag, cnt
  FROM ads_do_cfdi_daily
  WHERE <公共条件> AND fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag IS NOT NULL AND fcl_detail_tag != ''
) t
GROUP BY stage, detail_tag
ORDER BY stage, cnt DESC
```

> 注：公共条件的 args 需要重复传 3 份（每个 UNION 分支一份）。

## 4. GET /dashboard/v1/do/cool_top — 冷却 Top20 筛选器

```sql
SELECT filter_name, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件> AND fff_status != 'success' AND fff_detail_tag = 'cooldown' AND filter_name IS NOT NULL
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 20
```

## 5. GET /dashboard/v1/do/trigger_rank — 触发频次排行 Top10

```sql
SELECT filter_name, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件> AND filter_name IS NOT NULL
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 10
```

## 6. GET /dashboard/v1/do/sw_version — 软件版本分布

```sql
SELECT fff_sw_version AS sw_version, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件> AND fff_sw_version IS NOT NULL AND fff_sw_version != ''
GROUP BY fff_sw_version
ORDER BY cnt DESC
LIMIT 20
```

## 7. GET /dashboard/v1/do/project_car — 项目×车型分布

```sql
SELECT project_name, car_type, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件>
  AND project_name IS NOT NULL AND project_name != ''
  AND car_type IS NOT NULL AND car_type != ''
GROUP BY project_name, car_type
ORDER BY project_name, car_type
```

## 8. GET /dashboard/v1/do/mem_top — FDR 内存不足 Top20

```sql
SELECT event_name, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件>
  AND fff_status != 'discard' AND fdr_status != 'success' AND fcl_status = ''
  AND fdr_detail_tag = 'memory'
  AND event_name IS NOT NULL AND event_name != ''
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 9. GET /dashboard/v1/do/disk_top — FDR 磁盘不足 Top20

```sql
SELECT event_name, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件>
  AND fff_status != 'discard' AND fdr_status != 'success' AND fcl_status = ''
  AND fdr_detail_tag = 'disk'
  AND event_name IS NOT NULL AND event_name != ''
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 10. GET /dashboard/v1/do/close_top — 关闭次数 Top 筛选器

```sql
SELECT filter_name, COUNT(*) AS cnt
FROM dwd_cfdi_basic_fff_close
WHERE <公共条件> AND filter_name IS NOT NULL AND filter_name != ''
GROUP BY filter_name ORDER BY cnt DESC LIMIT 20
```

## 11. GET /dashboard/v1/do/quota_top — FCL Quota 超限 Top20

```sql
SELECT event_name, SUM(cnt) AS cnt
FROM ads_do_cfdi_daily
WHERE <公共条件>
  AND fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard'
  AND fcl_detail_tag = 'quota_exceeded'
  AND event_name IS NOT NULL AND event_name != ''
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 12. GET /dashboard/v1/do/project_event — 项目触发回流事件总数

```sql
SELECT project_name, COUNT(DISTINCT event_name) AS event_count
FROM ads_do_cfdi_daily
WHERE <公共条件>
  AND fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status != 'discard'
  AND project_name IS NOT NULL AND project_name != ''
GROUP BY project_name ORDER BY event_count DESC
```

## 13. GET /dashboard/v1/do/net_speed — 各车型平均上传带宽

```sql
SELECT DATE(create_at) AS dt, car_type,
  ROUND(AVG((package_size / 1024.0 / 1024.0) / (total_cost / 1000.0)), 2) AS avg_bw
FROM dwd_cfdi_basic_fcl_uploadinfo
WHERE dt BETWEEN ? AND ?   -- 默认近7天；可选 project_name / car_type
  AND package_size > 10485760 AND total_cost > 0
  AND car_type IS NOT NULL AND car_type != ''
GROUP BY DATE(create_at), car_type ORDER BY dt, car_type
```

## 14. GET /dashboard/v1/do/fcl_bw — FCL 整体平均上传带宽

```sql
SELECT DATE(create_at) AS dt,
  ROUND(AVG((package_size / 1024.0 / 1024.0) / (total_cost / 1000.0)), 2) AS avg_bw
FROM dwd_cfdi_basic_fcl_uploadinfo
WHERE dt BETWEEN ? AND ?
  AND package_size > 10485760 AND total_cost > 0
GROUP BY DATE(create_at) ORDER BY dt
```

## 15. GET /dashboard/v1/do/top_vehicles — Top20 活跃车辆

主查询：

```sql
SELECT anonymous_id, car_type, project_name,
  COUNT(*) AS trigger_count,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS success_count,
  ROUND(SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END)*100.0/COUNT(*), 1) AS cfdi_rate
FROM dwd_cfdi_status_monitor_analysis
WHERE <车辆条件> AND anonymous_id IS NOT NULL AND anonymous_id != ''
GROUP BY anonymous_id, car_type, project_name
ORDER BY trigger_count DESC
LIMIT 20
```

补充查询（主失败原因 `vehicleFailReasons`，`:589`）：

```sql
SELECT anonymous_id,
  CASE
    WHEN fcl_status != 'success' AND fdr_status = 'success' AND fff_status = 'success'
      THEN CONCAT('FCL-', COALESCE(fcl_detail,''))
    WHEN fdr_status != 'success' AND fff_status = 'success'
      THEN CONCAT('FDR-', COALESCE(fdr_detail,''))
    ELSE CONCAT('FFF-', COALESCE(fff_detail,''))
  END AS fail_reason,
  COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <车辆条件>
  AND (fcl_status != 'success' OR fdr_status != 'success' OR fff_status != 'success')
  AND anonymous_id IN (?, ?, ...)
GROUP BY anonymous_id, fail_reason
ORDER BY anonymous_id, cnt DESC
```

## 16. GET /dashboard/v1/do/anomaly_vehicles — 异常车辆（成功率低于阈值）

入参同上 + `max_rate`。主查询与 top_vehicles 相同，但加：

```sql
... GROUP BY anonymous_id, car_type, project_name
HAVING cfdi_rate < ?
ORDER BY cfdi_rate ASC
LIMIT 100
```

失败原因补充查询同上。

## 17. GET /dashboard/v1/do/active_trend — 活跃车辆趋势

```sql
SELECT dt, COUNT(DISTINCT anonymous_id) AS active_count
FROM dwd_cfdi_status_monitor_analysis
WHERE <车辆条件>
GROUP BY dt ORDER BY dt
```

## 18. GET /dashboard/v1/do/funnel — DO 漏斗（节点统计 + 失败原因 Top10）

并发 4 条 SQL，公共 base：`FROM ads_do_cfdi_daily WHERE <公共条件> AND event_name != 'forever_log'`

```sql
-- ① 节点统计
SELECT
  SUM(cnt) AS fff_total,
  SUM(CASE WHEN fff_status='success' OR fdr_status='success' OR fcl_status='success' THEN cnt ELSE 0 END) AS fff_allow,
  SUM(CASE WHEN fdr_status='success' OR fcl_status='success' THEN cnt ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fdr_status='discard' THEN cnt ELSE 0 END) AS fdr_fail,
  SUM(CASE WHEN fcl_status='success' THEN cnt ELSE 0 END) AS fcl_success,
  SUM(CASE WHEN fcl_status='discard' THEN cnt ELSE 0 END) AS fcl_fail
FROM ads_do_cfdi_daily WHERE <公共条件> AND event_name != 'forever_log'

-- ② FFF 失败原因
SELECT fff_detail_tag AS name, SUM(cnt) AS cnt <base> AND fff_status='discard'
GROUP BY fff_detail_tag ORDER BY cnt DESC LIMIT 10

-- ③ FDR 失败原因
SELECT fdr_detail_tag AS name, SUM(cnt) AS cnt <base> AND fdr_status='discard'
GROUP BY fdr_detail_tag ORDER BY cnt DESC LIMIT 10

-- ④ FCL 失败原因
SELECT fcl_detail_tag AS name, SUM(cnt) AS cnt <base> AND fcl_status='discard'
GROUP BY fcl_detail_tag ORDER BY cnt DESC LIMIT 10
```

---

# 二、FO 模块（`/dashboard/v1/*`、`/dashboard/v1/fo/*`）

## 1. GET /dashboard/v1/dimensions — 下拉维度（FO/DO 共用，60 分钟缓存）

并发 4 条 DISTINCT 查询（近 7 天）：

```sql
SELECT DISTINCT filter_name AS val FROM dwd_cfdi_basic_fff_running
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) ORDER BY val;

SELECT DISTINCT project_name AS val FROM dwd_cfdi_basic_fff_running
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) AND project_name IS NOT NULL ORDER BY val;

SELECT DISTINCT event_name AS val FROM dwd_cfdi_basic_fff_trigger
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) ORDER BY val;

SELECT DISTINCT car_type AS val FROM dwd_cfdi_basic_fff_running
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) AND car_type IS NOT NULL ORDER BY val;
```

> 缓存策略：逻辑过期 + singleflight 冷启动防击穿（`internal/data/dashboard_fo.go:1212`）。

## 2. GET /dashboard/v1/diag/funnel — 数采全链路分析（FO/DO 共用）

与 DO funnel 结构相同（并发 4 条 SQL），但阶段判定口径更严格（含 `fcl_status = ''` 表示未到 FCL）：

```sql
SELECT
  SUM(cnt) AS fff_total,
  SUM(CASE WHEN fff_status!='discard' THEN cnt ELSE 0 END) AS fff_allow,
  SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') THEN cnt ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fff_status != 'discard' AND fdr_status != 'success' AND fcl_status = '' THEN cnt ELSE 0 END) AS fdr_fail,
  SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status != 'discard' THEN cnt ELSE 0 END) AS fcl_success,
  SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' THEN cnt ELSE 0 END) AS fcl_fail
FROM ads_do_cfdi_daily WHERE <公共条件> AND event_name != 'forever_log'
-- + 三条失败原因 Top10（fff/fdr/fcl_detail_tag，带阶段前置条件）
```

## 3. GET /dashboard/v1/fo/detail/running — 筛选器运行明细（分页）

并发 COUNT + 数据两条 SQL，表 `dwd_cfdi_basic_fff_running`：

```sql
SELECT COUNT(*) FROM dwd_cfdi_basic_fff_running WHERE <条件>;

SELECT dt, filter_name, anonymous_id, timestamp_utc, create_at, collect_type,
  sw_version, project_name, car_type, vehicle_source, switch_on, version,
  on_autopilot, function_mode, status, fdi_project_name, project_car_type, vehicle_source_cn
FROM dwd_cfdi_basic_fff_running WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

条件：`dt`（默认当天）+ 可选 `filter_name / project_name / car_type`。

## 4. GET /dashboard/v1/fo/detail/trigger — 筛选器触发明细（分页）

表 `dwd_cfdi_basic_fff_trigger`，结构同上：

```sql
SELECT COUNT(*) FROM dwd_cfdi_basic_fff_trigger WHERE <条件>;

SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at, trigger_time, utc_diff_us,
  `before`, `after`, filter_name, trigger_type, collect_type, status, on_autopilot, function_mode,
  sw_version, project_name, car_type, vehicle_source, bj02_lat, bj02_lon, road_type,
  fdi_project_name, project_car_type, vehicle_source_cn, tags, detail
FROM dwd_cfdi_basic_fff_trigger WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

> 注：`before`/`after` 是 SQL 保留字，需反引号转义。

## 5. GET /dashboard/v1/fo/detail/close — 筛选器关闭明细（分页）

表 `dwd_cfdi_basic_fff_close`：

```sql
SELECT dt, filter_name, version, reason, anonymous_id, create_at,
  sw_version, timestamp_utc, project_name, car_type, vehicle_source,
  fdi_project_name, project_car_type, vehicle_source_cn
FROM dwd_cfdi_basic_fff_close WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

## 6. GET /dashboard/v1/fo/detail/fdr — FDR 落盘明细（分页）

表 `dwd_basic_fdr_trigger`：

```sql
SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at, sw_version, dse,
  td_mb, tm_mb, trigger_timestamp, begin_timestamp_uts, end_timestamp_uts, dump_timestamp,
  status, detail, time_cost_ms, type, project_name, car_type, vehicle_source,
  fdi_project_name, project_car_type, vehicle_source_cn
FROM dwd_basic_fdr_trigger WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

## 7. GET /dashboard/v1/fo/detail/fcl — FCL 上传明细（分页）

表 `dwd_cfdi_basic_fcl_trigger`：

```sql
SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at,
  status, detail, complete_percent, local_file, upload_fail_times, prefix_stitch,
  trigger_source, sw_version, project_name, car_type, vehicle_source,
  fdi_project_name, project_car_type, vehicle_source_cn
FROM dwd_cfdi_basic_fcl_trigger WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

## 8. GET /dashboard/v1/fo/detail/uuid — 全链路明细（分页）

表 `dwd_cfdi_status_monitor_analysis`，支持 `only_fail` / `stage_filter` 过滤：

```sql
SELECT dt, anonymous_id, event_name, uuid, create_at, filter_name,
  fff_sw_version, fdr_sw_version, fcl_sw_version, trigger_type, collect_type,
  fff_updated_at, fdr_updated_at, fcl_updated_at,
  fff_status, fdr_status, fcl_status, fff_detail, fdr_detail, fcl_detail,
  begin_timestamp_uts, dump_timestamp, end_timestamp_uts,
  md5, bag_name, complete_percent, project_name, car_type, vehicle_source,
  project_car_type, dse
FROM dwd_cfdi_status_monitor_analysis WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

附加条件（`buildUuidDetailWhere`，`:828`）：

- `only_fail=1` → `fcl_status != 'success'`
- `stage_filter` 取值：`fff_success / fff_discard / fdr_success / fdr_discard / fcl_success / fcl_discard` → 对应 `xxx_status = 'success'|'discard'`

## 9. GET /dashboard/v1/fo/diag/stage_trend — 三阶段触发趋势（堆叠柱图）

单条大 SQL（表 `ads_do_cfdi_daily`），按 dt 聚合三阶段各 26 个明细指标：

```sql
SELECT dt,
  -- FFF 阶段：success / cooldown / drm_quota / acquire_data / trigger_maximum / bag_invalid
  --          / event_not_recognized / tls_error / quota_exceeded / event_in_blacklist / other
  SUM(CASE WHEN fff_status != 'discard' THEN cnt ELSE 0 END) AS fff_success,
  SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='cooldown' THEN cnt ELSE 0 END) AS fff_cooldown,
  ... （其余 9 个 fff 明细，other = 排除已知 tag）
  -- FDR 阶段：success / memory / disk / bag_invalid / bag_dir_missing / event_not_recognized / unauthorized / other
  --   前置条件：fff_status != 'discard' AND fdr_status != 'success' AND fcl_status = ''
  -- FCL 阶段：success / quota_exceeded / reach_upload_limit / event_in_blacklist / geofence_error
  --          / tls_error / bag_missing / upload_error / network_error / other
  --   前置条件：fff_status != 'discard' AND (fdr_status='success' OR fcl_status != '') AND fcl_status='discard'
FROM ads_do_cfdi_daily
WHERE <公共条件> AND event_name != 'forever_log'
GROUP BY dt ORDER BY dt ASC
```

> 完整 26 列 CASE WHEN 见 `internal/data/dashboard_fo.go:1122-1159`。

## 10. GET /dashboard/v1/fo/diag/close_reason — 算子关闭原因分布

表 `dwd_cfdi_basic_fff_close`，按 reason 关键字归类：

```sql
SELECT
  CASE
    WHEN INSTR(reason, 'out of memory')    > 0 THEN 'out of memory'
    WHEN INSTR(reason, 'not enough memory') > 0 THEN 'not enough memory'
    WHEN INSTR(reason, 'close operator')    > 0 THEN 'close operator for crash'
    WHEN INSTR(reason, 'with error')        > 0 THEN 'with error'
    ELSE '其他'
  END AS category,
  COUNT(*) AS cnt
FROM dwd_cfdi_basic_fff_close
WHERE <条件>
GROUP BY category
ORDER BY cnt DESC
```

## 11. GET /dashboard/v1/fo/running/trend — 算子活跃车辆趋势

入参必填：`filter_name, start_dt, end_dt`；选填：`project_name`。

```sql
SELECT dt, COUNT(DISTINCT anonymous_id) AS vehicle_count
FROM dwd_cfdi_basic_fff_running
WHERE switch_on = 1 AND filter_name = ? AND dt BETWEEN ? AND ?
  AND project_name = ?   -- 可选
GROUP BY dt ORDER BY dt
```

---

## 实现要点备忘

- 所有查询走 `dorisQuery(ctx)`（`base_repo.go:43`），带默认 30s 超时（可配 `doris.query_timeout`）。
- 分页接口均为 **COUNT + 数据双 SQL 并发**（`sync.WaitGroup`）。
- 聚合接口多为**多 SQL 并发**（overview 2 条、funnel 4 条、dimensions 4 条）。
- `forever_log` 事件在 DO 统计类接口中统一排除（注意：Doris 字符串比较大小写敏感，数据实际为小写 `forever_log`，历史上用大写 `Forever_log` 过滤是死条件）。
- 日期不传时：DO 聚合接口默认近 7 天（`dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)`）；FO 明细接口默认当天（`dt = CURDATE()`）。
