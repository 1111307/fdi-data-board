# DO / FO Dashboard 明细表直查 SQL（聚合表引入前的历史版本）

> **快照来源**：git commit `a1e2f6a^`（即 `a1e2f6a feat:FO页面读预聚合表` 的父提交）。
> 这是 **第一个引入 `ads_do_cfdi_daily` 聚合表** 的提交之前的最后状态 —— 所有看板接口全部直接扫 Doris **明细表**。
> 代码位置：`internal/data/dashboard_do.go`、`internal/data/dashboard_fo.go`（该快照版本）。

## 涉及的明细表

| 表名 | 说明 |
|------|------|
| `dwd_cfdi_status_monitor_analysis` | 全链路状态明细表（uuid 维度，含 fff/fdr/fcl 三阶段 status + detail），DO 聚合类接口主表 |
| `dwd_cfdi_basic_fff_running` | 筛选器运行明细 |
| `dwd_cfdi_basic_fff_trigger` | 筛选器触发明细 |
| `dwd_cfdi_basic_fff_close` | 筛选器关闭明细 |
| `dwd_basic_fdr_trigger` | FDR 落盘明细 |
| `dwd_cfdi_basic_fcl_trigger` | FCL 上传明细 |
| `dwd_cfdi_basic_fcl_uploadinfo` | FCL 上传带宽信息 |

## 公共 WHERE 条件构建

`buildDoCommonWhere`（DO 聚合接口共用，作用于 `dwd_cfdi_status_monitor_analysis`）：

```sql
WHERE dt BETWEEN ? AND ?            -- 不传日期时：dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)
  AND filter_name = ?               -- 可选
  AND event_name = ? / IN (...)     -- 可选，多选
  AND project_name = ?              -- 可选
  AND car_type = ? / IN (...)       -- 可选，多选
```

FO 明细分页接口的 `buildXxxWhere` 类似，日期不传时默认 `dt = CURDATE()`。

---

# 一、DO 模块（`/dashboard/v1/do/*`）— 明细表版本

## 1. GET /do/overview — 事件横向对比

**单条 SQL**（明细版只扫 `dwd_cfdi_status_monitor_analysis`，一条出全部计数；聚合版拆成了 ads + dwd 两条）：

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

> 公共条件 args 需重复 3 份。注意明细版直接用原始 `detail` 文本 + LIKE 归类，聚合版改用了预计算的 `xxx_detail_tag`。

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

失败原因补充查询（`vehicleFailReasons`）：

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

主查询同 top_vehicles，但加（注意：该版本 `max_rate` 是 `fmt.Sprintf` 内联进 SQL 的）：

```sql
... GROUP BY anonymous_id, car_type, project_name
HAVING cfdi_rate < <max_rate>   -- 内联整数
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

## 18. GET /do/funnel — DO 漏斗

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

-- ② FFF 失败原因 Top10
SELECT fff_detail AS name, COUNT(*) AS cnt <base>
  AND fff_status!='success' AND fff_detail IS NOT NULL AND fff_detail!=''
GROUP BY fff_detail ORDER BY cnt DESC LIMIT 10

-- ③ FDR 失败原因 Top10
SELECT fdr_detail AS name, COUNT(*) AS cnt <base>
  AND fff_status='success' AND fdr_status!='success' AND fdr_detail IS NOT NULL AND fdr_detail!=''
GROUP BY fdr_detail ORDER BY cnt DESC LIMIT 10

-- ④ FCL 失败原因 Top10
SELECT fcl_detail AS name, COUNT(*) AS cnt <base>
  AND fdr_status='success' AND fcl_status!='success' AND fcl_detail IS NOT NULL AND fcl_detail!=''
GROUP BY fcl_detail ORDER BY cnt DESC LIMIT 10
```

---

# 二、FO 模块 — 明细表版本

## 1. GET /dashboard/v1/dimensions — 下拉维度（60 分钟缓存）

并发 4 条 DISTINCT（近 7 天），与现版本一致：

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

**明细版**（扫 `dwd_cfdi_status_monitor_analysis`，与 DO funnel 口径相同），并发 4 条：

```sql
SELECT
  COUNT(*) AS fff_total,
  SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_allow,
  SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fff_status='success' AND fdr_status!='success' THEN 1 ELSE 0 END) AS fdr_fail,
  SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS fcl_success,
  SUM(CASE WHEN fdr_status='success' AND fcl_status!='success' THEN 1 ELSE 0 END) AS fcl_fail
FROM dwd_cfdi_status_monitor_analysis WHERE <公共条件> AND event_name != 'Forever_log'
-- + 三条失败原因 Top10（fff/fdr/fcl_detail，带阶段前置条件，同 DO funnel ②③④）
```

## 3~7. 明细分页接口（与现版本一致，本来就查明细表）

