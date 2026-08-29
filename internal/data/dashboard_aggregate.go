package data

import (
	"strings"
)

const (
	aggAllValue = "__ALL__"

	// aggForeverLogValue forever_log 在 Doris 中是小写，且字符串比较大小写敏感
	aggForeverLogValue = "forever_log"

	tableStatusDailySummary       = "fdi.dwd_cfdi_status_monitor_analysis_daily_summary"
	tableFffTriggerDailySummary   = "fdi.dwd_cfdi_basic_fff_trigger_daily_summary"
	tableFffCloseDailySummary     = "fdi.dwd_cfdi_basic_fff_close_daily_summary"
	tableFffRunningDailySummary   = "fdi.dwd_cfdi_basic_fff_running_daily_summary"
	tableFclUploadDailySummary    = "fdi.dwd_cfdi_basic_fcl_uploadinfo_daily_summary"
	tableFclTriggerDailySummary   = "fdi.dwd_cfdi_basic_fcl_trigger_daily_summary"
	tableFdrTriggerDailySummary   = "fdi.dwd_basic_fdr_trigger_daily_summary"
	tableFdrFragmentDailySummary  = "fdi.dwd_cfdi_basic_fdr_fragment_daily_summary"
	tableFdrBandwidthDailySummary = "fdi.dwd_cfdi_basic_fdr_bandwidth_daily_summary"
	tableVehicleDailySummaryAgg   = "fdi.ads_cfdi_vehicle_daily_summary_agg"
)

func buildAggEventCondition(eventNames []string) (string, []interface{}) {
	if len(eventNames) == 0 {
		return "event_name = ?", []interface{}{aggAllValue}
	}
	if len(eventNames) == 1 {
		return "event_name = ?", []interface{}{eventNames[0]}
	}
	placeholders := strings.Repeat("?,", len(eventNames))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]interface{}, 0, len(eventNames))
	for _, eventName := range eventNames {
		args = append(args, eventName)
	}
	return "event_name IN (" + placeholders + ")", args
}

// buildAggDateCondition 日期条件：未传日期时默认近 7 天（含今天）
func buildAggDateCondition(startDt, endDt string) (string, []interface{}) {
	if startDt != "" && endDt != "" {
		return "dt BETWEEN ? AND ?", []interface{}{startDt, endDt}
	}
	return "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)", nil
}

func buildAggCommonWhere(summaryGrain string, filterName string, eventNames []string, projectName string, carTypes []string, startDt, endDt string) (string, []interface{}) {
	var conds []string
	var args []interface{}

	dateCond, dateArgs := buildAggDateCondition(startDt, endDt)
	conds = append(conds, dateCond)
	args = append(args, dateArgs...)

	if summaryGrain != "" {
		conds = append(conds, "summary_grain = ?")
		args = append(args, summaryGrain)
	}

	eventCond, eventArgs := buildAggEventCondition(eventNames)
	conds = append(conds, eventCond)
	args = append(args, eventArgs...)

	if filterName != "" {
		conds = append(conds, "filter_name = ?")
		args = append(args, filterName)
	}
	if projectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, projectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", carTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func buildAggDimensionWhere(summaryGrain string, projectName string, carTypes []string, startDt, endDt string) (string, []interface{}) {
	var conds []string
	var args []interface{}

	dateCond, dateArgs := buildAggDateCondition(startDt, endDt)
	conds = append(conds, dateCond)
	args = append(args, dateArgs...)

	if summaryGrain != "" {
		conds = append(conds, "summary_grain = ?")
		args = append(args, summaryGrain)
	}
	if projectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, projectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", carTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func isAggAllOrEmpty(value string) bool {
	return value == "" || value == aggAllValue
}

func fdrStageFailedCondition() string {
	return "fff_status != 'discard' AND fdr_status = 'discard' AND fcl_status = ''"
}

// grainForFilter status_monitor 类汇总表：传了 filter_name 用 filter 粒度，否则用 overview
func grainForFilter(filterName string) string {
	if filterName == "" {
		return "overview"
	}
	return "filter"
}

// buildAggRealEventCondition 按 event_name 展开（GROUP BY event）的场景：
// 排除 __ALL__ 汇总行，且永远排除 forever_log；传了事件再叠加 IN 过滤
func buildAggRealEventCondition(eventNames []string) (string, []interface{}) {
	conds := []string{"event_name <> ?", "event_name <> ?"}
	args := []interface{}{aggAllValue, aggForeverLogValue}
	if len(eventNames) > 0 {
		placeholders := strings.Repeat("?,", len(eventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range eventNames {
			args = append(args, e)
		}
	}
	return strings.Join(conds, " AND "), args
}

// buildSummaryCommonWhere 汇总表公共 WHERE（跨事件聚合场景）：
// 汇总表同一粒度同时写 __ALL__ 汇总行和真实事件行，未传 eventNames 时必须只查 __ALL__ 行，
// 否则 SUM 会把 __ALL__ 行和真实事件行重复计算（结果翻倍）。传了 eventNames 用 IN。
func buildSummaryCommonWhere(filterName string, eventNames []string, projectName string, carTypes []string, startDt, endDt string) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if startDt != "" && endDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, startDt, endDt)
	} else {
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}
	if filterName != "" {
		conds = append(conds, "filter_name = ?")
		args = append(args, filterName)
	}
	eventCond, eventArgs := buildAggEventCondition(eventNames)
	conds = append(conds, eventCond)
	args = append(args, eventArgs...)
	if projectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, projectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", carTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}
