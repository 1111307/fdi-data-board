package data

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"

	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

var DashboardFoProviderSet = wire.NewSet(NewFoDashboardRepo)

var _ biz.FoDashboardRepo = (*foDashboardRepo)(nil)

type foDashboardRepo struct {
	*baseRepo
}

func NewFoDashboardRepo(data *Data) biz.FoDashboardRepo {
	return &foDashboardRepo{baseRepo: &baseRepo{data: data}}
}

// fffRunningRow Doris 扫描结构
// dt/create_at(DATE) 和 timestamp_utc(DATETIME) 用 time.Time 接收，避免 parseTime=True 带来的时区后缀问题
type fffRunningRow struct {
	Dt              time.Time `gorm:"column:dt"`
	FilterName      string    `gorm:"column:filter_name"`
	AnonymousId     string    `gorm:"column:anonymous_id"`
	TimestampUtc    time.Time `gorm:"column:timestamp_utc"`
	CreateAt        time.Time `gorm:"column:create_at"`
	CollectType     string    `gorm:"column:collect_type"`
	SwVersion       string    `gorm:"column:sw_version"`
	ProjectName     string    `gorm:"column:project_name"`
	CarType         string    `gorm:"column:car_type"`
	VehicleSource   string    `gorm:"column:vehicle_source"`
	SwitchOn        int8      `gorm:"column:switch_on"`
	Version         string    `gorm:"column:version"`
	OnAutopilot     string    `gorm:"column:on_autopilot"`
	FunctionMode    string    `gorm:"column:function_mode"`
	Status          string    `gorm:"column:status"`
	FdiProjectName  string    `gorm:"column:fdi_project_name"`
	ProjectCarType  string    `gorm:"column:project_car_type"`
	VehicleSourceCn string    `gorm:"column:vehicle_source_cn"`
}

func (r *foDashboardRepo) ListFffRunning(ctx context.Context, param *biz.FffRunningParam) ([]*dashboard_api.FffRunningItem, int64, error) {
	db := r.dorisDB(ctx)

	where, args := buildFffRunningWhere(param)

	var (
		total    int64
		rows     []*fffRunningRow
		countErr error
		dataErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countSQL := "SELECT COUNT(*) FROM dwd_cfdi_basic_fff_running" + where
		countErr = db.Raw(countSQL, args...).Scan(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (param.Page - 1) * param.PageSize
		dataSQL := fmt.Sprintf(
			`SELECT dt, filter_name, anonymous_id, timestamp_utc, create_at, collect_type,
			sw_version, project_name, car_type, vehicle_source, switch_on, version,
			on_autopilot, function_mode, status, fdi_project_name, project_car_type, vehicle_source_cn
			FROM dwd_cfdi_basic_fff_running%s LIMIT %d OFFSET %d`,
			where, param.PageSize, offset,
		)
		dataErr = db.Raw(dataSQL, args...).Scan(&rows).Error
	}()

	wg.Wait()

	if countErr != nil {
		return nil, 0, countErr
	}
	if dataErr != nil {
		return nil, 0, dataErr
	}

	list := make([]*dashboard_api.FffRunningItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, toFffRunningItem(row))
	}
	return list, total, nil
}

func buildFffRunningWhere(param *biz.FffRunningParam) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if param.StartDt != "" && param.EndDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, param.StartDt, param.EndDt)
	} else if param.StartDt != "" {
		conds = append(conds, "dt >= ?")
		args = append(args, param.StartDt)
	} else if param.EndDt != "" {
		conds = append(conds, "dt <= ?")
		args = append(args, param.EndDt)
	} else {
		conds = append(conds, "dt = CURDATE()")
	}

	if param.FilterName != "" {
		conds = append(conds, "filter_name = ?")
		args = append(args, param.FilterName)
	}
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}

	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFffRunningItem(row *fffRunningRow) *dashboard_api.FffRunningItem {
	return &dashboard_api.FffRunningItem{
		Dt:              row.Dt.Format("2006-01-02"),
		FilterName:      row.FilterName,
		AnonymousId:     row.AnonymousId,
		TimestampUtc:    row.TimestampUtc.Format("2006-01-02 15:04:05"),
		CreateAt:        row.CreateAt.Format("2006-01-02"),
		CollectType:     row.CollectType,
		SwVersion:       row.SwVersion,
		ProjectName:     row.ProjectName,
		CarType:         row.CarType,
		VehicleSource:   row.VehicleSource,
		SwitchOn:        row.SwitchOn == 1,
		Version:         row.Version,
		OnAutopilot:     row.OnAutopilot == "true" || row.OnAutopilot == "1",
		FunctionMode:    row.FunctionMode,
		Status:          row.Status,
		FdiProjectName:  row.FdiProjectName,
		ProjectCarType:  row.ProjectCarType,
		VehicleSourceCn: row.VehicleSourceCn,
	}
}
