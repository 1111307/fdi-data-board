# 数采领域字典(lookup_dict 工具数据源)

## kind=table: 表与字段

三阶段:**FFF**(筛选器触发)→ **FDR**(落盘)→ **FCL**(上传)。各阶段**独立统计**,禁止相加/相除算"全链路成功率"。

**表与工具的对应**:
- FFF 触发(筛选器在车端触发录制)→ dwd_cfdi_basic_fff_trigger_daily_summary
- FFF 运行(筛选器每 5 分钟心跳上报,含开关状态)→ dwd_cfdi_basic_fff_running_daily_summary
- FFF 关闭(筛选器被关闭的原因)→ dwd_cfdi_basic_fff_close_daily_summary
- FDR 落盘(录制数据写盘)→ dwd_basic_fdr_trigger_daily_summary
- FDR 带宽(各传感器/数据组带宽,field 粒度按 bandwidth_field 分组,**数值单位为字节**)→ dwd_cfdi_basic_fdr_bandwidth_daily_summary
- FCL 上传(数据包上传云端)→ fcl_trigger / fcl_uploadinfo_daily_summary
- 全链路漏斗(按 uuid 追踪三阶段)→ dwd_cfdi_status_monitor_analysis_daily_summary
- 去重车辆数(唯一可跨维度用车数的表,**后缀 _agg**)→ ads_cfdi_vehicle_daily_summary_agg

**关键字段(汇总表通用)**:
- `event_count` / `success_count` / `failed_count`:事件次数(可跨天 SUM)
- `vehicle_count`:"当前完整分组内去重车辆数"——**不可跨天/跨维度 SUM**(跨行会重复计车)
- `__ALL__` 哨兵(双向规则):表里同时存 `__ALL__` 汇总行和真实值明细行。查**总量**(用户没点名事件/筛选器)必须只查 `__ALL__` 行;查**分布**必须排除 `__ALL__`。方向用反,数字必错(重复计车或翻倍)。
- `dt`:统计日(分区键);`summary_grain`:overview(总量)/filter/status/reason(粒度)

**_agg 表去重车辆字段**(全部"分组内去重";明细行按 event_name 展开,同车跑多个筛选器就落多行,**SUM 明细行必重复计车**;总量只查 `__ALL__` 行——该口径同车一天只一行,SUM 安全):
- `running_vehicle_count`:FFF Running 去重车辆数
- `running_switch_on_vehicle_count`:FFF Running 且开关开启的去重车辆数
- `trigger_vehicle_count` / `fff_success_vehicle_count` / `fff_failed_vehicle_count`:FFF 触发/成功/失败去重
- `fdr_vehicle_count` / `fdr_success_vehicle_count` / `fdr_failed_vehicle_count`:FDR 去重
- `fcl_vehicle_count` / `fcl_success_vehicle_count` / `fcl_failed_vehicle_count`:FCL 去重
- `upload_vehicle_count` / `overall_vehicle_count`:上传/全链路去重(overall 仅全链路任务写入)

**event_name/filter_name 列兼容(重要)**:
- 有 event_name 列的表(trigger/monitor/_agg 等):前端传的值直接查 event_name 列;
- **没有 event_name 列的表**(fff_running/fff_running_daily_summary/fff_close/fff_close_daily_summary):前端传的值被当作 filter_name 查。
  传筛选器名(如 auto_off_filter)能查到;传事件名(如 ACC_FAIL_md_fault_reported)返回 0——这是如实结果(运行/关闭表只记录筛选器维度,事件名不在这些表里),不是查询失败。
- 解释 0 结果时:若某名字在 running/overview 类查询返回 0,说明它是事件名,引导用户改查 get_fail_reason/get_top/get_stage_trend 等走 event_name 的工具。

**标识**:`anonymous_id`=车辆标识(形如 byd0FB4C567E640E21F)、`uuid`=单次触发链路、`md5`=数据包。

## kind=detail: 明细表字段(每行 = 一次真实事件)

