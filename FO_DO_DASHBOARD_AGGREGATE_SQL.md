# FO / DO Dashboard 聚合表查询 SQL

本文只列出可以直接改为查询聚合表的接口。明细分页和车辆行级分析接口依赖 `uuid`、`anonymous_id`、`timestamp_utc`、原始 `detail` 等行级字段，不能用当前日汇总表完整替换。

## 通用约定

### 日期条件

```sql
dt BETWEEN :start_dt AND :end_dt
```

### 事件条件

聚合表通常同时写入真实 `event_name` 和 `__ALL__` 汇总行。没有传 `event_names` 时必须只查 `__ALL__`，否则会把真实事件行和 `__ALL__` 行重复计算。

```sql
-- 未传 event_names
AND event_name = '__ALL__'

-- 已传 event_names
AND event_name IN (:event_names)
```

### 维度条件

```sql
AND (:project_name = '' OR project_name = :project_name)
AND (:car_types_empty = 1 OR car_type IN (:car_types))
```

如接口支持 `filter_name`：

```sql
AND (:filter_name = '' OR filter_name = :filter_name)
```

## 可替换接口清单

| 接口 | 聚合表 |
|---|---|
| `GET /dashboard/v1/dimensions` | 多张汇总表 |
| `GET /dashboard/v1/diag/funnel` | `fdi.dwd_cfdi_status_monitor_analysis_daily_summary` |
| `GET /dashboard/v1/fo/diag/stage_trend` | `fdi.dwd_cfdi_status_monitor_analysis_daily_summary` |
| `GET /dashboard/v1/fo/diag/close_reason` | `fdi.dwd_cfdi_basic_fff_close_daily_summary` |
| `GET /dashboard/v1/fo/running/trend` | `fdi.dwd_cfdi_basic_fff_running_daily_summary` |
| `GET /dashboard/v1/do/overview` | `fdi.ads_cfdi_vehicle_daily_summary` |
| `GET /dashboard/v1/do/trend` | `fdi.dwd_cfdi_status_monitor_analysis_daily_summary` |
| `GET /dashboard/v1/do/fail_reason` | `fdi.dwd_cfdi_status_monitor_analysis_daily_summary` |
| `GET /dashboard/v1/do/cool_top` | `fdi.dwd_cfdi_basic_fff_trigger_daily_summary` |
| `GET /dashboard/v1/do/sw_version` | `fdi.dwd_cfdi_basic_fff_trigger_daily_summary` |
| `GET /dashboard/v1/do/trigger_rank` | `fdi.dwd_cfdi_basic_fff_trigger_daily_summary` |
| `GET /dashboard/v1/do/project_car` | `fdi.dwd_cfdi_basic_fff_trigger_daily_summary` |
| `GET /dashboard/v1/do/active_trend` | `fdi.ads_cfdi_vehicle_daily_summary` |
| `GET /dashboard/v1/do/net_speed` | `fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary` |
| `GET /dashboard/v1/do/fcl_bw` | `fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary` |
| `GET /dashboard/v1/do/quota_top` | `fdi.dwd_cfdi_basic_fcl_trigger_daily_summary` |
| `GET /dashboard/v1/do/project_event` | `fdi.dwd_cfdi_basic_fff_trigger_daily_summary` |
| `GET /dashboard/v1/do/mem_top` | `fdi.dwd_basic_fdr_trigger_daily_summary` |
| `GET /dashboard/v1/do/disk_top` | `fdi.dwd_basic_fdr_trigger_daily_summary` |
| `GET /dashboard/v1/do/close_top` | `fdi.dwd_cfdi_basic_fff_close_daily_summary` |
| `GET /dashboard/v1/do/funnel` | `fdi.dwd_cfdi_status_monitor_analysis_daily_summary` |

## 公共接口

### `GET /dashboard/v1/dimensions`

```sql
SELECT DISTINCT filter_name
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND summary_grain = 'filter'
  AND filter_name <> '__ALL__'
ORDER BY filter_name;
```

```sql
SELECT DISTINCT event_name
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND event_name <> '__ALL__'
ORDER BY event_name;
```

```sql
SELECT DISTINCT project_name
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND project_name <> ''
ORDER BY project_name;
```

```sql
SELECT DISTINCT car_type
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND car_type <> ''
ORDER BY car_type;
```

### `GET /dashboard/v1/diag/funnel`

与 `GET /dashboard/v1/do/funnel` 共用 SQL。

```sql
SELECT
  SUM(event_count) AS fff_total,
  SUM(fff_success_count) AS fff_allow,
  SUM(fdr_success_count) AS fdr_success,
  SUM(fdr_failed_count) AS fdr_fail,
  SUM(fcl_success_count) AS fcl_success,
  SUM(fcl_failed_count) AS fcl_fail,
  IFNULL(SUM(overall_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS cfdi_rate
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name: 未传 event_names 用 = '__ALL__'; 已传用 IN (:event_names) */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types));
```

