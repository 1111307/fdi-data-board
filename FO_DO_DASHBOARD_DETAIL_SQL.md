# FO / DO Dashboard 明细表查询 SQL（聚合表改造前快照）

> **快照版本**：`4fea9a4`（即 `a1e2f6a^`，`a1e2f6a feat:FO页面读预聚合表` 的父提交）
>
> ⚠️ 关于提交点的说明：你给的 `6e3d362`（当前 HEAD）**已经引入聚合表** `ads_do_cfdi_daily`（由 `a1e2f6a`/`94309ba` 两个提交引入）。本文档取自**最后一个全部接口只查明细表（`dwd_*`）的提交 `4fea9a4`**。
>
> 数据源：Doris，GORM + MySQL 协议。当时代码带慢查询日志（`slowLog.Observe`），路由尚无 UM 鉴权中间件。
>
> ⚠️ **历史缺陷提醒**：本文档中的 `event_name != 'Forever_log'`（大写）是当时的**死条件**——Doris 字符串比较大小写敏感，而明细表数据实际为小写 `forever_log`，该排除从未生效。**复用本文 SQL 时必须改为小写 `forever_log`**（现网代码已于 2026-08-10 修复）。

## 涉及的明细表（全部 `dwd_*`，无任何聚合表）

| 表名 | 说明 |
|------|------|
| `dwd_cfdi_status_monitor_analysis` | 全链路状态明细（uuid 维度，FFF/FDR/FCL 三阶段 status + detail 原文） |
| `dwd_cfdi_basic_fff_running` | 筛选器运行明细 |
| `dwd_cfdi_basic_fff_trigger` | 筛选器触发明细 |
| `dwd_cfdi_basic_fff_close` | 筛选器关闭明细 |
| `dwd_basic_fdr_trigger` | FDR 落盘明细 |
| `dwd_cfdi_basic_fcl_trigger` | FCL 上传明细 |
| `dwd_cfdi_basic_fcl_uploadinfo` | FCL 上传带宽信息 |

## 公共 WHERE 条件（`buildDoCommonWhere` / `buildVehicleWhere`）

```sql
WHERE dt BETWEEN ? AND ?           -- 不传日期时：dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)（近7天）
  AND filter_name = ?              -- 可选
  AND event_name = ? / IN (...)    -- 可选，多选
  AND project_name = ?             -- 可选
  AND car_type   = ? / IN (...)    -- 可选，多选
```

> 关键区别：此快照**没有 `detail_tag` 规整列**，所有失败归因都靠 `fff_detail / fdr_detail / fcl_detail` **原文串匹配**（`=` / `IN` / `LIKE` / `INSTR`）。

---

# 一、DO 模块（`/dashboard/v1/do/*`，共 18 个接口）

## 1. GET /do/overview — 事件横向对比

单条 SQL（明细表一条搞定，后来的聚合版拆成了 ads + dwd 两条并发）：

```sql
SELECT event_name,
  COUNT(DISTINCT anonymous_id) AS vehicle_count,
  COUNT(*) AS trigger_count,
  SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_count,
  SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_count,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS fcl_count
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件> AND event_name != 'Forever_log'
GROUP BY event_name ORDER BY trigger_count DESC
```

## 2. GET /do/trend — 数据总览趋势

```sql
SELECT dt,
  COUNT(*) AS total,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS success
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件>
GROUP BY dt ORDER BY dt ASC
```

## 3. GET /do/fail_reason — 失败原因分析（三阶段归因）

三段 UNION ALL + 外层对 detail 原文做 LIKE 归一化（args 需重复 3 份）：