| 表 | 含义 | 关键字段 |
|---|---|---|
| fff_trigger | 筛选器触发明细 | `before/after`=触发前后秒数(录制窗口)、`trigger_type`=触发类型、`collect_type`=采集类型、`status`=触发状态、`tags`=标签数组、`detail`=详情文本 |
| fff_running | 筛选器心跳明细(每5分钟) | `switch_on`=开关(1开0关)、`version`=筛选器版本、`status`=状态JSON |
| fff_close | 筛选器关闭明细 | `reason`=关闭原因文本(如 out of memory)、`version`=筛选器版本 |
| fdr_trigger | 落盘明细 | `td_mb`=TD大小(MB)、`tm_mb`=TM大小(MB)、`dse`=DSE信息、`time_cost_ms`=耗时(毫秒)、`begin/end/dump_timestamp`=三阶段时间戳(微秒) |
| fcl_trigger | 上传触发明细 | `complete_percent`=完成百分比、`upload_fail_times`=失败次数、`local_file`=本地文件路径、`trigger_source`=触发来源(FFF/Forever_log/FDC) |
| monitor_analysis(uuid明细) | 全链路明细(按uuid) | `fff/fdr/fcl_status`=三阶段各自状态、`fff/fdr/fcl_updated_at`=各阶段时间戳(微秒)、`fff/fdr/fcl_detail`=各阶段详情文本、`begin/dump/end_timestamp`=链路三节点、`md5`=包哈希、`bag_name`=包名 |

字段语义:`td_mb`/`tm_mb` 是字符串型 MB 大小(注意不是数字)、`before/after` 是录制窗口秒数(触发前 N 秒 + 触发后 M 秒的录制范围)、`trigger_source=Forever_log` 表示手动/永久日志触发(非事件触发)。

## kind=trap: 聚合陷阱

1. **vehicle_count 不可跨行 SUM**:这些字段都是"当前分组内去重"。同一辆车会出现在多个 event_name 分组行(同时跑多个筛选器),SUM 明细行会重复计车——实测某日全表 SUM=774w,`__ALL__` 口径正确值=28.6w,膨胀 27 倍。正确取数:**总量车辆数只查 `__ALL__` 行**;**指定事件/筛选器**查 `event_name=该值` 的行;跨天区间先按天 SUM(__ALL__ 行当天已去重)再取 MAX 得"峰值日活"。
2. **event_count 可 SUM**:事件类计数(event_count/success_count/failed_count/running_count 等)跨行相加安全。
3. **__ALL__ 双向**:查分布(列各值排行)必须排除 __ALL__ 与空串,否则翻倍;查总量(未点名筛选值)必须只查 __ALL__ 行,否则明细行重复计车。汇总行和明细行混在一起 SUM 必错。
4. **P95 不可再聚合**:分位值只能原样引用。
5. **单位**:package_size=字节(÷1024³=GB)、td_mb/tm_mb=MB、time_cost_ms=毫秒、fragment=千分比(÷10=%)、bandwidth=MB/s。
6. **数据延迟**:当日/近两日可能不全,判断"无数据"前先考虑时间窗是否太近。
7. **字符串数字**:td_mb/tm_mb 是字符串型,数值比较需转换。
8. **lianhuashan 不支持车辆数查询**(背景:心跳链路 anonymous_id 全部丢失,车辆维度数据不可信——明细表查它车辆数恒为 1,_agg 表只记几十辆,真实车队远大于此;全局车辆总数口径已剔除该项目)。用户问 lianhuashan 车辆数时,**对外统一口径:"该项目暂不支持车辆数查询"**——不解释原因、不说数据异常。
9. **"正常丢弃/有效拦截"不计入成功率,且与真失败分开展示**:落盘 detail_tag 为 `event_not_recognized`/`max_files_exceeded`/`unauthorized`(正常丢弃)、上传 detail_tag 为 `event_in_blacklist`/`quota_exceeded`/`unexpected_bag_upload_query`/`reach_upload_limit`(有效拦截)的,都是策略性丢弃(事件黑名单、配额超限等,非故障),计算对应阶段成功率时既不进分子也不进分母——fdr_quality 的 fdr_total 已剔除,成功率=fdr_success÷fdr_total 即有效口径;解释时单独归类为"正常丢弃/有效拦截",不与真失败混排。**FFF 阶段全部为"有效拦截"**(cooldown 等防抖行为,触发没有"失败"语义,禁止说"触发成功率")。注意:funnel 数据源(ads_do_cfdi_daily)存白话原文(如 `Exceeds the maximum number of files`),fdr/fcl 汇总表存规范名,两套命名勿混用。
10. **未终态事件(进行中)**:FDR 阶段存在 waiting/dumping 等未终态事件,既非成功也非失败——总数 = 成功 + 正常丢弃 + 失败 + 进行中;差额不是漏统计,解释时说明"进行中"。当日/近两日进行中占比较高属正常(数据还在写入),历史日期仍大量进行中的多为车端断电等死账。
