package data

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/wire"
	"golang.org/x/sync/singleflight"

	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

var DashboardFoProviderSet = wire.NewSet(NewFoDashboardRepo)

var _ biz.FoDashboardRepo = (*foDashboardRepo)(nil)

type foDashboardRepo struct {
	*baseRepo
	dimsSfg singleflight.Group
}

// foDimsCache 维度枚举缓存条目
type foDimsCache struct {
	dims       *biz.FoDimensions
	expiresAt  time.Time
	refreshing atomic.Bool
}

const foDimsCacheKey = "fo_dashboard:dims"
const foDimsCacheTTL = 60 * time.Minute

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

// fffTriggerRow dwd_cfdi_basic_fff_trigger 扫描结构
// before/after 是 SQL 保留字，SELECT 时需要反引号，GORM column tag 保持原名
type fffTriggerRow struct {
	Dt              time.Time `gorm:"column:dt"`
	Uuid            string    `gorm:"column:uuid"`
	EventName       string    `gorm:"column:event_name"`
	AnonymousId     string    `gorm:"column:anonymous_id"`
	TimestampUtc    time.Time `gorm:"column:timestamp_utc"`
	CreateAt        time.Time `gorm:"column:create_at"`
	TriggerTime     int64     `gorm:"column:trigger_time"`
	UtcDiffUs       int64     `gorm:"column:utc_diff_us"`
	Before          int       `gorm:"column:before"`
	After           int       `gorm:"column:after"`
	FilterName      string    `gorm:"column:filter_name"`
	TriggerType     string    `gorm:"column:trigger_type"`
	CollectType     string    `gorm:"column:collect_type"`
	Status          string    `gorm:"column:status"`
	OnAutopilot     string    `gorm:"column:on_autopilot"`
	FunctionMode    string    `gorm:"column:function_mode"`
	SwVersion       string    `gorm:"column:sw_version"`
	ProjectName     string    `gorm:"column:project_name"`
	CarType         string    `gorm:"column:car_type"`
	VehicleSource   string    `gorm:"column:vehicle_source"`
	Bj02Lat         string    `gorm:"column:bj02_lat"`
	Bj02Lon         string    `gorm:"column:bj02_lon"`
	RoadType        string    `gorm:"column:road_type"`
	FdiProjectName  string    `gorm:"column:fdi_project_name"`
	ProjectCarType  string    `gorm:"column:project_car_type"`
	VehicleSourceCn string    `gorm:"column:vehicle_source_cn"`
	Tags            string    `gorm:"column:tags"`
	Detail          string    `gorm:"column:detail"`
}

