# FO / DO Dashboard 明细表 SQL 改聚合表 SQL

本文按“明细表查询 SQL（聚合表改造前快照）”逐个接口给出当前可用聚合表写法。

## 重要口径限制

1. 聚合表同时写入真实 `event_name` 和 `__ALL__` 汇总行。未传 `event_names` 时必须查 `event_name='__ALL__'`；传了 `event_names` 时查 `event_name IN (...)`。不能不加事件条件，否则会重复计算。
2. 当前多数汇总表把 `filter`、`status`、`reason` 拆成不同 `summary_grain`。因此“按失败原因统计，同时按 `filter_name` 过滤”不能用当前聚合表等价查询，需要新增组合粒度。
3. 旧 SQL 的 `COUNT(DISTINCT anonymous_id)` 是在整个日期范围内去重；当前日汇总表的 `vehicle_count` 是每一天、每个低维分组内去重。跨多天 `SUM(vehicle_count)` 是车辆日口径，不是全周期唯一车辆数。单日查询等价性最高。
4. 明细分页接口和单车分析接口需要 `uuid`、`anonymous_id`、`timestamp_utc`、原始 `detail` 等行级字段，当前日汇总表不能替换。

## 通用条件模板

### 事件条件

```sql
-- 未传 event_names
AND event_name = '__ALL__'

-- 已传 event_names
AND event_name IN (:event_names)
```

### 项目 / 车型条件

```sql
AND (:project_name = '' OR project_name = :project_name)
AND (:car_types_empty = 1 OR car_type IN (:car_types))
```

### status monitor 普通指标 grain

用于 `event_count`、阶段成功/失败数、项目车型分布等普通指标。未传 `filter_name` 用 `overview`，传了 `filter_name` 用 `filter`。

```sql
AND (
  (:filter_name = '' AND summary_grain = 'overview')
  OR (:filter_name <> '' AND summary_grain = 'filter' AND filter_name = :filter_name)
)
```

## 一、DO 模块

### 1. `GET /do/overview` 事件横向对比

旧 SQL 来源：`dwd_cfdi_status_monitor_analysis`。

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

```sql
SELECT
  event_name,
  SUM(vehicle_count) AS vehicle_count,
  SUM(event_count) AS trigger_count,
  SUM(fff_success_count) AS fff_count,
  IFNULL(SUM(fff_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS fff_rate,
  SUM(fdr_success_count) AS fdr_count,
  IFNULL(SUM(fdr_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS fdr_rate,
  SUM(fcl_success_count) AS fcl_count,
  IFNULL(SUM(fcl_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS fcl_rate,
  IFNULL(SUM(overall_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS cfdi_rate
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND event_name <> '__ALL__'
  AND event_name <> 'forever_log'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND (
    (:filter_name = '' AND summary_grain = 'overview')
    OR (:filter_name <> '' AND summary_grain = 'filter' AND filter_name = :filter_name)
  )
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY trigger_count DESC;
```

说明：跨多天时 `vehicle_count` 是每日去重后累加，不是旧 SQL 的全周期去重。

### 2. `GET /do/trend` 数据总览趋势

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

```sql
SELECT
  dt,
  SUM(event_count) AS total,
  SUM(fcl_success_count) AS success,
  IFNULL(SUM(fcl_success_count) / NULLIF(SUM(event_count), 0) * 100, 0) AS success_rate
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  /* event_name condition */
  AND (
    (:filter_name = '' AND summary_grain = 'overview')
    OR (:filter_name <> '' AND summary_grain = 'filter' AND filter_name = :filter_name)
  )
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt ASC;
```

### 3. `GET /do/fail_reason` 失败原因分析

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

限制：当前 `stage_reason` 粒度没有 `filter_name`，所以传 `filter_name` 时不能等价查询。需要新增 `filter_stage_reason` 粒度才能完整替代旧 SQL。

