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

func buildDoTrendWhere(param *biz.DoTrendParam) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if param.StartDt != "" && param.EndDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, param.StartDt, param.EndDt)
	} else {
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}

	if param.FilterName != "" {
		conds = append(conds, "filter_name = ?")
		args = append(args, param.FilterName)
	}
	conds, args = appendMultiCond(conds, args, "event_name", param.EventNames)
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
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