```sql
SELECT stage,
  CASE WHEN detail LIKE 'Bag invalid:%' THEN 'Bag invalid'
       WHEN detail LIKE 'tls%' THEN 'TLS error'
       WHEN detail LIKE 'event_name do not recognized%' THEN 'event_name'
       WHEN detail LIKE 'Dump bag dir missing%' THEN 'Dump bag dir missing'
       WHEN detail LIKE 'query cloud DISCARD, detail:Filter quota exceeded' THEN 'Filter quota exceeded'
       WHEN detail LIKE 'query cloud DISCARD, detail:EventName is in blacklist' THEN 'EventName is in blacklist'
       ELSE detail
  END AS detail,
  SUM(cnt) AS cnt
FROM (
  SELECT 'FFF' AS stage, fff_detail AS detail, COUNT(*) AS cnt
  FROM dwd_cfdi_status_monitor_analysis
  WHERE <公共条件> AND fff_status != 'success' AND fff_detail IS NOT NULL
  GROUP BY fff_detail
  UNION ALL
  SELECT 'FDR', fdr_detail, COUNT(*)
  FROM dwd_cfdi_status_monitor_analysis
  WHERE <公共条件> AND fff_status = 'success' AND fdr_status != 'success' AND fdr_detail IS NOT NULL
  GROUP BY fdr_detail
  UNION ALL
  SELECT 'FCL', fcl_detail, COUNT(*)
  FROM dwd_cfdi_status_monitor_analysis
  WHERE <公共条件> AND fdr_status = 'success' AND fcl_status != 'success' AND fcl_detail IS NOT NULL
  GROUP BY fcl_detail
) t
GROUP BY stage, detail
ORDER BY stage, cnt DESC
```

## 4. GET /do/cool_top — 冷却 Top20 筛选器

```sql
SELECT filter_name, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件> AND fff_status != 'success' AND fff_detail = 'check_is_no_need_cooldown' AND filter_name IS NOT NULL
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 20
```

## 5. GET /do/trigger_rank — 触发频次排行 Top10

```sql
SELECT filter_name, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件> AND filter_name IS NOT NULL
GROUP BY filter_name
ORDER BY cnt DESC
LIMIT 10
```

## 6. GET /do/sw_version — 软件版本分布

```sql
SELECT fff_sw_version AS sw_version, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件> AND fff_sw_version IS NOT NULL AND fff_sw_version != ''
GROUP BY fff_sw_version
ORDER BY cnt DESC
LIMIT 20
```

## 7. GET /do/project_car — 项目×车型分布

```sql
SELECT project_name, car_type, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件>
  AND project_name IS NOT NULL AND project_name != ''
  AND car_type IS NOT NULL AND car_type != ''
GROUP BY project_name, car_type
ORDER BY project_name, car_type
```

## 8. GET /do/mem_top — FDR 内存不足 Top20

```sql
SELECT event_name, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件>
  AND fff_status = 'success' AND fdr_status != 'success'
  AND fdr_detail IN ('because of full gc', 'mem pool water line')
  AND event_name IS NOT NULL AND event_name != ''
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 9. GET /do/disk_top — FDR 磁盘不足 Top20

```sql
SELECT event_name, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件>
  AND fff_status = 'success' AND fdr_status != 'success'
  AND fdr_detail IN ('Disk overrun', 'Exceeds the maximum number of files')
  AND event_name IS NOT NULL AND event_name != ''
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 10. GET /do/close_top — 关闭次数 Top 筛选器

```sql
SELECT filter_name, COUNT(*) AS cnt
FROM dwd_cfdi_basic_fff_close
WHERE <公共条件> AND filter_name IS NOT NULL AND filter_name != ''
GROUP BY filter_name ORDER BY cnt DESC LIMIT 20
```

## 11. GET /do/quota_top — FCL Quota 超限 Top20

```sql
SELECT event_name, COUNT(*) AS cnt
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件>
  AND fdr_status = 'success' AND fcl_status != 'success'
  AND fcl_detail = 'query cloud DISCARD, detail:Filter quota exceeded'
  AND event_name IS NOT NULL AND event_name != ''
GROUP BY event_name ORDER BY cnt DESC LIMIT 20
```

## 12. GET /do/project_event — 项目触发回流事件总数

```sql
SELECT project_name, COUNT(DISTINCT event_name) AS event_count
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件>
  AND fcl_status = 'success'
  AND project_name IS NOT NULL AND project_name != ''
GROUP BY project_name ORDER BY event_count DESC
```