func (r *foDashboardRepo) ListFffTrigger(ctx context.Context, param *biz.FffTriggerParam) ([]*dashboard_api.FffTriggerItem, int64, error) {
	db := r.dorisDB(ctx)
	where, args := buildFffTriggerWhere(param)

	var (
		total    int64
		rows     []*fffTriggerRow
		countErr error
		dataErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countSQL := "SELECT COUNT(*) FROM dwd_cfdi_basic_fff_trigger" + where
		countErr = db.Raw(countSQL, args...).Scan(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (param.Page - 1) * param.PageSize
		// before/after 是 SQL 保留字，使用反引号转义
		dataSQL := fmt.Sprintf(
			"SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at, trigger_time, utc_diff_us,"+
				" `before`, `after`, filter_name, trigger_type, collect_type, status, on_autopilot, function_mode,"+
				" sw_version, project_name, car_type, vehicle_source, bj02_lat, bj02_lon, road_type,"+
				" fdi_project_name, project_car_type, vehicle_source_cn, tags, detail"+
				" FROM dwd_cfdi_basic_fff_trigger%s LIMIT %d OFFSET %d",
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

	list := make([]*dashboard_api.FffTriggerItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, toFffTriggerItem(row))
	}
	return list, total, nil
}

func buildFffTriggerWhere(param *biz.FffTriggerParam) (string, []interface{}) {
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
	if len(param.EventNames) == 1 {
		conds = append(conds, "event_name = ?")
		args = append(args, param.EventNames[0])
	} else if len(param.EventNames) > 1 {
		placeholders := strings.Repeat("?,", len(param.EventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range param.EventNames {
			args = append(args, e)
		}
	}
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFffTriggerItem(row *fffTriggerRow) *dashboard_api.FffTriggerItem {
	return &dashboard_api.FffTriggerItem{
		Dt:              row.Dt.Format("2006-01-02"),
		Uuid:            row.Uuid,
		EventName:       row.EventName,
		AnonymousId:     row.AnonymousId,
		TimestampUtc:    row.TimestampUtc.Format("2006-01-02 15:04:05.999999"),
		CreateAt:        row.CreateAt.Format("2006-01-02"),
		TriggerTime:     row.TriggerTime,
		UtcDiffUs:       row.UtcDiffUs,
		Before:          row.Before,
		After:           row.After,
		FilterName:      row.FilterName,
		TriggerType:     row.TriggerType,
		CollectType:     row.CollectType,
		Status:          row.Status,
		OnAutopilot:     row.OnAutopilot == "true" || row.OnAutopilot == "1",
		FunctionMode:    row.FunctionMode,
		SwVersion:       row.SwVersion,
		ProjectName:     row.ProjectName,
		CarType:         row.CarType,
		VehicleSource:   row.VehicleSource,
		Bj02Lat:         row.Bj02Lat,
		Bj02Lon:         row.Bj02Lon,
		RoadType:        row.RoadType,
		FdiProjectName:  row.FdiProjectName,
		ProjectCarType:  row.ProjectCarType,
		VehicleSourceCn: row.VehicleSourceCn,
		Tags:            row.Tags,
		Detail:          row.Detail,
	}
}

// fffCloseRow dwd_cfdi_basic_fff_close 扫描结构
type fffCloseRow struct {
	Dt              time.Time `gorm:"column:dt"`
	FilterName      string    `gorm:"column:filter_name"`
	Version         string    `gorm:"column:version"`
	Reason          string    `gorm:"column:reason"`
	AnonymousId     string    `gorm:"column:anonymous_id"`
	CreateAt        time.Time `gorm:"column:create_at"`
	SwVersion       string    `gorm:"column:sw_version"`
	TimestampUtc    time.Time `gorm:"column:timestamp_utc"`
	ProjectName     string    `gorm:"column:project_name"`
	CarType         string    `gorm:"column:car_type"`
	VehicleSource   string    `gorm:"column:vehicle_source"`
	FdiProjectName  string    `gorm:"column:fdi_project_name"`
	ProjectCarType  string    `gorm:"column:project_car_type"`
	VehicleSourceCn string    `gorm:"column:vehicle_source_cn"`
}

func (r *foDashboardRepo) ListFffClose(ctx context.Context, param *biz.FffCloseParam) ([]*dashboard_api.FffCloseItem, int64, error) {
	db := r.dorisDB(ctx)
	where, args := buildFffCloseWhere(param)

	var (
		total    int64
		rows     []*fffCloseRow
		countErr error
		dataErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countSQL := "SELECT COUNT(*) FROM dwd_cfdi_basic_fff_close" + where
		countErr = db.Raw(countSQL, args...).Scan(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (param.Page - 1) * param.PageSize
		dataSQL := fmt.Sprintf(
			`SELECT dt, filter_name, version, reason, anonymous_id, create_at,
			sw_version, timestamp_utc, project_name, car_type, vehicle_source,
			fdi_project_name, project_car_type, vehicle_source_cn
			FROM dwd_cfdi_basic_fff_close%s LIMIT %d OFFSET %d`,
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

	list := make([]*dashboard_api.FffCloseItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, toFffCloseItem(row))
	}
	return list, total, nil
}

func buildFffCloseWhere(param *biz.FffCloseParam) (string, []interface{}) {
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

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFffCloseItem(row *fffCloseRow) *dashboard_api.FffCloseItem {
	return &dashboard_api.FffCloseItem{
		Dt:              row.Dt.Format("2006-01-02"),
		FilterName:      row.FilterName,
		Version:         row.Version,
		Reason:          row.Reason,
		AnonymousId:     row.AnonymousId,
		CreateAt:        row.CreateAt.Format("2006-01-02"),
		SwVersion:       row.SwVersion,
		TimestampUtc:    row.TimestampUtc.Format("2006-01-02 15:04:05"),
		ProjectName:     row.ProjectName,
		CarType:         row.CarType,
		VehicleSource:   row.VehicleSource,
		FdiProjectName:  row.FdiProjectName,
		ProjectCarType:  row.ProjectCarType,
		VehicleSourceCn: row.VehicleSourceCn,
	}
}

// fdrTriggerRow dwd_basic_fdr_trigger 扫描结构
// type 是 Go 关键字，用 RecordType 接收
type fdrTriggerRow struct {
	Dt                time.Time `gorm:"column:dt"`
	Uuid              string    `gorm:"column:uuid"`
	EventName         string    `gorm:"column:event_name"`
	AnonymousId       string    `gorm:"column:anonymous_id"`
	TimestampUtc      time.Time `gorm:"column:timestamp_utc"`
	CreateAt          time.Time `gorm:"column:create_at"`
	SwVersion         string    `gorm:"column:sw_version"`
	Dse               string    `gorm:"column:dse"`
	TdMb              string    `gorm:"column:td_mb"`
	TmMb              string    `gorm:"column:tm_mb"`
	TriggerTimestamp  int64     `gorm:"column:trigger_timestamp"`
	BeginTimestampUts int64     `gorm:"column:begin_timestamp_uts"`
	EndTimestampUts   int64     `gorm:"column:end_timestamp_uts"`
	DumpTimestamp     int64     `gorm:"column:dump_timestamp"`
	Status            string    `gorm:"column:status"`
	Detail            string    `gorm:"column:detail"`
	TimeCostMs        string    `gorm:"column:time_cost_ms"`
	RecordType        string    `gorm:"column:type"`
	ProjectName       string    `gorm:"column:project_name"`
	CarType           string    `gorm:"column:car_type"`
	VehicleSource     string    `gorm:"column:vehicle_source"`
	FdiProjectName    string    `gorm:"column:fdi_project_name"`
	ProjectCarType    string    `gorm:"column:project_car_type"`
	VehicleSourceCn   string    `gorm:"column:vehicle_source_cn"`
}

func (r *foDashboardRepo) ListFdrTrigger(ctx context.Context, param *biz.FdrTriggerParam) ([]*dashboard_api.FdrTriggerItem, int64, error) {
	db := r.dorisDB(ctx)
	where, args := buildFdrTriggerWhere(param)

	var (
		total    int64
		rows     []*fdrTriggerRow
		countErr error
		dataErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countSQL := "SELECT COUNT(*) FROM dwd_basic_fdr_trigger" + where
		countErr = db.Raw(countSQL, args...).Scan(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (param.Page - 1) * param.PageSize
		dataSQL := fmt.Sprintf(
			`SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at, sw_version, dse,
			td_mb, tm_mb, trigger_timestamp, begin_timestamp_uts, end_timestamp_uts, dump_timestamp,
			status, detail, time_cost_ms, type, project_name, car_type, vehicle_source,
			fdi_project_name, project_car_type, vehicle_source_cn
			FROM dwd_basic_fdr_trigger%s LIMIT %d OFFSET %d`,
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

	list := make([]*dashboard_api.FdrTriggerItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, toFdrTriggerItem(row))
	}
	return list, total, nil
}

func buildFdrTriggerWhere(param *biz.FdrTriggerParam) (string, []interface{}) {
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
	if len(param.EventNames) == 1 {
		conds = append(conds, "event_name = ?")
		args = append(args, param.EventNames[0])
	} else if len(param.EventNames) > 1 {
		placeholders := strings.Repeat("?,", len(param.EventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range param.EventNames {
			args = append(args, e)
		}
	}
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFdrTriggerItem(row *fdrTriggerRow) *dashboard_api.FdrTriggerItem {
	return &dashboard_api.FdrTriggerItem{
		Dt:                row.Dt.Format("2006-01-02"),
		Uuid:              row.Uuid,
		EventName:         row.EventName,
		AnonymousId:       row.AnonymousId,
		TimestampUtc:      row.TimestampUtc.Format("2006-01-02 15:04:05"),
		CreateAt:          row.CreateAt.Format("2006-01-02"),
		SwVersion:         row.SwVersion,
		Dse:               row.Dse,
		TdMb:              row.TdMb,
		TmMb:              row.TmMb,
		TriggerTimestamp:  row.TriggerTimestamp,
		BeginTimestampUts: row.BeginTimestampUts,
		EndTimestampUts:   row.EndTimestampUts,
		DumpTimestamp:     row.DumpTimestamp,
		Status:            row.Status,
		Detail:            row.Detail,
		TimeCostMs:        row.TimeCostMs,
		RecordType:        row.RecordType,
		ProjectName:       row.ProjectName,
		CarType:           row.CarType,
		VehicleSource:     row.VehicleSource,
		FdiProjectName:    row.FdiProjectName,
		ProjectCarType:    row.ProjectCarType,
		VehicleSourceCn:   row.VehicleSourceCn,
	}
}

// fclTriggerRow dwd_cfdi_basic_fcl_trigger 扫描结构
type fclTriggerRow struct {
	Dt              time.Time `gorm:"column:dt"`
	Uuid            string    `gorm:"column:uuid"`
	EventName       string    `gorm:"column:event_name"`
	AnonymousId     string    `gorm:"column:anonymous_id"`
	TimestampUtc    time.Time `gorm:"column:timestamp_utc"`
	CreateAt        time.Time `gorm:"column:create_at"`
	Status          string    `gorm:"column:status"`
	Detail          string    `gorm:"column:detail"`
	CompletePercent int       `gorm:"column:complete_percent"`
	LocalFile       string    `gorm:"column:local_file"`
	UploadFailTimes int       `gorm:"column:upload_fail_times"`
	PrefixStitch    string    `gorm:"column:prefix_stitch"`
	TriggerSource   string    `gorm:"column:trigger_source"`
	SwVersion       string    `gorm:"column:sw_version"`
	ProjectName     string    `gorm:"column:project_name"`
	CarType         string    `gorm:"column:car_type"`
	VehicleSource   string    `gorm:"column:vehicle_source"`
	FdiProjectName  string    `gorm:"column:fdi_project_name"`
	ProjectCarType  string    `gorm:"column:project_car_type"`
	VehicleSourceCn string    `gorm:"column:vehicle_source_cn"`
}

func (r *foDashboardRepo) ListFclTrigger(ctx context.Context, param *biz.FclTriggerParam) ([]*dashboard_api.FclTriggerItem, int64, error) {
	db := r.dorisDB(ctx)
	where, args := buildFclTriggerWhere(param)

	var (
		total    int64
		rows     []*fclTriggerRow
		countErr error
		dataErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countSQL := "SELECT COUNT(*) FROM dwd_cfdi_basic_fcl_trigger" + where
		countErr = db.Raw(countSQL, args...).Scan(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (param.Page - 1) * param.PageSize
		dataSQL := fmt.Sprintf(
			`SELECT dt, uuid, event_name, anonymous_id, timestamp_utc, create_at,
			status, detail, complete_percent, local_file, upload_fail_times, prefix_stitch,
			trigger_source, sw_version, project_name, car_type, vehicle_source,
			fdi_project_name, project_car_type, vehicle_source_cn
			FROM dwd_cfdi_basic_fcl_trigger%s LIMIT %d OFFSET %d`,
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

	list := make([]*dashboard_api.FclTriggerItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, toFclTriggerItem(row))
	}
	return list, total, nil
}

func buildFclTriggerWhere(param *biz.FclTriggerParam) (string, []interface{}) {
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
	if len(param.EventNames) == 1 {
		conds = append(conds, "event_name = ?")
		args = append(args, param.EventNames[0])
	} else if len(param.EventNames) > 1 {
		placeholders := strings.Repeat("?,", len(param.EventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range param.EventNames {
			args = append(args, e)
		}
	}
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFclTriggerItem(row *fclTriggerRow) *dashboard_api.FclTriggerItem {
	return &dashboard_api.FclTriggerItem{
		Dt:              row.Dt.Format("2006-01-02"),
		Uuid:            row.Uuid,
		EventName:       row.EventName,
		AnonymousId:     row.AnonymousId,
		TimestampUtc:    row.TimestampUtc.Format("2006-01-02 15:04:05"),
		CreateAt:        row.CreateAt.Format("2006-01-02"),
		Status:          row.Status,
		Detail:          row.Detail,
		CompletePercent: row.CompletePercent,
		LocalFile:       row.LocalFile,
		UploadFailTimes: row.UploadFailTimes,
		PrefixStitch:    row.PrefixStitch,
		TriggerSource:   row.TriggerSource,
		SwVersion:       row.SwVersion,
		ProjectName:     row.ProjectName,
		CarType:         row.CarType,
		VehicleSource:   row.VehicleSource,
		FdiProjectName:  row.FdiProjectName,
		ProjectCarType:  row.ProjectCarType,
		VehicleSourceCn: row.VehicleSourceCn,
	}
}

// uuidDetailRow dwd_cfdi_status_monitor_analysis 扫描结构
// timestamp_utc 可为 NULL，用 *time.Time 接收
type uuidDetailRow struct {
	Dt                time.Time `gorm:"column:dt"`
	AnonymousId       string    `gorm:"column:anonymous_id"`
	EventName         string    `gorm:"column:event_name"`
	Uuid              string    `gorm:"column:uuid"`
	CreateAt          time.Time `gorm:"column:create_at"`
	FilterName        string    `gorm:"column:filter_name"`
	FffSwVersion      string    `gorm:"column:fff_sw_version"`
	FdrSwVersion      string    `gorm:"column:fdr_sw_version"`
	FclSwVersion      string    `gorm:"column:fcl_sw_version"`
	TriggerType       string    `gorm:"column:trigger_type"`
	CollectType       string    `gorm:"column:collect_type"`
	FffUpdatedAt      int64     `gorm:"column:fff_updated_at"`
	FdrUpdatedAt      int64     `gorm:"column:fdr_updated_at"`
	FclUpdatedAt      int64     `gorm:"column:fcl_updated_at"`
	FffStatus         string    `gorm:"column:fff_status"`
	FdrStatus         string    `gorm:"column:fdr_status"`
	FclStatus         string    `gorm:"column:fcl_status"`
	FffDetail         string    `gorm:"column:fff_detail"`
	FdrDetail         string    `gorm:"column:fdr_detail"`
	FclDetail         string    `gorm:"column:fcl_detail"`
	BeginTimestampUts int64     `gorm:"column:begin_timestamp_uts"`
	DumpTimestamp     int64     `gorm:"column:dump_timestamp"`
	EndTimestampUts   int64     `gorm:"column:end_timestamp_uts"`
	Md5               string    `gorm:"column:md5"`
	BagName           string    `gorm:"column:bag_name"`
	CompletePercent   int       `gorm:"column:complete_percent"`
	ProjectName       string    `gorm:"column:project_name"`
	CarType           string    `gorm:"column:car_type"`
	VehicleSource     string    `gorm:"column:vehicle_source"`
	ProjectCarType    string    `gorm:"column:project_car_type"`
	Dse               string    `gorm:"column:dse"`
}

func (r *foDashboardRepo) ListUuidDetail(ctx context.Context, param *biz.UuidDetailParam) ([]*dashboard_api.UuidDetailItem, int64, error) {
	db := r.dorisDB(ctx)
	where, args := buildUuidDetailWhere(param)

	var (
		total    int64
		rows     []*uuidDetailRow
		countErr error
		dataErr  error
	)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		countSQL := "SELECT COUNT(*) FROM dwd_cfdi_status_monitor_analysis" + where
		countErr = db.Raw(countSQL, args...).Scan(&total).Error
	}()

	go func() {
		defer wg.Done()
		offset := (param.Page - 1) * param.PageSize
		dataSQL := fmt.Sprintf(
			`SELECT dt, anonymous_id, event_name, uuid, create_at, filter_name,
			fff_sw_version, fdr_sw_version, fcl_sw_version, trigger_type, collect_type,
			fff_updated_at, fdr_updated_at, fcl_updated_at,
			fff_status, fdr_status, fcl_status, fff_detail, fdr_detail, fcl_detail,
			begin_timestamp_uts, dump_timestamp, end_timestamp_uts,
			md5, bag_name, complete_percent, project_name, car_type, vehicle_source,
			project_car_type, dse
			FROM dwd_cfdi_status_monitor_analysis%s LIMIT %d OFFSET %d`,
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

	list := make([]*dashboard_api.UuidDetailItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, toUuidDetailItem(row))
	}
	return list, total, nil
}

func buildUuidDetailWhere(param *biz.UuidDetailParam) (string, []interface{}) {
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
	if len(param.EventNames) == 1 {
		conds = append(conds, "event_name = ?")
		args = append(args, param.EventNames[0])
	} else if len(param.EventNames) > 1 {
		placeholders := strings.Repeat("?,", len(param.EventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range param.EventNames {
			args = append(args, e)
		}
	}
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	if param.OnlyFail {
		conds = append(conds, "fcl_status != 'success'")
	}
	switch param.StageFilter {
	case "fff_discard":
		conds = append(conds, "fff_status = 'discard'")
	case "fdr_discard":
		conds = append(conds, "fff_status = 'success'", "fdr_status = 'discard'")
	case "fcl_discard":
		conds = append(conds, "fdr_status = 'success'", "fcl_status != 'success'", "fcl_status != ''")
	case "fcl_success":
		conds = append(conds, "fcl_status = 'success'")
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toUuidDetailItem(row *uuidDetailRow) *dashboard_api.UuidDetailItem {

	return &dashboard_api.UuidDetailItem{
		Dt:                row.Dt.Format("2006-01-02"),
		AnonymousId:       row.AnonymousId,
		EventName:         row.EventName,
		Uuid:              row.Uuid,
		CreateAt:          row.CreateAt.Format("2006-01-02"),
		FilterName:        row.FilterName,
		FffSwVersion:      row.FffSwVersion,
		FdrSwVersion:      row.FdrSwVersion,
		FclSwVersion:      row.FclSwVersion,
		TriggerType:       row.TriggerType,
		CollectType:       row.CollectType,
		FffUpdatedAt:      row.FffUpdatedAt,
		FdrUpdatedAt:      row.FdrUpdatedAt,
		FclUpdatedAt:      row.FclUpdatedAt,
		FffStatus:         row.FffStatus,
		FdrStatus:         row.FdrStatus,
		FclStatus:         row.FclStatus,
		FffDetail:         row.FffDetail,
		FdrDetail:         row.FdrDetail,
		FclDetail:         row.FclDetail,
		BeginTimestampUts: row.BeginTimestampUts,
		DumpTimestamp:     row.DumpTimestamp,
		EndTimestampUts:   row.EndTimestampUts,
		Md5:               row.Md5,
		BagName:           row.BagName,
		CompletePercent:   row.CompletePercent,
		ProjectName:       row.ProjectName,
		CarType:           row.CarType,
		VehicleSource:     row.VehicleSource,
		ProjectCarType:    row.ProjectCarType,
		Dse:               row.Dse,
	}
}

// closeReasonRow 算子关闭原因聚合结果扫描结构
type closeReasonRow struct {
	Category string `gorm:"column:category"`
	Cnt      int64  `gorm:"column:cnt"`
}

func (r *foDashboardRepo) GetCloseReason(ctx context.Context, param *biz.CloseReasonParam) ([]*dashboard_api.CloseReasonItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildCloseReasonWhere(param)

	sql := fmt.Sprintf(`
		SELECT
		  CASE
		    WHEN INSTR(reason, 'out of memory')  > 0 THEN 'out of memory'
			WHEN INSTR(reason, 'not enough memory')  > 0 THEN 'not enough memory'
		    WHEN INSTR(reason, 'close operator')      > 0 THEN 'close operator for crash'
		    WHEN INSTR(reason, 'with error')     > 0 THEN 'with error'
		    ELSE '其他'
		  END AS category,
		  COUNT(*) AS cnt
		FROM dwd_cfdi_basic_fff_close%s
		GROUP BY category
		ORDER BY cnt DESC`, where)

	var rows []*closeReasonRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*dashboard_api.CloseReasonItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &dashboard_api.CloseReasonItem{
			Name:  row.Category,
			Value: row.Cnt,
		})
	}
	return list, nil
}

func buildCloseReasonWhere(param *biz.CloseReasonParam) (string, []interface{}) {
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

	return " WHERE " + strings.Join(conds, " AND "), args
}

// stageTrendRow 三阶段趋势聚合结果（三张表共用同一扫描结构）
type stageTrendRow struct {
	Dt       time.Time `gorm:"column:dt"`
	Success  int64     `gorm:"column:success"`
	Cat2     int64     `gorm:"column:cat2"`
	Cat3     int64     `gorm:"column:cat3"`
	Cat4     int64     `gorm:"column:cat4"`
	Cat5     int64     `gorm:"column:cat5"`
	CatOther int64     `gorm:"column:cat_other"`
}

// funnelFffRow FFF trigger 单行聚合结果
type funnelFffRow struct {
	Total int64 `gorm:"column:total"`
	Allow int64 `gorm:"column:allow"`
	R1    int64 `gorm:"column:r1"` // 冷却丢弃
	R2    int64 `gorm:"column:r2"` // DRM Quota
	R3    int64 `gorm:"column:r3"` // 触发上限
	R4    int64 `gorm:"column:r4"` // 数采限制
	R5    int64 `gorm:"column:r5"` // 其他丢弃
}

// funnelStageRow FDR/FCL 单行聚合结果
type funnelStageRow struct {
	StageSuccess int64 `gorm:"column:stage_success"`
	R1           int64 `gorm:"column:r1"`
	R2           int64 `gorm:"column:r2"`
	R3           int64 `gorm:"column:r3"`
	R4           int64 `gorm:"column:r4"`
	R5           int64 `gorm:"column:r5"`
}

func (r *foDashboardRepo) GetFunnel(ctx context.Context, param *biz.FunnelParam) (*biz.FunnelData, error) {
	db := r.dorisDB(ctx)

	// 复用 stage_trend 的 WHERE 构建函数，FunnelParam 和 StageTrendParam 字段相同
	stageParam := &biz.StageTrendParam{
		FilterName:  param.FilterName,
		EventNames:  param.EventNames,
		ProjectName: param.ProjectName,
		StartDt:     param.StartDt,
		EndDt:       param.EndDt,
	}
	fffWhere, fffArgs := buildStageFffWhere(stageParam)
	comWhere, comArgs := buildStageCommonWhere(stageParam)

	fffSQL := `SELECT
		COUNT(*) AS total,
		SUM(CASE WHEN status='success' THEN 1 ELSE 0 END) AS allow,
		SUM(CASE WHEN status='discard' AND detail='check_is_no_need_cooldown' THEN 1 ELSE 0 END) AS r1,
		SUM(CASE WHEN status='discard' AND detail IN ('check_drm_quota','check_drm_quota_weight') THEN 1 ELSE 0 END) AS r2,
		SUM(CASE WHEN status='discard' AND detail='check_not_reach_trigger_maximum' THEN 1 ELSE 0 END) AS r3,
		SUM(CASE WHEN status='discard' AND detail='check_need_acquire_data' THEN 1 ELSE 0 END) AS r4,
		SUM(CASE WHEN status='discard'
			AND detail NOT IN ('check_is_no_need_cooldown','check_drm_quota','check_drm_quota_weight',
			                   'check_not_reach_trigger_maximum','check_need_acquire_data')
			THEN 1 ELSE 0 END) AS r5
		FROM dwd_cfdi_basic_fff_trigger` + fffWhere

	fdrSQL := `SELECT
		SUM(CASE WHEN status='success' THEN 1 ELSE 0 END) AS stage_success,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'because of full gc')>0 THEN 1 ELSE 0 END) AS r1,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'do not recognized')>0 THEN 1 ELSE 0 END) AS r2,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'mem pool water line')>0 THEN 1 ELSE 0 END) AS r3,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'Bag invalid')>0 THEN 1 ELSE 0 END) AS r4,
		SUM(CASE WHEN status='discard'
			AND INSTR(detail,'because of full gc')=0 AND INSTR(detail,'do not recognized')=0
			AND INSTR(detail,'mem pool water line')=0 AND INSTR(detail,'Bag invalid')=0
			THEN 1 ELSE 0 END) AS r5
		FROM dwd_basic_fdr_trigger` + comWhere

	fclSQL := `SELECT
		COUNT(DISTINCT CASE WHEN status='success' THEN concat_ws('|', anonymous_id, local_file) end) AS stage_success,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'geofence forbidden')>0 THEN 1 ELSE 0 END) AS r1,
		SUM(CASE WHEN status='discard' AND (INSTR(detail,'bag not exist')>0 OR INSTR(detail,'meta file lost')>0) THEN 1 ELSE 0 END) AS r2,
		SUM(CASE WHEN status='discard' AND (INSTR(detail,'reach upload limit')>0 OR INSTR(detail,'Filter quota exceeded')>0) THEN 1 ELSE 0 END) AS r3,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'EventName is in blacklist')>0 THEN 1 ELSE 0 END) AS r4,
		SUM(CASE WHEN status='discard'
			AND INSTR(detail,'geofence forbidden')=0 AND INSTR(detail,'bag not exist')=0
			AND INSTR(detail,'meta file lost')=0 AND INSTR(detail,'reach upload limit')=0
			AND INSTR(detail,'Filter quota exceeded')=0 AND INSTR(detail,'EventName is in blacklist')=0
			THEN 1 ELSE 0 END) AS r5
		FROM dwd_cfdi_basic_fcl_trigger` + comWhere + ` AND trigger_source != 'Forever_log'`

	var (
		fffRow funnelFffRow
		fdrRow funnelStageRow
		fclRow funnelStageRow
		fffErr error
		fdrErr error
		fclErr error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); fffErr = db.Raw(fffSQL, fffArgs...).Scan(&fffRow).Error }()
	go func() { defer wg.Done(); fdrErr = db.Raw(fdrSQL, comArgs...).Scan(&fdrRow).Error }()
	go func() { defer wg.Done(); fclErr = db.Raw(fclSQL, comArgs...).Scan(&fclRow).Error }()
	wg.Wait()

	if fffErr != nil {
		return nil, fffErr
	}
	if fdrErr != nil {
		return nil, fdrErr
	}
	if fclErr != nil {
		return nil, fclErr
	}

	cfdiRate := 0.0
	if fffRow.Total > 0 {
		cfdiRate = math.Round(float64(fclRow.StageSuccess)/float64(fffRow.Total)*1000) / 10
	}

	mkReasons := func(names []string, counts []int64) []*dashboard_api.FunnelFailReason {
		list := make([]*dashboard_api.FunnelFailReason, 0, len(names))
		for i, name := range names {
			if i < len(counts) && counts[i] > 0 {
				list = append(list, &dashboard_api.FunnelFailReason{Name: name, Count: counts[i]})
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Count > list[j].Count })
		return list
	}

	return &biz.FunnelData{
		Stat: &dashboard_api.FunnelStat{
			FffTotal:   fffRow.Total,
			FffAllow:   fffRow.Allow,
			FdrSuccess: fdrRow.StageSuccess,
			FdrFail:    fdrRow.R1 + fdrRow.R2 + fdrRow.R3 + fdrRow.R4 + fdrRow.R5,
			FclSuccess: fclRow.StageSuccess,
			FclFail:    fclRow.R1 + fclRow.R2 + fclRow.R3 + fclRow.R4 + fclRow.R5,
			CfdiRate:   cfdiRate,
		},
		FffFail: mkReasons(
			[]string{"冷却丢弃", "DRM Quota", "触发上限", "数采限制", "其他丢弃"},
			[]int64{fffRow.R1, fffRow.R2, fffRow.R3, fffRow.R4, fffRow.R5},
		),
		FdrFail: mkReasons(
			[]string{"Full GC", "事件不识别", "内存限制", "Bag Invalid", "其他丢弃"},
			[]int64{fdrRow.R1, fdrRow.R2, fdrRow.R3, fdrRow.R4, fdrRow.R5},
		),
		FclFail: mkReasons(
			[]string{"Geofence限制", "bag/meta丢失", "上传Quota", "事件黑名单", "其他丢弃"},
			[]int64{fclRow.R1, fclRow.R2, fclRow.R3, fclRow.R4, fclRow.R5},
		),
	}, nil
}

func (r *foDashboardRepo) GetStageTrend(ctx context.Context, param *biz.StageTrendParam) (*biz.StageTrendData, error) {
	db := r.dorisDB(ctx)

	fffWhere, fffArgs := buildStageFffWhere(param)
	comWhere, comArgs := buildStageCommonWhere(param) // FDR/FCL 无 filter_name 列

	fffSQL := `SELECT dt,
		SUM(CASE WHEN status='success' THEN 1 ELSE 0 END) AS success,
		SUM(CASE WHEN status='discard' AND detail='check_is_no_need_cooldown' THEN 1 ELSE 0 END) AS cat2,
		SUM(CASE WHEN status='discard' AND detail IN ('check_drm_quota','check_drm_quota_weight') THEN 1 ELSE 0 END) AS cat3,
		SUM(CASE WHEN status='discard' AND detail='check_not_reach_trigger_maximum' THEN 1 ELSE 0 END) AS cat4,
		SUM(CASE WHEN status='discard' AND detail='check_need_acquire_data' THEN 1 ELSE 0 END) AS cat5,
		SUM(CASE WHEN status='discard'
			AND detail NOT IN ('check_is_no_need_cooldown','check_drm_quota','check_drm_quota_weight',
			                   'check_not_reach_trigger_maximum','check_need_acquire_data')
			THEN 1 ELSE 0 END) AS cat_other
		FROM dwd_cfdi_basic_fff_trigger` + fffWhere + ` GROUP BY dt ORDER BY dt ASC`

	fdrSQL := `SELECT dt,
		SUM(CASE WHEN status='success' THEN 1 ELSE 0 END) AS success,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'because of full gc')>0 THEN 1 ELSE 0 END) AS cat2,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'do not recognized')>0 THEN 1 ELSE 0 END) AS cat3,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'mem pool water line')>0 THEN 1 ELSE 0 END) AS cat4,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'Bag invalid')>0 THEN 1 ELSE 0 END) AS cat5,
		SUM(CASE WHEN status='discard'
			AND INSTR(detail,'because of full gc')=0 AND INSTR(detail,'do not recognized')=0
			AND INSTR(detail,'mem pool water line')=0 AND INSTR(detail,'Bag invalid')=0
			THEN 1 ELSE 0 END) AS cat_other
		FROM dwd_basic_fdr_trigger` + comWhere + ` GROUP BY dt ORDER BY dt ASC`

	fclSQL := `SELECT dt,
		COUNT(DISTINCT CASE WHEN status='success' THEN concat_ws('|', anonymous_id, local_file) end) AS Success,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'geofence forbidden')>0 THEN 1 ELSE 0 END) AS cat2,
		SUM(CASE WHEN status='discard' AND (INSTR(detail,'bag not exist')>0 OR INSTR(detail,'meta file lost')>0) THEN 1 ELSE 0 END) AS cat3,
		SUM(CASE WHEN status='discard' AND (INSTR(detail,'reach upload limit')>0 OR INSTR(detail,'Filter quota exceeded')>0) THEN 1 ELSE 0 END) AS cat4,
		SUM(CASE WHEN status='discard' AND INSTR(detail,'EventName is in blacklist')>0 THEN 1 ELSE 0 END) AS cat5,
		SUM(CASE WHEN status='discard'
			AND INSTR(detail,'geofence forbidden')=0 AND INSTR(detail,'bag not exist')=0
			AND INSTR(detail,'meta file lost')=0 AND INSTR(detail,'reach upload limit')=0
			AND INSTR(detail,'Filter quota exceeded')=0 AND INSTR(detail,'EventName is in blacklist')=0
			THEN 1 ELSE 0 END) AS cat_other
		FROM dwd_cfdi_basic_fcl_trigger` + comWhere + ` AND trigger_source != 'Forever_log' GROUP BY dt ORDER BY dt ASC`

	var (
		fffRows []*stageTrendRow
		fdrRows []*stageTrendRow
		fclRows []*stageTrendRow
		fffErr  error
		fdrErr  error
		fclErr  error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); fffErr = db.Raw(fffSQL, fffArgs...).Scan(&fffRows).Error }()
	go func() { defer wg.Done(); fdrErr = db.Raw(fdrSQL, comArgs...).Scan(&fdrRows).Error }()
	go func() { defer wg.Done(); fclErr = db.Raw(fclSQL, comArgs...).Scan(&fclRows).Error }()
	wg.Wait()

	if fffErr != nil {
		return nil, fffErr
	}
	if fdrErr != nil {
		return nil, fdrErr
	}
	if fclErr != nil {
		return nil, fclErr
	}

	dates, fffMap, fdrMap, fclMap := alignStageDates(fffRows, fdrRows, fclRows)

	return &biz.StageTrendData{
		Dates: dates,
		Fff:   buildStageSeries(dates, fffMap, []string{"FFF 成功", "冷却丢弃", "DRM Quota", "触发上限", "数采限制", "其他丢弃"}),
		Fdr:   buildStageSeries(dates, fdrMap, []string{"FDR 成功", "Full GC", "事件不识别", "内存限制", "Bag Invalid", "其他丢弃"}),
		Fcl:   buildStageSeries(dates, fclMap, []string{"FCL 成功", "Geofence限制", "bag/meta丢失", "上传Quota", "事件黑名单", "其他丢弃"}),
	}, nil
}

func alignStageDates(fffRows, fdrRows, fclRows []*stageTrendRow) (
	dates []string,
	fffMap, fdrMap, fclMap map[string]*stageTrendRow,
) {
	dateSet := map[string]bool{}
	fffMap = map[string]*stageTrendRow{}
	fdrMap = map[string]*stageTrendRow{}
	fclMap = map[string]*stageTrendRow{}

	for _, row := range fffRows {
		dt := row.Dt.Format("2006-01-02")
		dateSet[dt] = true
		fffMap[dt] = row
	}
	for _, row := range fdrRows {
		dt := row.Dt.Format("2006-01-02")
		dateSet[dt] = true
		fdrMap[dt] = row
	}
	for _, row := range fclRows {
		dt := row.Dt.Format("2006-01-02")
		dateSet[dt] = true
		fclMap[dt] = row
	}
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return
}

func buildStageSeries(dates []string, rowMap map[string]*stageTrendRow, names []string) []*dashboard_api.StageTrendSeries {
	series := make([]*dashboard_api.StageTrendSeries, len(names))
	for i, name := range names {
		series[i] = &dashboard_api.StageTrendSeries{Name: name, Data: make([]int64, len(dates))}
	}
	for i, dt := range dates {
		row, ok := rowMap[dt]
		if !ok {
			continue
		}
		vals := []int64{row.Success, row.Cat2, row.Cat3, row.Cat4, row.Cat5, row.CatOther}
		for j, v := range vals {
			if j < len(series) {
				series[j].Data[i] = v
			}
		}
	}
	return series
}

func buildStageFffWhere(param *biz.StageTrendParam) (string, []interface{}) {
	var conds []string
	var args []interface{}
	conds, args = appendStageDateCond(conds, args, param.StartDt, param.EndDt)
	if param.FilterName != "" {
		conds = append(conds, "filter_name = ?")
		args = append(args, param.FilterName)
	}
	conds, args = appendStageEventCond(conds, args, param.EventNames)
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func buildStageCommonWhere(param *biz.StageTrendParam) (string, []interface{}) {
	var conds []string
	var args []interface{}
	conds, args = appendStageDateCond(conds, args, param.StartDt, param.EndDt)
	conds, args = appendStageEventCond(conds, args, param.EventNames)
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func appendStageDateCond(conds []string, args []interface{}, startDt, endDt string) ([]string, []interface{}) {
	if startDt != "" && endDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, startDt, endDt)
	} else {
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}
	return conds, args
}

func appendStageEventCond(conds []string, args []interface{}, eventNames []string) ([]string, []interface{}) {
	if len(eventNames) == 1 {
		conds = append(conds, "event_name = ?")
		args = append(args, eventNames[0])
	} else if len(eventNames) > 1 {
		placeholders := strings.Repeat("?,", len(eventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range eventNames {
			args = append(args, e)
		}
	}
	return conds, args
}

// GetDimensions 查询 FO Dashboard 下拉维度，采用逻辑过期策略：
// 缓存存在时始终立即返回（即使已过期），过期后在后台异步刷新一次；
// 首次冷启动用 singleflight 保证只有一个请求打 Doris。
func (r *foDashboardRepo) GetDimensions(ctx context.Context) (*biz.FoDimensions, error) {
	if v, ok := r.data.dimCache.Load(foDimsCacheKey); ok {
		entry := v.(*foDimsCache)
		if time.Now().After(entry.expiresAt) && entry.refreshing.CompareAndSwap(false, true) {
			go func() {
				dims, err := r.fetchDimensions(context.Background())
				if err != nil {
					entry.refreshing.Store(false)
					return
				}
				r.data.dimCache.Store(foDimsCacheKey, &foDimsCache{
					dims:      dims,
					expiresAt: time.Now().Add(foDimsCacheTTL),
				})
			}()
		}
		return entry.dims, nil
	}

	// 冷启动：singleflight 保证并发请求只打一次 Doris
	v, err, _ := r.dimsSfg.Do(foDimsCacheKey, func() (interface{}, error) {
		// 等待 singleflight 期间可能已有其他请求写入缓存
		if cached, ok := r.data.dimCache.Load(foDimsCacheKey); ok {
			return cached.(*foDimsCache).dims, nil
		}
		dims, err := r.fetchDimensions(ctx)
		if err != nil {
			return nil, err
		}
		r.data.dimCache.Store(foDimsCacheKey, &foDimsCache{
			dims:      dims,
			expiresAt: time.Now().Add(foDimsCacheTTL),
		})
		return dims, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*biz.FoDimensions), nil
}

// fetchDimensions 并发查询 Doris 获取四类维度枚举值
func (r *foDashboardRepo) fetchDimensions(ctx context.Context) (*biz.FoDimensions, error) {
	db := r.dorisDB(ctx)

	type strRow struct{ Val string }

	var (
		filterNames  []string
		projectNames []string
		eventNames   []string
		carTypes     []string
		errFilter    error
		errProject   error
		errEvent     error
		errCarType   error
	)

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		var rows []strRow
		errFilter = db.Raw(`SELECT DISTINCT filter_name AS val
			FROM dwd_cfdi_basic_fff_running
			WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
			ORDER BY val`).Scan(&rows).Error
		for _, r := range rows {
			filterNames = append(filterNames, r.Val)
		}
	}()

	go func() {
		defer wg.Done()
		var rows []strRow
		errProject = db.Raw(`SELECT DISTINCT project_name AS val
			FROM dwd_cfdi_basic_fff_running
			WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
			AND project_name IS NOT NULL
			ORDER BY val`).Scan(&rows).Error
		for _, r := range rows {
			projectNames = append(projectNames, r.Val)
		}
	}()

	go func() {
		defer wg.Done()
		var rows []strRow
		errEvent = db.Raw(`SELECT DISTINCT event_name AS val
			FROM dwd_cfdi_basic_fff_trigger
			WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
			ORDER BY val`).Scan(&rows).Error
		for _, r := range rows {
			eventNames = append(eventNames, r.Val)
		}
	}()

	go func() {
		defer wg.Done()
		var rows []strRow
		errCarType = db.Raw(`SELECT DISTINCT car_type AS val
			FROM dwd_cfdi_basic_fff_running
			WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
			AND car_type IS NOT NULL
			ORDER BY val`).Scan(&rows).Error
		for _, r := range rows {
			carTypes = append(carTypes, r.Val)
		}
	}()

	wg.Wait()

	if errFilter != nil {
		return nil, errFilter
	}
	if errProject != nil {
		return nil, errProject
	}
	if errEvent != nil {
		return nil, errEvent
	}
	if errCarType != nil {
		return nil, errCarType
	}

	return &biz.FoDimensions{
		FilterNames:  filterNames,
		EventNames:   eventNames,
		ProjectNames: projectNames,
		CarTypes:     carTypes,
	}, nil
}