| 接口 | 表 | 说明 |
|------|-----|------|
| GET /fo/detail/running | `dwd_cfdi_basic_fff_running` | 筛选器运行明细 |
| GET /fo/detail/trigger | `dwd_cfdi_basic_fff_trigger` | 触发明细（`before`/`after` 需反引号） |
| GET /fo/detail/close | `dwd_cfdi_basic_fff_close` | 关闭明细 |
| GET /fo/detail/fdr | `dwd_basic_fdr_trigger` | FDR 落盘明细 |
| GET /fo/detail/fcl | `dwd_cfdi_basic_fcl_trigger` | FCL 上传明细 |

均为 `SELECT COUNT(*) ...` + `SELECT <列> ... LIMIT ? OFFSET ?` 并发两条。

## 8. GET /fo/detail/uuid — 全链路明细（分页）

表 `dwd_cfdi_status_monitor_analysis`。与现版本基本一致，但 **`stage_filter` 判定不同**（明细版带阶段前置条件）：

```sql
-- only_fail=1 → fcl_status != 'success'
-- stage_filter:
--   fff_discard → fff_status = 'discard'
--   fdr_discard → fff_status = 'success' AND fdr_status = 'discard'
--   fcl_discard → fdr_status = 'success' AND fcl_status != 'success' AND fcl_status != ''
--   fcl_success → fcl_status = 'success'
```

## 9. GET /fo/diag/stage_trend — 三阶段触发趋势

**明细版是 6 分类**（success + 4 个明细类 + other），扫 `dwd_cfdi_status_monitor_analysis`，用 `INSTR(detail,...)` 文本匹配归类：

```sql
SELECT dt,
  -- FFF：success / 冷却丢弃 / DRM Quota / 触发上限 / 数采限制 / 其他
  SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_success,
  SUM(CASE WHEN fff_status!='success' AND fff_detail='check_is_no_need_cooldown' THEN 1 ELSE 0 END) AS fff_cat2,
  SUM(CASE WHEN fff_status!='success' AND fff_detail IN ('check_drm_quota','check_drm_quota_weight') THEN 1 ELSE 0 END) AS fff_cat3,
  SUM(CASE WHEN fff_status!='success' AND fff_detail='check_not_reach_trigger_maximum' THEN 1 ELSE 0 END) AS fff_cat4,
  SUM(CASE WHEN fff_status!='success' AND fff_detail='check_need_acquire_data' THEN 1 ELSE 0 END) AS fff_cat5,
  SUM(CASE WHEN fff_status!='success'
      AND fff_detail NOT IN ('check_is_no_need_cooldown','check_drm_quota','check_drm_quota_weight',
                             'check_not_reach_trigger_maximum','check_need_acquire_data')
      THEN 1 ELSE 0 END) AS fff_cat_other,
  -- FDR：success / Full GC / 事件不识别 / 内存限制 / Bag Invalid / 其他
  SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_success,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'because of full gc')>0 THEN 1 ELSE 0 END) AS fdr_cat2,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'do not recognized')>0 THEN 1 ELSE 0 END) AS fdr_cat3,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'mem pool water line')>0 THEN 1 ELSE 0 END) AS fdr_cat4,
  SUM(CASE WHEN fdr_status!='success' AND INSTR(fdr_detail,'Bag invalid')>0 THEN 1 ELSE 0 END) AS fdr_cat5,
  SUM(CASE WHEN fdr_status!='success'
      AND INSTR(fdr_detail,'because of full gc')=0 AND INSTR(fdr_detail,'do not recognized')=0
      AND INSTR(fdr_detail,'mem pool water line')=0 AND INSTR(fdr_detail,'Bag invalid')=0
      THEN 1 ELSE 0 END) AS fdr_cat_other,
  -- FCL：success / Geofence限制 / bag/meta丢失 / 上传Quota / 事件黑名单 / 其他
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

> 系列名（中文）：FFF = 成功/冷却丢弃/DRM Quota/触发上限/数采限制/其他丢弃；FDR = 成功/Full GC/事件不识别/内存限制/Bag Invalid/其他丢弃；FCL = 成功/Geofence限制/bag、meta丢失/上传Quota/事件黑名单/其他丢弃。

## 10. GET /fo/diag/close_reason — 算子关闭原因分布

表 `dwd_cfdi_basic_fff_close`，与现版本一致：

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

## 与聚合版的关键差异备忘

| 维度 | 明细版（本文档快照） | 聚合版（现 HEAD） |
|------|------|------|
| 主表 | `dwd_cfdi_status_monitor_analysis`（行级） | `ads_do_cfdi_daily`（按天+事件+维度预聚合，`SUM(cnt)`） |
| 失败原因字段 | 原始 `xxx_detail` 文本 + `LIKE/INSTR` 归类 | 预计算 `xxx_detail_tag` 枚举 |
| 阶段判定 | `status='success'/'discard'` 二值 | 含 `fcl_status=''` 表示"未到 FCL"的更严格链路口径 |
| 计数方式 | `COUNT(*)` / `COUNT(DISTINCT)` | `SUM(cnt)` |
| overview | 单条 SQL | 拆成 ads（计数）+ dwd（车辆数）两条并发 |
| stage_trend | 6 分类、INSTR 文本匹配 | 26 分类、detail_tag 精确匹配 |