## 13. GET /do/net_speed — 各车型平均上传带宽

```sql
SELECT DATE(create_at) AS dt, car_type,
  ROUND(AVG((package_size / 1024.0 / 1024.0) / (total_cost / 1000.0)), 2) AS avg_bw
FROM dwd_cfdi_basic_fcl_uploadinfo
WHERE dt BETWEEN ? AND ?   -- 默认近7天；可选 project_name / car_type
  AND package_size > 10485760 AND total_cost > 0
  AND car_type IS NOT NULL AND car_type != ''
GROUP BY DATE(create_at), car_type ORDER BY dt, car_type
```

## 14. GET /do/fcl_bw — FCL 整体平均上传带宽

```sql
SELECT DATE(create_at) AS dt,
  ROUND(AVG((package_size / 1024.0 / 1024.0) / (total_cost / 1000.0)), 2) AS avg_bw
FROM dwd_cfdi_basic_fcl_uploadinfo
WHERE dt BETWEEN ? AND ?
  AND package_size > 10485760 AND total_cost > 0
GROUP BY DATE(create_at) ORDER BY dt
```

## 15. GET /do/top_vehicles — Top20 活跃车辆

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

补充查询（每车主失败原因 `vehicleFailReasons`）：

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

## 16. GET /do/anomaly_vehicles — 异常车辆

主查询同 top_vehicles，但 HAVING 阈值是 **`fmt.Sprintf` 直接拼接 int**（非占位符参数）：

```sql
... GROUP BY anonymous_id, car_type, project_name
HAVING cfdi_rate < <max_rate>   -- fmt.Sprintf("HAVING cfdi_rate < %d", param.MaxRate)
ORDER BY cfdi_rate ASC
LIMIT 100
```

失败原因补充查询同上。

## 17. GET /do/active_trend — 活跃车辆趋势

```sql
SELECT dt, COUNT(DISTINCT anonymous_id) AS active_count
FROM dwd_cfdi_status_monitor_analysis
WHERE <车辆条件>
GROUP BY dt ORDER BY dt
```

## 18. GET /do/funnel — DO 漏斗（节点统计 + 失败原因 Top10）

并发 4 条 SQL，base：`FROM dwd_cfdi_status_monitor_analysis WHERE <公共条件> AND event_name != 'Forever_log'`

```sql
-- ① 节点统计
SELECT
  COUNT(*) AS fff_total,
  SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_allow,
  SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fff_status='success' AND fdr_status!='success' THEN 1 ELSE 0 END) AS fdr_fail,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS fcl_success,
  SUM(CASE WHEN fdr_status='success' AND fcl_status!='success' THEN 1 ELSE 0 END) AS fcl_fail
FROM dwd_cfdi_status_monitor_analysis WHERE <公共条件> AND event_name != 'Forever_log'

-- ② FFF 失败原因
SELECT fff_detail AS name, COUNT(*) AS cnt <base>
  AND fff_status!='success' AND fff_detail IS NOT NULL AND fff_detail!=''
GROUP BY fff_detail ORDER BY cnt DESC LIMIT 10

-- ③ FDR 失败原因
SELECT fdr_detail AS name, COUNT(*) AS cnt <base>
  AND fff_status='success' AND fdr_status!='success' AND fdr_detail IS NOT NULL AND fdr_detail!=''
GROUP BY fdr_detail ORDER BY cnt DESC LIMIT 10

-- ④ FCL 失败原因
SELECT fcl_detail AS name, COUNT(*) AS cnt <base>
  AND fdr_status='success' AND fcl_status!='success' AND fcl_detail IS NOT NULL AND fcl_detail!=''
GROUP BY fcl_detail ORDER BY cnt DESC LIMIT 10
```

---

# 二、FO 模块（共 10 个接口；此快照**还没有** `running/trend`）

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

## 2. GET /dashboard/v1/diag/funnel — 数采全链路分析（FO/DO 共用）