```sql
SELECT stage, detail, SUM(cnt) AS cnt
FROM (
  SELECT
    'FFF' AS stage,
    fff_detail_tag AS detail,
    SUM(fff_failed_count) AS cnt
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN :start_dt AND :end_dt
    AND summary_grain = 'stage_reason'
    AND :filter_name = ''
    AND fff_detail_tag NOT IN ('__ALL__', '')
    /* event_name condition */
    AND (:project_name = '' OR project_name = :project_name)
    AND (:car_types_empty = 1 OR car_type IN (:car_types))
  GROUP BY fff_detail_tag

  UNION ALL

  SELECT
    'FDR' AS stage,
    fdr_detail_tag AS detail,
    SUM(fdr_failed_count) AS cnt
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN :start_dt AND :end_dt
    AND summary_grain = 'stage_reason'
    AND :filter_name = ''
    AND fdr_detail_tag NOT IN ('__ALL__', '')
    /* event_name condition */
    AND (:project_name = '' OR project_name = :project_name)
    AND (:car_types_empty = 1 OR car_type IN (:car_types))
  GROUP BY fdr_detail_tag

  UNION ALL

  SELECT
    'FCL' AS stage,
    fcl_detail_tag AS detail,
    SUM(fcl_failed_count) AS cnt
  FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
  WHERE dt BETWEEN :start_dt AND :end_dt
    AND summary_grain = 'stage_reason'
    AND :filter_name = ''
    AND fcl_detail_tag NOT IN ('__ALL__', '')
    /* event_name condition */
    AND (:project_name = '' OR project_name = :project_name)
    AND (:car_types_empty = 1 OR car_type IN (:car_types))
  GROUP BY fcl_detail_tag
) t
GROUP BY stage, detail
ORDER BY stage, cnt DESC;
```

### 4. `GET /do/cool_top` 冷却 Top20 筛选器

旧 SQL 要求：`fff_detail='check_is_no_need_cooldown'` 并按 `filter_name` 排名。

当前聚合表不能等价替换，因为 `dwd_cfdi_basic_fff_trigger_daily_summary`：

- `reason` 粒度有 `detail_tag='cooldown'`，但 `filter_name='__ALL__'`。
- `filter` 粒度有 `filter_name`，但 `detail_tag='__ALL__'`。

需要新增 `filter_reason` 粒度后 SQL 才能写成：

```sql
SELECT
  filter_name,
  SUM(failed_count) AS cnt
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter_reason'
  AND detail_tag = 'cooldown'
  AND filter_name <> '__ALL__'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 20;
```

### 5. `GET /do/trigger_rank` 触发频次排行 Top10

旧 SQL 按 `filter_name` 排名，不是按 `event_name` 排名。

聚合表：`fdi.dwd_cfdi_basic_fff_trigger_daily_summary`。

```sql
SELECT
  filter_name,
  SUM(event_count) AS cnt
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter'
  AND filter_name <> '__ALL__'
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 10;
```

### 6. `GET /do/sw_version` 软件版本分布

旧 SQL 使用 `fff_sw_version`。

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

```sql
SELECT
  fff_sw_version AS sw_version,
  SUM(event_count) AS cnt
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  /* event_name condition */
  AND (
    (:filter_name = '' AND summary_grain = 'overview')
    OR (:filter_name <> '' AND summary_grain = 'filter' AND filter_name = :filter_name)
  )
  AND fff_sw_version <> ''
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY fff_sw_version
ORDER BY cnt DESC
LIMIT 20;
```

### 7. `GET /do/project_car` 项目 x 车型分布

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

```sql
SELECT
  project_name,
  car_type,
  SUM(event_count) AS cnt
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  /* event_name condition */
  AND (
    (:filter_name = '' AND summary_grain = 'overview')
    OR (:filter_name <> '' AND summary_grain = 'filter' AND filter_name = :filter_name)
  )
  AND project_name <> ''
  AND car_type <> ''
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY project_name, car_type
ORDER BY project_name, car_type;
```

### 8. `GET /do/mem_top` FDR 内存不足 Top20

推荐聚合表：`fdi.dwd_basic_fdr_trigger_daily_summary`。

```sql
SELECT
  event_name,
  SUM(failed_count) AS cnt
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag IN ('full_gc', 'mem_pool_water_line')
  AND event_name <> '__ALL__'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY cnt DESC
LIMIT 20;
```

说明：旧 SQL 带 `fff_status='success'` 前置条件。FDR 专项汇总来源是 FDR 触发表，一般可视为进入 FDR 后的记录；如果必须严格复刻状态链路前置条件，需要在 status monitor 新增 `stage_status_reason` 或 `filter_stage_reason` 组合粒度。

### 9. `GET /do/disk_top` FDR 磁盘不足 Top20

聚合表：`fdi.dwd_basic_fdr_trigger_daily_summary`。

```sql
SELECT
  event_name,
  SUM(failed_count) AS cnt
FROM fdi.dwd_basic_fdr_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag IN ('disk_overrun', 'max_files_exceeded')
  AND event_name <> '__ALL__'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY cnt DESC
LIMIT 20;
```

### 10. `GET /do/close_top` 关闭次数 Top 筛选器

聚合表：`fdi.dwd_cfdi_basic_fff_close_daily_summary`。

```sql
SELECT
  filter_name,
  SUM(close_count) AS cnt
FROM fdi.dwd_cfdi_basic_fff_close_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter'
  AND filter_name <> '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 20;
```

### 11. `GET /do/quota_top` FCL Quota 超限 Top20

