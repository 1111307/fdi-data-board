package data

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/wire"

	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

var DashboardDoProviderSet = wire.NewSet(NewDoDashboardRepo)

var _ biz.DoDashboardRepo = (*doDashboardRepo)(nil)

type doDashboardRepo struct {
	*baseRepo
}

func NewDoDashboardRepo(data *Data) biz.DoDashboardRepo {
	return &doDashboardRepo{baseRepo: &baseRepo{data: data}}
}

// doOverviewRow 事件横向对比聚合行
type doOverviewRow struct {
	EventName    string `gorm:"column:event_name"`
	VehicleCount int64  `gorm:"column:vehicle_count"`
	TriggerCount int64  `gorm:"column:trigger_count"`
	FffCount     int64  `gorm:"column:fff_count"`
	FdrCount     int64  `gorm:"column:fdr_count"`
	FclCount     int64  `gorm:"column:fcl_count"`
}

func (r *doDashboardRepo) GetOverview(ctx context.Context, param *biz.DoOverviewParam) ([]*dashboard_api.DoOverviewItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT event_name,
		COUNT(DISTINCT anonymous_id) AS vehicle_count,
		COUNT(*) AS trigger_count,
		SUM(CASE WHEN fff_status='success' THEN 1 ELSE 0 END) AS fff_count,
		SUM(CASE WHEN fdr_status='success' THEN 1 ELSE 0 END) AS fdr_count,
		SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS fcl_count
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		GROUP BY event_name ORDER BY trigger_count DESC`

	var rows []*doOverviewRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	pct := func(a, b int64) float64 {
		if b == 0 {
			return 0
		}
		return math.Round(float64(a)/float64(b)*1000) / 10
	}

	list := make([]*dashboard_api.DoOverviewItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &dashboard_api.DoOverviewItem{
			EventName:    row.EventName,
			VehicleCount: row.VehicleCount,
			TriggerCount: row.TriggerCount,
			CfdiRate:     pct(row.FclCount, row.TriggerCount),
			FffCount:     row.FffCount,
			FffRate:      pct(row.FffCount, row.TriggerCount),
			FdrCount:     row.FdrCount,
			FdrRate:      pct(row.FdrCount, row.FffCount),
			FclCount:     row.FclCount,
			FclRate:      pct(row.FclCount, row.FdrCount),
		})
	}
	return list, nil
}

// doTrendRow 趋势聚合行
type doTrendRow struct {
	Dt      time.Time `gorm:"column:dt"`
	Total   int64     `gorm:"column:total"`
	Success int64     `gorm:"column:success"`
}

func (r *doDashboardRepo) GetTrend(ctx context.Context, param *biz.DoTrendParam) (*biz.DoTrendData, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoTrendWhere(param)

	sql := `SELECT dt,
		COUNT(*) AS total,
		SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS success
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		GROUP BY dt ORDER BY dt ASC`

	var rows []*doTrendRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	dates := make([]string, 0, len(rows))
	counts := make([]int64, 0, len(rows))
	rates := make([]float64, 0, len(rows))

	for _, row := range rows {
		dates = append(dates, row.Dt.Format("2006-01-02"))
		counts = append(counts, row.Success)
		rate := 0.0
		if row.Total > 0 {
			rate = math.Round(float64(row.Success)/float64(row.Total)*1000) / 10
		}
		rates = append(rates, rate)
	}

	return &biz.DoTrendData{
		Dates:         dates,
		SuccessCounts: counts,
		SuccessRates:  rates,
	}, nil
}

// failReasonRow UNION ALL 聚合行
type failReasonRow struct {
	Stage  string `gorm:"column:stage"`
	Detail string `gorm:"column:detail"`
	Cnt    int64  `gorm:"column:cnt"`
}

func (r *doDashboardRepo) GetFailReason(ctx context.Context, param *biz.DoFailReasonParam) ([]*dashboard_api.DoFailReasonItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	// UNION ALL 三段，每段有相同的 WHERE 参数，args 需要重复三份
	tripleArgs := append(append(append([]interface{}{}, args...), args...), args...)

	sql := `SELECT 'FFF' AS stage, fff_detail AS detail, COUNT(*) AS cnt
		FROM dwd_cfdi_status_monitor_analysis` + where + ` AND fff_status != 'success' AND fff_detail IS NOT NULL
		GROUP BY fff_detail
		UNION ALL
		SELECT 'FDR' AS stage, fdr_detail AS detail, COUNT(*) AS cnt
		FROM dwd_cfdi_status_monitor_analysis` + where + ` AND fff_status = 'success' AND fdr_status != 'success' AND fdr_detail IS NOT NULL
		GROUP BY fdr_detail
		UNION ALL
		SELECT 'FCL' AS stage, fcl_detail AS detail, COUNT(*) AS cnt
		FROM dwd_cfdi_status_monitor_analysis` + where + ` AND fdr_status = 'success' AND fcl_status != 'success AND fcl_detail IS NOT NULL
		GROUP BY fcl_detail
		ORDER BY stage, cnt DESC`

	var rows []*failReasonRow
	if err := db.Raw(sql, tripleArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*dashboard_api.DoFailReasonItem, 0, len(rows))
	for _, row := range rows {
		name := row.Stage + "-" + row.Detail
		list = append(list, &dashboard_api.DoFailReasonItem{
			Name:  name,
			Value: row.Cnt,
		})
	}
	return list, nil
}

// buildDoCommonWhere 构建 DO dashboard 公共 WHERE 子句（dwd_cfdi_status_monitor_analysis）
func buildDoCommonWhere(filterName string, eventNames []string, projectName string, carTypes []string, startDt, endDt string) (string, []interface{}) {
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
	conds, args = appendMultiCond(conds, args, "event_name", eventNames)
	if projectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, projectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", carTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func buildDoTrendWhere(param *biz.DoTrendParam) (string, []interface{}) {
	return buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
}

func appendMultiCond(conds []string, args []interface{}, col string, vals []string) ([]string, []interface{}) {
	if len(vals) == 1 {
		conds = append(conds, col+" = ?")
		args = append(args, vals[0])
	} else if len(vals) > 1 {
		placeholders := strings.Repeat("?,", len(vals))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, col+" IN ("+placeholders+")")
		for _, v := range vals {
			args = append(args, v)
		}
	}
	return conds, args
}

// toDoTrendResponse 供 FunnelChart 等公共接口复用（预留）
var _ = dashboard_api.DoTrendResponse{}