与 DO funnel 完全相同的明细表版本（并发 4 条，见上文 DO #18），阶段判定用原始 status 值：

```sql
SELECT
  COUNT(*) AS fff_total,
  SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_allow,
  SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fff_status='success' AND fdr_status!='success' THEN 1 ELSE 0 END) AS fdr_fail,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS fcl_success,
  SUM(CASE WHEN fdr_status='success' AND fcl_status!='success' THEN 1 ELSE 0 END) AS fcl_fail
FROM dwd_cfdi_status_monitor_analysis WHERE <公共条件> AND event_name != 'Forever_log'
-- + 三条失败原因 Top10（fff_detail / fdr_detail / fcl_detail 原文，带阶段前置条件）
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

条件：`dt`（默认当天 `dt = CURDATE()`）+ 可选 `filter_name / project_name / car_type`。

## 4. GET /dashboard/v1/fo/detail/trigger — 筛选器触发明细（分页）

表 `dwd_cfdi_basic_fff_trigger`：

```sql
SELECT COUNT(*) FROM dwd_cfdi_basic_fff_trigger WHERE <条件>;

SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at, trigger_time, utc_diff_us,
  `before`, `after`, filter_name, trigger_type, collect_type, status, on_autopilot, function_mode,
  sw_version, project_name, car_type, vehicle_source, bj02_lat, bj02_lon, road_type,
  fdi_project_name, project_car_type, vehicle_source_cn, tags, detail
FROM dwd_cfdi_basic_fff_trigger WHERE <条件> LIMIT <page_size> OFFSET <offset>;
```

> `before`/`after` 是 SQL 保留字，需反引号转义。

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

表 `dwd_cfdi_status_monitor_analysis`：

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

附加条件（此快照的 `stage_filter` 语义与现版不同，**带上游阶段前置条件**）：

| stage_filter | 拼接条件 |
|---|---|
| `fff_discard` | `fff_status = 'discard'` |
| `fdr_discard` | `fff_status = 'success' AND fdr_status = 'discard'` |
| `fcl_discard` | `fdr_status = 'success' AND fcl_status != 'success' AND fcl_status != ''` |
| `fcl_success` | `fcl_status = 'success'` |
| `only_fail=1` | `fcl_status != 'success'` |

> 现版（HEAD）已改为单阶段直查并新增 `fff_success / fdr_success`。

## 9. GET /dashboard/v1/fo/diag/stage_trend — 三阶段触发趋势

单条大 SQL（明细表），按 dt 聚合三阶段各 6 个分类（success + cat2~cat5 + other），靠 `INSTR` 匹配 detail 原文：

```sql
SELECT dt,
  SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_success,
  SUM(CASE WHEN fff_status!='success' AND fff_detail='check_is_no_need_cooldown' THEN 1 ELSE 0 END) AS fff_cat2,
  SUM(CASE WHEN fff_status!='success' AND fff_detail IN ('check_drm_quota','check_drm_quota_weight') THEN 1 ELSE 0 END) AS fff_cat3,
  SUM(CASE WHEN fff_status!='success' AND fff_detail='check_not_reach_trigger_maximum' THEN 1 ELSE 0 END) AS fff_cat4,
  SUM(CASE WHEN fff_status!='success' AND fff_detail='check_need_acquire_data' THEN 1 ELSE 0 END) AS fff_cat5,
  SUM(CASE WHEN fff_status!='success'
      AND fff_detail NOT IN ('check_is_no_need_cooldown','check_drm_quota','check_drm_quota_weight',
                             'check_not_reach_trigger_maximum','check_need_acquire_data')
      THEN 1 ELSE 0 END) AS fff_cat_other,
  SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'because of full gc')>0 THEN 1 ELSE 0 END) AS fdr_cat2,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'do not recognized')>0 THEN 1 ELSE 0 END) AS fdr_cat3,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'mem pool water line')>0 THEN 1 ELSE 0 END) AS fdr_cat4,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'Bag invalid')>0 THEN 1 ELSE 0 END) AS fdr_cat5,
  SUM(CASE WHEN fdr_status!='success'
      AND INSTR(fdr_detail,'because of full gc')=0 AND INSTR(fdr_detail,'do not recognized')=0
      AND INSTR(fdr_detail,'mem pool water line')=0 AND INSTR(fdr_detail,'Bag invalid')=0
      THEN 1 ELSE 0 END) AS fdr_cat_other,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS fcl_success,
  SUM(CASE WHEN fcl_status!='success' AND INSTR(fcl_detail,'geofence forbidden')>0 THEN 1 ELSE 0 END) AS fcl_cat2,
  SUM(CASE WHEN fcl_status!='success' AND (INSTR(fcl_detail,'bag not exist')>0 OR INSTR(fcl_detail,'meta file lost')>0) THEN 1 ELSE 0 END) AS fcl_cat3,
  SUM(CASE WHEN fcl_status!='success' AND (INSTR(fcl_detail,'reach upload limit')>0 OR INSTR(fcl_detail,'Filter quota exceeded')>0) THEN 1 ELSE 0 END) AS fcl_cat4,
  SUM(CASE WHEN fcl_status!='success' AND INSTR(fcl_detail,'EventName is in blacklist')>0 THEN 1 ELSE 0 END) AS fcl_cat5,
  SUM(CASE WHEN fcl_status!='success'
      AND INSTR(fcl_detail,'geofence forbidden')=0 AND INSTR(fcl_detail,'bag not exist')=0
      AND INSTR(fcl_detail,'meta file lost')=0 AND INSTR(fcl_detail,'reach upload limit')=0
      AND INSTR(fcl_detail,'Filter quota exceeded')=0 AND INSTR(fcl_detail,'EventName is in blacklist')=0
      THEN 1 ELSE 0 END) AS fcl_cat_other