聚合表：`fdi.dwd_cfdi_basic_fcl_trigger_daily_summary`。

```sql
SELECT
  event_name,
  SUM(failed_count) AS cnt
FROM fdi.dwd_cfdi_basic_fcl_trigger_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND detail_tag = 'quota_exceeded'
  AND event_name <> '__ALL__'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY event_name
ORDER BY cnt DESC
LIMIT 20;
```

### 12. `GET /do/project_event` 项目触发回流事件总数

旧 SQL 是 `COUNT(DISTINCT event_name)` 且要求 `fcl_status='success'`。

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

```sql
SELECT
  project_name,
  COUNT(DISTINCT CASE WHEN fcl_success_count > 0 THEN event_name END) AS event_count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name <> '__ALL__'
  AND event_name <> 'forever_log'
  AND (:event_names_empty = 1 OR event_name IN (:event_names))
  AND project_name <> ''
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY project_name
ORDER BY event_count DESC;
```

说明：如果传 `filter_name`，当前 `overview` 无法过滤；用 `filter` 粒度又可能影响 `COUNT(DISTINCT event_name)` 口径。严格支持需要新增项目/事件/filter 组合口径验证。

### 13. `GET /do/net_speed` 各车型平均上传带宽

聚合表：`fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary`。

```sql
SELECT
  dt,
  car_type,
  ROUND(IFNULL(SUM(upload_bandwidth_sum) / NULLIF(SUM(upload_bandwidth_count), 0), 0), 2) AS avg_bw
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  AND car_type <> ''
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, car_type
ORDER BY dt, car_type;
```

### 14. `GET /do/fcl_bw` FCL 整体平均上传带宽

聚合表：`fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary`。

```sql
SELECT
  dt,
  ROUND(IFNULL(SUM(upload_bandwidth_sum) / NULLIF(SUM(upload_bandwidth_count), 0), 0), 2) AS avg_bw
FROM fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND event_name = '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt;
```

### 15. `GET /do/top_vehicles` Top20 活跃车辆

不能用当前日聚合表等价替换。原因：

- 返回字段包含 `anonymous_id`。
- 当前 `ads_cfdi_vehicle_daily_summary` 是低维汇总，不保留单车维度。
- 还需要每车主失败原因，当前聚合表没有单车失败原因分布。

需要新增车辆粒度 ADS，例如：

```text
fdi.ads_cfdi_vehicle_event_daily_summary(
  dt, event_name, anonymous_id, car_type, project_name,
  trigger_count, success_count, failed_count, main_reason, ...
)
```

### 16. `GET /do/anomaly_vehicles` 异常车辆

不能用当前日聚合表等价替换。原因同 `top_vehicles`，需要单车维度和单车成功率。

### 17. `GET /do/active_trend` 活跃车辆趋势

聚合表：`fdi.ads_cfdi_vehicle_daily_summary`。

```sql
SELECT
  dt,
  SUM(vehicle_count) AS active_count
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt;
```

### 18. `GET /do/funnel` DO 漏斗

节点统计聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

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
  AND event_name <> 'forever_log'
  /* event_name condition */
  AND (
    (:filter_name = '' AND summary_grain = 'overview')
    OR (:filter_name <> '' AND summary_grain = 'filter' AND filter_name = :filter_name)
  )
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types));
```

失败原因 Top10 与 `GET /do/fail_reason` 相同，使用 `stage_reason` 粒度；传 `filter_name` 时当前聚合表不能等价支持。

## 二、FO 模块

### 1. `GET /dashboard/v1/dimensions` 下拉维度

```sql
SELECT DISTINCT filter_name AS val
FROM fdi.dwd_cfdi_basic_fff_running_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND summary_grain = 'filter'
  AND filter_name <> '__ALL__'