FFF 失败原因 Top10：

```sql
SELECT
  fff_detail_tag AS name,
  SUM(fff_failed_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'stage_reason'
  AND fff_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY fff_detail_tag
ORDER BY count DESC
LIMIT 10;
```

FDR 失败原因 Top10：

```sql
SELECT
  fdr_detail_tag AS name,
  SUM(fdr_failed_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'stage_reason'
  AND fdr_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY fdr_detail_tag
ORDER BY count DESC
LIMIT 10;
```

FCL 失败原因 Top10：

```sql
SELECT
  fcl_detail_tag AS name,
  SUM(fcl_failed_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'stage_reason'
  AND fcl_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY fcl_detail_tag
ORDER BY count DESC
LIMIT 10;
```

## FO 接口

### `GET /dashboard/v1/fo/diag/stage_trend`

成功量用 `overview`，失败原因拆分用 `stage_reason`。

```sql
SELECT
  dt,
  'success' AS name,
  SUM(fff_success_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt

UNION ALL

SELECT
  dt,
  fff_detail_tag AS name,
  SUM(fff_failed_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'stage_reason'
  AND fff_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, fff_detail_tag
ORDER BY dt, name;
```

FDR：

```sql
SELECT
  dt,
  'success' AS name,
  SUM(fdr_success_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt

UNION ALL

SELECT
  dt,
  fdr_detail_tag AS name,
  SUM(fdr_failed_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'stage_reason'
  AND fdr_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, fdr_detail_tag
ORDER BY dt, name;
```

FCL：

```sql
SELECT
  dt,
  'success' AS name,
  SUM(fcl_success_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt

UNION ALL

SELECT
  dt,
  fcl_detail_tag AS name,
  SUM(fcl_failed_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'stage_reason'
  AND fcl_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, fcl_detail_tag
ORDER BY dt, name;
```

### `GET /dashboard/v1/fo/diag/close_reason`

```sql
SELECT
  close_reason_tag AS name,
  SUM(close_count) AS value
FROM fdi.dwd_cfdi_basic_fff_close_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND close_reason_tag <> '__ALL__'
  AND (:filter_name = '' OR filter_name = :filter_name)
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY close_reason_tag
ORDER BY value DESC;
```

### `GET /dashboard/v1/fo/running/trend`

该接口要求显式传 `filter_name`，应查 running 汇总表的 `filter` 粒度。

```sql
SELECT
  dt,
  SUM(vehicle_count) AS count
FROM fdi.dwd_cfdi_basic_fff_running_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter'
  AND filter_name = :filter_name
  AND switch_on = 1
  AND (:project_name = '' OR project_name = :project_name)
GROUP BY dt
ORDER BY dt;
```

## DO 接口

### `GET /dashboard/v1/do/overview`

```sql
SELECT
  event_name,
  SUM(vehicle_count) AS vehicle_count,
  SUM(trigger_vehicle_count) AS trigger_count,
  IFNULL(SUM(overall_success_vehicle_count) / NULLIF(SUM(trigger_vehicle_count), 0) * 100, 0) AS cfdi_rate,
  SUM(fff_success_vehicle_count) AS fff_count,
  IFNULL(SUM(fff_success_vehicle_count) / NULLIF(SUM(trigger_vehicle_count), 0) * 100, 0) AS fff_rate,
  SUM(fdr_success_vehicle_count) AS fdr_count,
  IFNULL(SUM(fdr_success_vehicle_count) / NULLIF(SUM(trigger_vehicle_count), 0) * 100, 0) AS fdr_rate,
  SUM(fcl_success_vehicle_count) AS fcl_count,
  IFNULL(SUM(fcl_success_vehicle_count) / NULLIF(SUM(trigger_vehicle_count), 0) * 100, 0) AS fcl_rate
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND event_name <> '__ALL__'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY trigger_count DESC;
```

### `GET /dashboard/v1/do/trend`

```sql
SELECT
  dt,
  SUM(overall_success_count) AS success_count,
  IFNULL(SUM(overall_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS success_rate
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt;
```

### `GET /dashboard/v1/do/fail_reason`

```sql
SELECT name, SUM(value) AS value
FROM (
  SELECT CONCAT('FFF-', fff_detail_tag) AS name, SUM(fff_failed_count) AS value
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN :start_dt AND :end_dt
    AND summary_grain = 'stage_reason'
    AND fff_detail_tag NOT IN ('__ALL__', '')
    /* event_name condition */
    AND (:project_name = '' OR project_name = :project_name)
    AND (:car_types_empty = 1 OR car_type IN (:car_types))
  GROUP BY fff_detail_tag

  UNION ALL

  SELECT CONCAT('FDR-', fdr_detail_tag), SUM(fdr_failed_count)
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN :start_dt AND :end_dt
    AND summary_grain = 'stage_reason'
    AND fdr_detail_tag NOT IN ('__ALL__', '')
    /* event_name condition */
    AND (:project_name = '' OR project_name = :project_name)
    AND (:car_types_empty = 1 OR car_type IN (:car_types))
  GROUP BY fdr_detail_tag

  UNION ALL

  SELECT CONCAT('FCL-', fcl_detail_tag), SUM(fcl_failed_count)
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN :start_dt AND :end_dt
    AND summary_grain = 'stage_reason'
    AND fcl_detail_tag NOT IN ('__ALL__', '')
    /* event_name condition */
    AND (:project_name = '' OR project_name = :project_name)
    AND (:car_types_empty = 1 OR car_type IN (:car_types))
  GROUP BY fcl_detail_tag
) t
GROUP BY name
ORDER BY value DESC;
```