FROM dwd_cfdi_status_monitor_analysis
WHERE <公共条件> AND event_name != 'Forever_log'
GROUP BY dt ORDER BY dt ASC
```

返回的系列名（中文展示名，与现版 tag 名不同）：

- FFF：`FFF 成功 / 冷却丢弃 / DRM Quota / 触发上限 / 数采限制 / 其他丢弃`
- FDR：`FDR 成功 / Full GC / 事件不识别 / 内存限制 / Bag Invalid / 其他丢弃`
- FCL：`FCL 成功 / Geofence限制 / bag/meta丢失 / 上传Quota / 事件黑名单 / 其他丢弃`

## 10. GET /dashboard/v1/fo/diag/close_reason — 算子关闭原因分布

表 `dwd_cfdi_basic_fff_close`，按 reason 关键字归类（与现版一致）：

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

---

## 附：明细版 vs 聚合版（HEAD）关键差异速查

| 维度 | 明细版（本文档快照 4fea9a4） | 聚合版（HEAD 6e3d362） |
|------|------|------|
| 核心表 | 全部 `dwd_*` 明细表 | DO 聚合接口改查 `ads_do_cfdi_daily`（`SUM(cnt)`） |
| 失败归因 | `xxx_detail` 原文 `=`/`IN`/`LIKE`/`INSTR` 匹配 | `xxx_detail_tag` 规整列等值匹配 |
| 阶段判定 | `fff_status='success'` 等精确值 | 增加 `fcl_status=''` 表示"未到 FCL"等更严口径 |
| 计数方式 | `COUNT(*)` / `SUM(CASE...1 ELSE 0)` | `SUM(cnt)` |
| overview | 单条 SQL | ads + dwd 两条并发（车辆数仍查明细） |
| uuid stage_filter | 带上游阶段前置条件 | 单阶段直查，新增 fff/fdr_success |
| stage_trend 分类 | 每阶段 6 类（中文展示名） | 每阶段 8~11 类（tag 名） |
| 其他 | 带慢查询日志；无 UM 鉴权中间件；无 running/trend 接口 | 慢查询日志移除；加 UM 鉴权；新增 running/trend |