ORDER BY val;
```

```sql
SELECT DISTINCT project_name AS val
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND project_name <> ''
ORDER BY val;
```

```sql
SELECT DISTINCT event_name AS val
FROM fdi.dwd_cfdi_basic_fff_trigger_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND event_name <> '__ALL__'
ORDER BY val;
```

```sql
SELECT DISTINCT car_type AS val
FROM fdi.ads_cfdi_vehicle_daily_summary
WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND car_type <> ''
ORDER BY val;
```

### 2. `GET /dashboard/v1/diag/funnel` 数采全链路分析

同 `GET /do/funnel`。

### 3. `GET /dashboard/v1/fo/detail/running`

不能用当前聚合表替换。该接口返回运行明细行，依赖 `anonymous_id`、`timestamp_utc`、`create_at`、原始 `status` 等字段。

### 4. `GET /dashboard/v1/fo/detail/trigger`

不能用当前聚合表替换。该接口返回触发明细行，依赖 `uuid`、`anonymous_id`、`trigger_time`、`before`、`after`、原始 `detail` 等字段。

### 5. `GET /dashboard/v1/fo/detail/close`

不能用当前聚合表替换。该接口返回关闭明细行，依赖 `anonymous_id`、`create_at`、原始 `reason` 等字段。

### 6. `GET /dashboard/v1/fo/detail/fdr`

不能用当前聚合表替换。该接口返回 FDR 明细行，依赖 `uuid`、`anonymous_id`、原始 `td_mb`、`tm_mb`、时间戳等字段。

### 7. `GET /dashboard/v1/fo/detail/fcl`

不能用当前聚合表替换。该接口返回 FCL 明细行，依赖 `uuid`、`anonymous_id`、`local_file`、`complete_percent`、原始 `detail` 等字段。

### 8. `GET /dashboard/v1/fo/detail/uuid`

不能用当前聚合表替换。该接口返回全链路 UUID 明细，必须查 `dwd_cfdi_status_monitor_analysis`。

### 9. `GET /dashboard/v1/fo/diag/stage_trend`

聚合表：`fdi.dwd_cfdi_status_monitor_analysis_daily_summary`。

限制：失败原因序列来自 `stage_reason` 粒度，该粒度没有 `filter_name`。传 `filter_name` 时不能等价替换旧 SQL。

FFF 序列：

```sql
SELECT
  dt,
  'success' AS name,
  SUM(fff_success_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND :filter_name = ''
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
  AND :filter_name = ''
  AND fff_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, fff_detail_tag
ORDER BY dt, name;
```

FDR 序列：

```sql
SELECT
  dt,
  'success' AS name,
  SUM(fdr_success_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND :filter_name = ''
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
  AND :filter_name = ''
  AND fdr_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, fdr_detail_tag
ORDER BY dt, name;
```

FCL 序列：

```sql
SELECT
  dt,
  'success' AS name,
  SUM(fcl_success_count) AS count
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'overview'
  AND :filter_name = ''
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
  AND :filter_name = ''
  AND fcl_detail_tag NOT IN ('__ALL__', '')
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt, fcl_detail_tag
ORDER BY dt, name;
```

如果只需要传 `filter_name` 时的阶段成功/失败总量，不拆失败原因，可以用 `filter` 粒度：

```sql
SELECT
  dt,
  SUM(fff_success_count) AS fff_success,
  SUM(fff_failed_count) AS fff_failed,
  SUM(fdr_success_count) AS fdr_success,
  SUM(fdr_failed_count) AS fdr_failed,
  SUM(fcl_success_count) AS fcl_success,
  SUM(fcl_failed_count) AS fcl_failed
FROM fdi.dwd_cfdi_status_monitor_analysis_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'filter'
  AND filter_name = :filter_name
  /* event_name condition */
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY dt
ORDER BY dt;
```

### 10. `GET /dashboard/v1/fo/diag/close_reason`

聚合表：`fdi.dwd_cfdi_basic_fff_close_daily_summary`。

限制：当前 `reason` 粒度没有 `filter_name`，传 `filter_name` 时不能等价替换旧 SQL。需要新增 `filter_reason` 粒度。

```sql
SELECT
  close_reason_tag AS category,
  SUM(close_count) AS cnt
FROM fdi.dwd_cfdi_basic_fff_close_daily_summary
WHERE dt BETWEEN :start_dt AND :end_dt
  AND summary_grain = 'reason'
  AND :filter_name = ''
  AND close_reason_tag <> '__ALL__'
  AND (:project_name = '' OR project_name = :project_name)
  AND (:car_types_empty = 1 OR car_type IN (:car_types))
GROUP BY close_reason_tag
ORDER BY cnt DESC;
```

## 需要新增组合聚合粒度的接口

| 接口 | 当前缺口 | 建议新增 grain |
|---|---|---|
| `/do/fail_reason` 带 `filter_name` | `stage_reason` 没有 `filter_name` | `filter_stage_reason` |
| `/do/cool_top` | 需要 `filter_name + detail_tag` | `filter_reason` |
| `/do/funnel` 失败原因带 `filter_name` | `stage_reason` 没有 `filter_name` | `filter_stage_reason` |
| `/fo/diag/stage_trend` 带 `filter_name` 且要拆失败原因 | `stage_reason` 没有 `filter_name` | `filter_stage_reason` |
| `/fo/diag/close_reason` 带 `filter_name` | `reason` 没有 `filter_name` | `filter_reason` |
| `/do/top_vehicles` | 无 `anonymous_id` 粒度 | 车辆日 ADS |
| `/do/anomaly_vehicles` | 无 `anonymous_id` 粒度 | 车辆日 ADS |