### `GET /dashboard/v1/do/cool_top`

```sql
SELECT
  filter_name,
  SUM(failed_count) AS count
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag = 'cooldown'
  AND filter_name <> '__ALL__'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY filter_name
ORDER BY count DESC
LIMIT 20;
```

### `GET /dashboard/v1/do/sw_version`

```sql
SELECT
  sw_version,
  SUM(event_count) AS count
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY sw_version
ORDER BY count DESC;
```

### `GET /dashboard/v1/do/trigger_rank`

```sql
SELECT
  event_name,
  SUM(event_count) AS count
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name <> '__ALL__'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY count DESC
LIMIT 10;
```

### `GET /dashboard/v1/do/project_car`

```sql
SELECT
  project_name,
  car_type,
  SUM(event_count) AS count
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND project_name <> ''
  AND car_type <> ''
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY project_name, car_type
ORDER BY count DESC;
```

### `GET /dashboard/v1/do/active_trend`

```sql
SELECT
  dt,
  SUM(vehicle_count) AS count
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt;
```

### `GET /dashboard/v1/do/net_speed`

```sql
SELECT
  dt,
  car_type,
  IFNULL(SUM(upload_bandwidth_sum) / NULLIF(SUM(upload_bandwidth_count), 0), 0) AS avg_upload_bandwidth
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, car_type
ORDER BY dt, car_type;
```

### `GET /dashboard/v1/do/fcl_bw`

```sql
SELECT
  dt,
  IFNULL(SUM(upload_bandwidth_sum) / NULLIF(SUM(upload_bandwidth_count), 0), 0) AS avg_upload_bandwidth
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt;
```

### `GET /dashboard/v1/do/quota_top`

```sql
SELECT
  event_name,
  SUM(failed_count) AS count
FROM fdi.dwd_cfdi_basic_fcl_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag = 'quota_exceeded'
  AND event_name <> '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY count DESC
LIMIT 20;
```

### `GET /dashboard/v1/do/project_event`

```sql
SELECT
  project_name,
  SUM(event_count) AS event_count
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  /* event_name condition */
  AND project_name <> ''
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY project_name
ORDER BY event_count DESC;
```

### `GET /dashboard/v1/do/mem_top`

```sql
SELECT
  event_name,
  SUM(failed_count) AS count
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag IN ('mem_pool_water_line', 'full_gc')
  AND event_name <> '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY count DESC
LIMIT 20;
```

### `GET /dashboard/v1/do/disk_top`

```sql
SELECT
  event_name,
  SUM(failed_count) AS count
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag = 'disk_overrun'
  AND event_name <> '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY count DESC
LIMIT 20;
```

### `GET /dashboard/v1/do/close_top`

```sql
SELECT
  filter_name,
  SUM(close_count) AS count
FROM fdi.dwd_cfdi_basic_fff_close_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter'
  AND filter_name <> '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY filter_name
ORDER BY count DESC
LIMIT 20;
```

### `GET /dashboard/v1/do/funnel`

同 `GET /dashboard/v1/diag/funnel`。

## 不能用当前聚合表完整替换的接口

| 接口 | 原因 |
|---|---|
| `GET /dashboard/v1/fo/detail/running` | 返回运行明细，需要行级字段 |
| `GET /dashboard/v1/fo/detail/trigger` | 返回触发明细，需要 `uuid`、`anonymous_id`、原始 `detail` |
| `GET /dashboard/v1/fo/detail/close` | 返回关闭明细，需要行级 `reason`、车辆和时间字段 |
| `GET /dashboard/v1/fo/detail/fdr` | 返回 FDR 明细，需要 TD/TM 原始值、时间戳、`uuid` |
| `GET /dashboard/v1/fo/detail/fcl` | 返回 FCL 明细，需要上传明细字段 |
| `GET /dashboard/v1/fo/detail/uuid` | 返回全链路 UUID 明细，必须查链路明细表 |
| `GET /dashboard/v1/do/top_vehicles` | 当前 ADS 不是车辆粒度，无法返回 `anonymous_id` |
| `GET /dashboard/v1/do/anomaly_vehicles` | 当前 ADS 不是车辆粒度，无法按单车成功率筛选 |
