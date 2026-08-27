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
	"gorm.io/gorm"

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

func (r *foDashboardRepo) ListFffRunning(ctx context.Context, param *biz.FffRunningParam) ([]*biz.FffRunningItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

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

	list := make([]*biz.FffRunningItem, 0, len(rows))
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
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	if len(param.AnonymousIds) > 0 {
		ph := strings.Repeat("?,", len(param.AnonymousIds))
		ph = ph[:len(ph)-1]
		conds = append(conds, "anonymous_id IN ("+ph+")")
		for _, a := range param.AnonymousIds {
			args = append(args, a)
		}
	}
	conds, args = appendFffRunningEventNamesAsFilterNames(conds, args, param.EventNames)

	if len(conds) == 0 {
		return "", args
	}
	if param.SwitchOn != nil {
		conds = append(conds, "switch_on = ?")
		args = append(args, *param.SwitchOn)
	}
	if param.SwVersion != "" {
		conds = append(conds, "sw_version = ?")
		args = append(args, param.SwVersion)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func appendFffRunningEventNamesAsFilterNames(conds []string, args []interface{}, rawEventNames []string) ([]string, []interface{}) {
	eventNames := nonAllValues(rawEventNames)
	if len(eventNames) == 0 {
		return conds, args
	}

	return appendMultiCond(conds, args, "filter_name", eventNames)
}

func buildFffRunningVehicleWhere(param *biz.FffRunningParam) (string, []interface{}) {
	var conds []string
	var args []interface{}

	// running 相关查询保留原口径：未传日期时默认今天
	if param.StartDt != "" && param.EndDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, param.StartDt, param.EndDt)
	} else {
		conds = append(conds, "dt = CURDATE()")
	}

	// _agg 表 event_name 列混存事件名(trigger 源)和筛选器名(running/close 源),
	// 前端统一传 event_name(可能是事件名也可能是筛选器名),直接查此列;
	// filter_name 旧参数保留兼容(语义同 event_name,都查 event_name 列)
	names := nonAllValues(param.EventNames)
	if param.FilterName != "" {
		names = append(names, param.FilterName)
	}
	if len(names) == 0 {
		// 未传事件/筛选器:必须只查 __ALL__ 汇总行。_agg 表按 project×car_type×event_name
		// 分组展开,明细行的车辆数是"分组内去重",全表 SUM 会把同一辆车在多个分组行
		// 重复计入(2026-08-26 实测:全表 SUM≈774w,__ALL__ 口径正确值≈28.6w)
		conds = append(conds, "event_name = ?")
		args = append(args, aggAllValue)
	} else {
		eventCond, eventArgs := buildAggEventCondition(names)
		conds = append(conds, eventCond)
		args = append(args, eventArgs...)
	}

	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	} else {
		// lianhuashan 心跳链路 anonymous_id 全部丢失(上报为 'empty'),
		// 车辆维度数据不可信(2026-08-26 实测:该项目 _agg 记 65 辆、真实车队远大于此),
		// 全局车辆总数口径剔除该项目;用户显式选了它仍如实返回,由前端标注数据异常
		conds = append(conds, "project_name <> 'lianhuashan'")
	}
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func nonAllValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && value != aggAllValue {
			result = append(result, value)
		}
	}
	return result
}

func toFffRunningItem(row *fffRunningRow) *biz.FffRunningItem {
	return &biz.FffRunningItem{
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

func (r *foDashboardRepo) ListFffTrigger(ctx context.Context, param *biz.FffTriggerParam) ([]*biz.FffTriggerItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
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

	list := make([]*biz.FffTriggerItem, 0, len(rows))
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
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	if len(param.AnonymousIds) > 0 {
		ph := strings.Repeat("?,", len(param.AnonymousIds))
		ph = ph[:len(ph)-1]
		conds = append(conds, "anonymous_id IN ("+ph+")")
		for _, a := range param.AnonymousIds {
			args = append(args, a)
		}
	}

	if param.Uuid != "" {
		conds = append(conds, "uuid = ?")
		args = append(args, param.Uuid)
	}
	if param.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, param.Status)
	}
	if param.TriggerType != "" {
		conds = append(conds, "trigger_type = ?")
		args = append(args, param.TriggerType)
	}
	if param.Tags != "" {
		conds = append(conds, "tags LIKE ?")
		args = append(args, "%"+param.Tags+"%")
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func buildFffTriggerReasonWhere(param *biz.FffTriggerParam) (string, []interface{}) {
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
		// 日期默认与全看板一致：未传时近 7 天
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}

	if len(param.EventNames) == 0 {
		conds = append(conds, "event_name = ?")
		args = append(args, aggAllValue)
	} else if len(param.EventNames) == 1 {
		conds = append(conds, "event_name = ?")
		args = append(args, param.EventNames[0])
	} else {
		placeholders := strings.Repeat("?,", len(param.EventNames))
		placeholders = placeholders[:len(placeholders)-1]
		conds = append(conds, "event_name IN ("+placeholders+")")
		for _, e := range param.EventNames {
			args = append(args, e)
		}
	}

	// filter_name 条件:不再强制 filter_name='__ALL__',也不再映射 event_names→filter_name。
	// reason 粒度的 filter_name 列实测全部是 __ALL__(9944 行无一例外):
	// - 传事件名 → event_name=? 命中 __ALL__ 行,有数据;若再拼 filter_name IN ('事件名')
	//   反而不命中 __ALL__ → 查空(上一版踩的坑)
	// - 传筛选器名 → event_name=? 不命中(reason 粒度没有该 event_name)→ 0,如实
	conds = append(conds, "filter_name = ?")
	args = append(args, aggAllValue)

	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFffTriggerItem(row *fffTriggerRow) *biz.FffTriggerItem {
	return &biz.FffTriggerItem{
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

func (r *foDashboardRepo) ListFffClose(ctx context.Context, param *biz.FffCloseParam) ([]*biz.FffCloseItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
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

	list := make([]*biz.FffCloseItem, 0, len(rows))
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
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	if len(param.AnonymousIds) > 0 {
		ph := strings.Repeat("?,", len(param.AnonymousIds))
		ph = ph[:len(ph)-1]
		conds = append(conds, "anonymous_id IN ("+ph+")")
		for _, a := range param.AnonymousIds {
			args = append(args, a)
		}
	}

	if param.Reason != "" {
		conds = append(conds, "reason LIKE ?")
		args = append(args, "%"+param.Reason+"%")
	}
	if param.Version != "" {
		conds = append(conds, "version = ?")
		args = append(args, param.Version)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFffCloseItem(row *fffCloseRow) *biz.FffCloseItem {
	return &biz.FffCloseItem{
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

func (r *foDashboardRepo) ListFdrTrigger(ctx context.Context, param *biz.FdrTriggerParam) ([]*biz.FdrTriggerItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
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

	list := make([]*biz.FdrTriggerItem, 0, len(rows))
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
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	if len(param.AnonymousIds) > 0 {
		ph := strings.Repeat("?,", len(param.AnonymousIds))
		ph = ph[:len(ph)-1]
		conds = append(conds, "anonymous_id IN ("+ph+")")
		for _, a := range param.AnonymousIds {
			args = append(args, a)
		}
	}

	if param.Uuid != "" {
		conds = append(conds, "uuid = ?")
		args = append(args, param.Uuid)
	}
	if param.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, param.Status)
	}
	if param.Detail != "" {
		conds = append(conds, "detail LIKE ?")
		args = append(args, "%"+param.Detail+"%")
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFdrTriggerItem(row *fdrTriggerRow) *biz.FdrTriggerItem {
	return &biz.FdrTriggerItem{
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

func (r *foDashboardRepo) ListFclTrigger(ctx context.Context, param *biz.FclTriggerParam) ([]*biz.FclTriggerItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
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

	list := make([]*biz.FclTriggerItem, 0, len(rows))
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
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	if len(param.AnonymousIds) > 0 {
		ph := strings.Repeat("?,", len(param.AnonymousIds))
		ph = ph[:len(ph)-1]
		conds = append(conds, "anonymous_id IN ("+ph+")")
		for _, a := range param.AnonymousIds {
			args = append(args, a)
		}
	}

	if param.Uuid != "" {
		conds = append(conds, "uuid = ?")
		args = append(args, param.Uuid)
	}
	if param.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, param.Status)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toFclTriggerItem(row *fclTriggerRow) *biz.FclTriggerItem {
	return &biz.FclTriggerItem{
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

func (r *foDashboardRepo) ListUuidDetail(ctx context.Context, param *biz.UuidDetailParam) ([]*biz.UuidDetailItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
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

	list := make([]*biz.UuidDetailItem, 0, len(rows))
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
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	if len(param.AnonymousIds) > 0 {
		ph := strings.Repeat("?,", len(param.AnonymousIds))
		ph = ph[:len(ph)-1]
		conds = append(conds, "anonymous_id IN ("+ph+")")
		for _, a := range param.AnonymousIds {
			args = append(args, a)
		}
	}
	if param.OnlyFail {
		conds = append(conds, "fcl_status != 'success'")
	}
	switch param.StageFilter {
	case "fff_success":
		conds = append(conds, "fff_status = 'success'")
	case "fff_discard":
		conds = append(conds, "fff_status = 'discard'")
	case "fdr_success":
		conds = append(conds, "fdr_status = 'success'")
	case "fdr_discard":
		conds = append(conds, "fdr_status = 'discard'")
	case "fcl_success":
		conds = append(conds, "fcl_status = 'success'")
	case "fcl_discard":
		conds = append(conds, "fcl_status = 'discard'")
	}

	if param.Uuid != "" {
		conds = append(conds, "uuid = ?")
		args = append(args, param.Uuid)
	}
	if param.FffStatus != "" {
		conds = append(conds, "fff_status = ?")
		args = append(args, param.FffStatus)
	}
	if param.FdrStatus != "" {
		conds = append(conds, "fdr_status = ?")
		args = append(args, param.FdrStatus)
	}
	if param.FclStatus != "" {
		conds = append(conds, "fcl_status = ?")
		args = append(args, param.FclStatus)
	}

	return " WHERE " + strings.Join(conds, " AND "), args
}

func toUuidDetailItem(row *uuidDetailRow) *biz.UuidDetailItem {
	return &biz.UuidDetailItem{
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

func (r *foDashboardRepo) GetCloseReason(ctx context.Context, param *biz.CloseReasonParam) ([]*biz.CloseReasonItem, error) {
	// reason 粒度没有 filter_name，传了 filter_name 回退明细表
	if param.FilterName != "" {
		return r.getCloseReasonFromDetail(ctx, param)
	}
	return r.getCloseReasonFromSummary(ctx, param)
}

// getCloseReasonFromSummary 未传 filter_name：查 fff_close 汇总表 reason 粒度
func (r *foDashboardRepo) getCloseReasonFromSummary(ctx context.Context, param *biz.CloseReasonParam) ([]*biz.CloseReasonItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
	where, args := buildCloseReasonWhere(param)

	sql := fmt.Sprintf(`
		SELECT close_reason_tag AS category, SUM(close_count) AS cnt
		FROM %s%s
		AND summary_grain = 'reason'
		AND close_reason_tag != '%s' AND close_reason_tag != ''
		GROUP BY close_reason_tag
		ORDER BY cnt DESC`, tableFffCloseDailySummary, where, aggAllValue)

	var rows []*closeReasonRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*biz.CloseReasonItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.CloseReasonItem{
			Name:  row.Category,
			Value: row.Cnt,
		})
	}
	return list, nil
}

// getCloseReasonFromDetail 传了 filter_name 时回退明细表（dwd_cfdi_basic_fff_close）
func (r *foDashboardRepo) getCloseReasonFromDetail(ctx context.Context, param *biz.CloseReasonParam) ([]*biz.CloseReasonItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
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

	list := make([]*biz.CloseReasonItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.CloseReasonItem{
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
	// close 汇总表只有 filter_name 列，event_names 的值当 filter_name 用
	conds, args = appendFffRunningEventNamesAsFilterNames(conds, args, param.EventNames)
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func (r *foDashboardRepo) GetFunnel(ctx context.Context, param *biz.FunnelParam) (*biz.FunnelData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
	base := " FROM ads_do_cfdi_daily" + where + " AND event_name != 'Forever_log'"

	type statRow struct {
		FffTotal   int64 `gorm:"column:fff_total"`
		FffAllow   int64 `gorm:"column:fff_allow"`
		FdrSuccess int64 `gorm:"column:fdr_success"`
		FdrFail    int64 `gorm:"column:fdr_fail"`
		FclSuccess int64 `gorm:"column:fcl_success"`
		FclFail    int64 `gorm:"column:fcl_fail"`
	}
	type detailRow struct {
		Name string `gorm:"column:name"`
		Cnt  int64  `gorm:"column:cnt"`
	}

	statSQL := `SELECT
		SUM(cnt) AS fff_total,
		SUM(CASE WHEN fff_status!='discard' THEN cnt ELSE 0 END) AS fff_allow,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') THEN cnt ELSE 0 END) AS fdr_success,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` THEN cnt ELSE 0 END) AS fdr_fail,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status != 'discard' THEN cnt ELSE 0 END) AS fcl_success,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' THEN cnt ELSE 0 END) AS fcl_fail` + base

	fffFailSQL := `SELECT fff_detail_tag AS name, SUM(cnt) AS cnt` + base +
		` AND fff_status = 'discard' GROUP BY fff_detail_tag ORDER BY cnt DESC LIMIT 10`

	fdrFailSQL := `SELECT fdr_detail_tag AS name, SUM(cnt) AS cnt` + base +
		` AND ` + fdrStageFailedCondition() + ` GROUP BY fdr_detail_tag ORDER BY cnt DESC LIMIT 10`

	fclFailSQL := `SELECT fcl_detail_tag AS name, SUM(cnt) AS cnt` + base +
		` AND fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' GROUP BY fcl_detail_tag ORDER BY cnt DESC LIMIT 10`

	var (
		stat                      statRow
		fffFail, fdrFail, fclFail []*detailRow
		statErr, f1, f2, f3       error
	)

	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); statErr = db.Raw(statSQL, args...).Scan(&stat).Error }()
	go func() { defer wg.Done(); f1 = db.Raw(fffFailSQL, args...).Scan(&fffFail).Error }()
	go func() { defer wg.Done(); f2 = db.Raw(fdrFailSQL, args...).Scan(&fdrFail).Error }()
	go func() { defer wg.Done(); f3 = db.Raw(fclFailSQL, args...).Scan(&fclFail).Error }()
	wg.Wait()

	if statErr != nil {
		return nil, statErr
	}
	if f1 != nil {
		return nil, f1
	}
	if f2 != nil {
		return nil, f2
	}
	if f3 != nil {
		return nil, f3
	}

	cfdiRate := 0.0
	if stat.FffTotal > 0 {
		cfdiRate = math.Round(float64(stat.FclSuccess)/float64(stat.FffTotal)*1000) / 10
	}

	toReasons := func(rows []*detailRow) []*biz.FunnelFailReason {
		out := make([]*biz.FunnelFailReason, 0, len(rows))
		for _, r := range rows {
			out = append(out, &biz.FunnelFailReason{Name: r.Name, Count: r.Cnt})
		}
		return out
	}

	return &biz.FunnelData{
		Stat: &biz.FunnelStat{
			FffTotal:   stat.FffTotal,
			FffAllow:   stat.FffAllow,
			FdrSuccess: stat.FdrSuccess,
			FdrFail:    stat.FdrFail,
			FclSuccess: stat.FclSuccess,
			FclFail:    stat.FclFail,
			CfdiRate:   cfdiRate,
		},
		FffFail: toReasons(fffFail),
		FdrFail: toReasons(fdrFail),
		FclFail: toReasons(fclFail),
	}, nil
}

// stageTrendRow 阶段趋势失败原因行（汇总表 reason 粒度）
type stageTrendRow struct {
	Name  string    `gorm:"column:name"`
	Dt    time.Time `gorm:"column:dt"`
	Count int64     `gorm:"column:count"`
}

// fffOverviewRow FFF overview 粒度逐日成功/失败
type fffOverviewRow struct {
	Dt         time.Time `gorm:"column:dt"`
	FffSuccess int64     `gorm:"column:fff_success"`
	FffFailed  int64     `gorm:"column:fff_failed"`
}

// stageSuccessFailedRow 阶段 overview 粒度逐日成功/失败（FDR/FCL 趋势）
type stageSuccessFailedRow struct {
	Dt      time.Time `gorm:"column:dt"`
	Success int64     `gorm:"column:success"`
	Failed  int64     `gorm:"column:failed"`
}

func fffStageTrendNames() []string {
	return []string{
		"success",
		"check_is_no_need_cooldown",
		"check_drm_quota",
		"check_need_acquire_data",
		"check_not_reach_trigger_maximum",
		"bag_invalid",
		"event_not_recognized",
		"tls_error",
		"query cloud DISCARD, detail:Filter quota exceeded",
		"query cloud DISCARD, detail:EventName is in blacklist",
		"other",
	}
}

func fdrStageTrendNames() []string {
	return []string{
		"success",
		"because of full gc",
		"mem pool water line",
		"Disk overrun",
		"Exceeds the maximum number of files",
		"bag_invalid",
		"bag_dir_missing",
		"event_not_recognized",
		"unauthorized",
		"other",
	}
}

func fclStageTrendNames() []string {
	return []string{
		"success",
		"query cloud DISCARD, detail:Filter quota exceeded",
		"reach upload limit",
		"query cloud DISCARD, detail:EventName is in blacklist",
		"geofence_error",
		"unexpected geofence cause",
		"tls_error",
		"bag not exist",
		"meta file lost",
		"meta file empty",
		"unexpected bag_upload_query cause",
		"s3 upload force quit",
		"create socket failed",
		"http request failed",
		"transfer dns failed",
		"other",
	}
}

func (r *foDashboardRepo) GetStageTrend(ctx context.Context, param *biz.StageTrendParam) (*biz.StageTrendData, error) {
	// 专项分析走汇总表；失败原因拆分依赖 reason 粒度（无 filter_name），传了 filter_name 回退明细表
	if param.FilterName != "" {
		return r.getStageTrendFromDetail(ctx, param)
	}
	return r.getStageTrendFromSummary(ctx, param)
}

func (r *foDashboardRepo) getStageTrendFromSummary(ctx context.Context, param *biz.StageTrendParam) (*biz.StageTrendData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	whereFffOverview, argsFffOverview := buildAggCommonWhere("overview", "", param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
	fffOverviewSQL := `SELECT dt,
		SUM(success_count) AS fff_success,
		SUM(failed_count) AS fff_failed
		FROM ` + tableFffTriggerDailySummary + whereFffOverview + `
		GROUP BY dt ORDER BY dt ASC`

	whereFffReason, argsFffReason := buildAggCommonWhere("reason", "", param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
	fffReasonSQL := `SELECT detail_tag AS name, dt, SUM(failed_count) AS count
		FROM ` + tableFffTriggerDailySummary + whereFffReason + `
		AND detail_tag != '` + aggAllValue + `' AND detail_tag != ''
		GROUP BY dt, detail_tag`

	whereFdr, argsFdr := buildAggCommonWhere("overview", "", param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
	fdrSQL := `SELECT dt, SUM(success_count) AS success, SUM(failed_count) AS failed
		FROM ` + tableFdrTriggerDailySummary + whereFdr + `
		GROUP BY dt ORDER BY dt ASC`

	whereFcl, argsFcl := buildAggCommonWhere("overview", "", param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
	fclSQL := `SELECT dt, SUM(success_count) AS success, SUM(failed_count) AS failed
		FROM ` + tableFclTriggerDailySummary + whereFcl + `
		GROUP BY dt ORDER BY dt ASC`

	var (
		fffOverviewRows []*fffOverviewRow
		fffReasonRows   []*stageTrendRow
		fdrRows         []*stageSuccessFailedRow
		fclRows         []*stageSuccessFailedRow
		e1, e2, e3, e4  error
	)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); e1 = db.Raw(fffOverviewSQL, argsFffOverview...).Scan(&fffOverviewRows).Error }()
	go func() { defer wg.Done(); e2 = db.Raw(fffReasonSQL, argsFffReason...).Scan(&fffReasonRows).Error }()
	go func() { defer wg.Done(); e3 = db.Raw(fdrSQL, argsFdr...).Scan(&fdrRows).Error }()
	go func() { defer wg.Done(); e4 = db.Raw(fclSQL, argsFcl...).Scan(&fclRows).Error }()
	wg.Wait()
	for _, e := range []error{e1, e2, e3, e4} {
		if e != nil {
			return nil, e
		}
	}

	dates := buildTrendDates(param.StartDt, param.EndDt)
	if len(dates) == 0 {
		dates = collectTrendDates(fffOverviewRows, fffReasonRows, fdrRows, fclRows)
	}

	fffSucc := make(map[string]int64, len(fffOverviewRows))
	fffFailed := make(map[string]int64, len(fffOverviewRows))
	for _, row := range fffOverviewRows {
		dt := row.Dt.Format("2006-01-02")
		fffSucc[dt] = row.FffSuccess
		fffFailed[dt] = row.FffFailed
	}

	fffNames := fffStageTrendNames()
	fdrNames := []string{"success", "failed"}
	fclNames := []string{"success", "failed"}

	return &biz.StageTrendData{
		Dates: dates,
		Fff:   buildStageSeries(dates, assembleFffStageVals(dates, fffSucc, fffFailed, fffReasonRows, fffNames), fffNames),
		Fdr:   buildStageSeries(dates, assembleSuccessFailedVals(dates, fdrRows), fdrNames),
		Fcl:   buildStageSeries(dates, assembleSuccessFailedVals(dates, fclRows), fclNames),
	}, nil
}

func buildTrendDates(startDt, endDt string) []string {
	if startDt == "" || endDt == "" {
		return nil
	}
	start, err1 := time.Parse("2006-01-02", startDt)
	end, err2 := time.Parse("2006-01-02", endDt)
	if err1 != nil || err2 != nil {
		return nil
	}
	if start.After(end) {
		start, end = end, start
	}
	dates := make([]string, 0, int(end.Sub(start).Hours()/24)+1)
	for dt := start; !dt.After(end); dt = dt.AddDate(0, 0, 1) {
		dates = append(dates, dt.Format("2006-01-02"))
	}
	return dates
}

func collectTrendDates(fffOverviewRows []*fffOverviewRow, fffReasonRows []*stageTrendRow, fdrRows, fclRows []*stageSuccessFailedRow) []string {
	seen := map[string]struct{}{}
	for _, row := range fffOverviewRows {
		seen[row.Dt.Format("2006-01-02")] = struct{}{}
	}
	for _, row := range fffReasonRows {
		seen[row.Dt.Format("2006-01-02")] = struct{}{}
	}
	for _, row := range fdrRows {
		seen[row.Dt.Format("2006-01-02")] = struct{}{}
	}
	for _, row := range fclRows {
		seen[row.Dt.Format("2006-01-02")] = struct{}{}
	}
	dates := make([]string, 0, len(seen))
	for dt := range seen {
		dates = append(dates, dt)
	}
	sort.Strings(dates)
	return dates
}

func assembleFffStageVals(dates []string, success map[string]int64, failed map[string]int64, rows []*stageTrendRow, names []string) map[string][]int64 {
	out := assembleStageVals(dates, success, rows, names)
	otherIdx := -1
	for i, name := range names {
		if name == "other" {
			otherIdx = i
			break
		}
	}
	if otherIdx < 0 {
		return out
	}
	for _, dt := range dates {
		vals := out[dt]
		var reasonTotal int64
		for i := 1; i < len(vals); i++ {
			reasonTotal += vals[i]
		}
		if missing := failed[dt] - reasonTotal; missing > 0 {
			vals[otherIdx] += missing
		}
	}
	return out
}

// assembleStageVals 把 success 量 + 各 detail_tag 失败量组装成 map[dt][]int64（顺序对齐 names）
func assembleStageVals(dates []string, succ map[string]int64, rows []*stageTrendRow, names []string) map[string][]int64 {
	tagDt := make(map[string]map[string]int64, len(rows))
	for _, r := range rows {
		dt := r.Dt.Format("2006-01-02")
		m, ok := tagDt[r.Name]
		if !ok {
			m = map[string]int64{}
			tagDt[r.Name] = m
		}
		m[dt] += r.Count
	}
	out := make(map[string][]int64, len(dates))
	for _, dt := range dates {
		vals := make([]int64, len(names))
		vals[0] = succ[dt]
		for j := 1; j < len(names); j++ {
			if m, ok := tagDt[names[j]]; ok {
				vals[j] = m[dt]
			}
		}
		out[dt] = vals
	}
	return out
}

func assembleSuccessFailedVals(dates []string, rows []*stageSuccessFailedRow) map[string][]int64 {
	dtMap := make(map[string][2]int64, len(rows))
	for _, r := range rows {
		dtMap[r.Dt.Format("2006-01-02")] = [2]int64{r.Success, r.Failed}
	}
	out := make(map[string][]int64, len(dates))
	for _, dt := range dates {
		v := dtMap[dt]
		out[dt] = []int64{v[0], v[1]}
	}
	return out
}

// getStageTrendFromDetail 传了 filter_name 时回退明细表（ads_do_cfdi_daily）
func (r *foDashboardRepo) getStageTrendFromDetail(ctx context.Context, param *biz.StageTrendParam) (*biz.StageTrendData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	type stageAllRow struct {
		Dt                  time.Time `gorm:"column:dt"`
		FffSuccess          int64     `gorm:"column:fff_success"`
		FffCooldown         int64     `gorm:"column:fff_cooldown"`
		FffDrmQuota         int64     `gorm:"column:fff_drm_quota"`
		FffNoAcquire        int64     `gorm:"column:fff_no_acquire"`
		FffTriggerMax       int64     `gorm:"column:fff_trigger_max"`
		FffBagInvalid       int64     `gorm:"column:fff_bag_invalid"`
		FffEventNotRec      int64     `gorm:"column:fff_event_not_recognized"`
		FffTlsError         int64     `gorm:"column:fff_tls_error"`
		FffQuotaExceeded    int64     `gorm:"column:fff_quota_exceeded"`
		FffBlacklist        int64     `gorm:"column:fff_blacklist"`
		FffOther            int64     `gorm:"column:fff_other"`
		FdrSuccess          int64     `gorm:"column:fdr_success"`
		FdrFullGC           int64     `gorm:"column:fdr_full_gc"`
		FdrMemPoolWaterLine int64     `gorm:"column:fdr_mem_pool_water_line"`
		FdrDiskOverrun      int64     `gorm:"column:fdr_disk_overrun"`
		FdrMaxFiles         int64     `gorm:"column:fdr_max_files"`
		FdrBagInvalid       int64     `gorm:"column:fdr_bag_invalid"`
		FdrBagDirMissing    int64     `gorm:"column:fdr_bag_dir_missing"`
		FdrEventNotRec      int64     `gorm:"column:fdr_event_not_recognized"`
		FdrUnauthorized     int64     `gorm:"column:fdr_unauthorized"`
		FdrOther            int64     `gorm:"column:fdr_other"`
		FclSuccess          int64     `gorm:"column:fcl_success"`
		FclFilterQuota      int64     `gorm:"column:fcl_filter_quota"`
		FclReachUploadLimit int64     `gorm:"column:fcl_reach_upload_limit"`
		FclEventBlacklist   int64     `gorm:"column:fcl_event_blacklist"`
		FclGeofenceError    int64     `gorm:"column:fcl_geofence_error"`
		FclUnexpectedGeo    int64     `gorm:"column:fcl_unexpected_geofence"`
		FclTlsError         int64     `gorm:"column:fcl_tls_error"`
		FclBagNotExist      int64     `gorm:"column:fcl_bag_not_exist"`
		FclMetaFileLost     int64     `gorm:"column:fcl_meta_file_lost"`
		FclMetaFileEmpty    int64     `gorm:"column:fcl_meta_file_empty"`
		FclBagUploadQuery   int64     `gorm:"column:fcl_bag_upload_query"`
		FclS3ForceQuit      int64     `gorm:"column:fcl_s3_force_quit"`
		FclCreateSocket     int64     `gorm:"column:fcl_create_socket"`
		FclHTTPRequest      int64     `gorm:"column:fcl_http_request"`
		FclTransferDNS      int64     `gorm:"column:fcl_transfer_dns"`
		FclOther            int64     `gorm:"column:fcl_other"`
	}

	sql := `SELECT dt,
		SUM(CASE WHEN fff_status != 'discard' THEN cnt ELSE 0 END) AS fff_success,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='check_is_no_need_cooldown' THEN cnt ELSE 0 END) AS fff_cooldown,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='check_drm_quota' THEN cnt ELSE 0 END) AS fff_drm_quota,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='check_need_acquire_data' THEN cnt ELSE 0 END) AS fff_no_acquire,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='check_not_reach_trigger_maximum' THEN cnt ELSE 0 END) AS fff_trigger_max,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='bag_invalid' THEN cnt ELSE 0 END) AS fff_bag_invalid,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='event_not_recognized' THEN cnt ELSE 0 END) AS fff_event_not_recognized,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='tls_error' THEN cnt ELSE 0 END) AS fff_tls_error,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='query cloud DISCARD, detail:Filter quota exceeded' THEN cnt ELSE 0 END) AS fff_quota_exceeded,
		SUM(CASE WHEN fff_status='discard' AND fff_detail_tag='query cloud DISCARD, detail:EventName is in blacklist' THEN cnt ELSE 0 END) AS fff_blacklist,
		SUM(CASE WHEN fff_status='discard'
			AND fff_detail_tag NOT IN ('check_is_no_need_cooldown','check_drm_quota','check_need_acquire_data','check_not_reach_trigger_maximum','bag_invalid','event_not_recognized','tls_error','query cloud DISCARD, detail:Filter quota exceeded','query cloud DISCARD, detail:EventName is in blacklist')
			THEN cnt ELSE 0 END) AS fff_other,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') THEN cnt ELSE 0 END) AS fdr_success,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='because of full gc' THEN cnt ELSE 0 END) AS fdr_full_gc,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='mem pool water line' THEN cnt ELSE 0 END) AS fdr_mem_pool_water_line,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='Disk overrun' THEN cnt ELSE 0 END) AS fdr_disk_overrun,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='Exceeds the maximum number of files' THEN cnt ELSE 0 END) AS fdr_max_files,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='bag_invalid' THEN cnt ELSE 0 END) AS fdr_bag_invalid,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='bag_dir_missing' THEN cnt ELSE 0 END) AS fdr_bag_dir_missing,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='event_not_recognized' THEN cnt ELSE 0 END) AS fdr_event_not_recognized,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + ` AND fdr_detail_tag='unauthorized' THEN cnt ELSE 0 END) AS fdr_unauthorized,
		SUM(CASE WHEN ` + fdrStageFailedCondition() + `
			AND fdr_detail_tag NOT IN ('because of full gc','mem pool water line','Disk overrun','Exceeds the maximum number of files','bag_invalid','bag_dir_missing','event_not_recognized','unauthorized')
			THEN cnt ELSE 0 END) AS fdr_other,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status != 'discard' THEN cnt ELSE 0 END) AS fcl_success,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='query cloud DISCARD, detail:Filter quota exceeded' THEN cnt ELSE 0 END) AS fcl_filter_quota,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='reach upload limit' THEN cnt ELSE 0 END) AS fcl_reach_upload_limit,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='query cloud DISCARD, detail:EventName is in blacklist' THEN cnt ELSE 0 END) AS fcl_event_blacklist,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='geofence_error' THEN cnt ELSE 0 END) AS fcl_geofence_error,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='unexpected geofence cause' THEN cnt ELSE 0 END) AS fcl_unexpected_geofence,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='tls_error' THEN cnt ELSE 0 END) AS fcl_tls_error,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='bag not exist' THEN cnt ELSE 0 END) AS fcl_bag_not_exist,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='meta file lost' THEN cnt ELSE 0 END) AS fcl_meta_file_lost,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='meta file empty' THEN cnt ELSE 0 END) AS fcl_meta_file_empty,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='unexpected bag_upload_query cause' THEN cnt ELSE 0 END) AS fcl_bag_upload_query,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='s3 upload force quit' THEN cnt ELSE 0 END) AS fcl_s3_force_quit,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='create socket failed' THEN cnt ELSE 0 END) AS fcl_create_socket,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='http request failed' THEN cnt ELSE 0 END) AS fcl_http_request,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard' AND fcl_detail_tag='transfer dns failed' THEN cnt ELSE 0 END) AS fcl_transfer_dns,
		SUM(CASE WHEN fff_status != 'discard' AND (fdr_status = 'success' OR fcl_status != '') AND fcl_status = 'discard'
			AND fcl_detail_tag NOT IN ('query cloud DISCARD, detail:Filter quota exceeded','reach upload limit','query cloud DISCARD, detail:EventName is in blacklist','geofence_error','unexpected geofence cause','tls_error','bag not exist','meta file lost','meta file empty','unexpected bag_upload_query cause','s3 upload force quit','create socket failed','http request failed','transfer dns failed')
			THEN cnt ELSE 0 END) AS fcl_other
		FROM ads_do_cfdi_daily` + where + ` AND event_name != '` + aggForeverLogValue + `'
		GROUP BY dt ORDER BY dt ASC`

	var rows []*stageAllRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	fffNames := fffStageTrendNames()
	fdrNames := fdrStageTrendNames()
	fclNames := fclStageTrendNames()

	fffVals := make(map[string][]int64, len(rows))
	fdrVals := make(map[string][]int64, len(rows))
	fclVals := make(map[string][]int64, len(rows))
	dates := make([]string, 0, len(rows))
	for _, row := range rows {
		dt := row.Dt.Format("2006-01-02")
		dates = append(dates, dt)
		fffVals[dt] = []int64{row.FffSuccess, row.FffCooldown, row.FffDrmQuota, row.FffNoAcquire, row.FffTriggerMax, row.FffBagInvalid, row.FffEventNotRec, row.FffTlsError, row.FffQuotaExceeded, row.FffBlacklist, row.FffOther}
		fdrVals[dt] = []int64{row.FdrSuccess, row.FdrFullGC, row.FdrMemPoolWaterLine, row.FdrDiskOverrun, row.FdrMaxFiles, row.FdrBagInvalid, row.FdrBagDirMissing, row.FdrEventNotRec, row.FdrUnauthorized, row.FdrOther}
		fclVals[dt] = []int64{row.FclSuccess, row.FclFilterQuota, row.FclReachUploadLimit, row.FclEventBlacklist, row.FclGeofenceError, row.FclUnexpectedGeo, row.FclTlsError, row.FclBagNotExist, row.FclMetaFileLost, row.FclMetaFileEmpty, row.FclBagUploadQuery, row.FclS3ForceQuit, row.FclCreateSocket, row.FclHTTPRequest, row.FclTransferDNS, row.FclOther}
	}

	return &biz.StageTrendData{
		Dates: dates,
		Fff:   buildStageSeries(dates, fffVals, fffNames),
		Fdr:   buildStageSeries(dates, fdrVals, fdrNames),
		Fcl:   buildStageSeries(dates, fclVals, fclNames),
	}, nil
}

func buildStageSeries(dates []string, valMap map[string][]int64, names []string) []*biz.StageTrendSeries {
	series := make([]*biz.StageTrendSeries, len(names))
	for i, name := range names {
		series[i] = &biz.StageTrendSeries{Name: name, Data: make([]int64, len(dates))}
	}
	for i, dt := range dates {
		vals, ok := valMap[dt]
		if !ok {
			continue
		}
		for j := range series {
			if j < len(vals) {
				series[j].Data[i] = vals[j]
			}
		}
	}
	return series
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
// GetCarTypesByProject 查指定项目下的车型列表(近3个月,联动场景专用,不走缓存)。
// 用 _agg 车辆日汇总表(普通表 ads_cfdi_vehicle_daily_summary 计划下线,不依赖);
// 90 天全量约 470 万行、单项目 5.5 万行,DISTINCT 秒回。
// 不用 fff_running(亿级明细且 project_name 无索引,90 天扫描数十秒)。
func (r *foDashboardRepo) GetCarTypesByProject(ctx context.Context, projectName string) ([]string, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	var rows []struct{ Val string }
	err := db.Raw(`SELECT DISTINCT car_type AS val
		FROM `+tableVehicleDailySummaryAgg+`
		WHERE dt >= DATE_SUB(CURDATE(), INTERVAL 90 DAY)
		AND project_name = ?
		AND car_type IS NOT NULL AND car_type != ''
		ORDER BY val`, projectName).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Val)
	}
	return out, nil
}

func (r *foDashboardRepo) fetchDimensions(ctx context.Context) (*biz.FoDimensions, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

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

// runningTrendRow GetFffRunningTrend 聚合扫描结构
type runningTrendRow struct {
	Dt           time.Time `gorm:"column:dt"`
	VehicleCount int64     `gorm:"column:vehicle_count"`
}

func (r *foDashboardRepo) GetFffRunningTrend(ctx context.Context, param *biz.FffRunningTrendParam) (*biz.FffRunningTrendData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	where, args := buildFffRunningTrendWhere(param)
	sql := `SELECT dt, SUM(running_switch_on_vehicle_count) AS vehicle_count
		FROM ` + tableVehicleDailySummaryAgg + where + ` GROUP BY dt ORDER BY dt`

	var rows []runningTrendRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	dates := make([]string, 0, len(rows))
	counts := make([]int64, 0, len(rows))
	for _, row := range rows {
		dates = append(dates, row.Dt.Format("2006-01-02"))
		counts = append(counts, row.VehicleCount)
	}
	return &biz.FffRunningTrendData{Dates: dates, Counts: counts}, nil
}

// buildFffRunningTrendWhere 构建 _agg 车辆汇总表的 WHERE。
// running/trend 需要的是「去重车辆数」，fff_running 汇总表的 vehicle_count 会因
// on_autopilot/function_mode 等易变维度重复计数（同车拆多行），改用
// ads_cfdi_vehicle_daily_summary_agg 的 running_switch_on_vehicle_count（车辆粒度，SUM 不重复）。
func buildFffRunningTrendWhere(param *biz.FffRunningTrendParam) (string, []interface{}) {
	conds := []string{
		"event_name = ?",
		"dt BETWEEN ? AND ?",
	}
	args := []interface{}{param.FilterName, param.StartDt, param.EndDt}

	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)
	return " WHERE " + strings.Join(conds, " AND "), args
}

func buildRunningOverviewSQL(param *biz.FffRunningParam) (string, []interface{}) {
	where, args := buildFffRunningWhere(param)
	vehicleWhere, vehicleArgs := buildFffRunningVehicleWhere(param)
	// vehicle_total:区间内峰值日活跃车辆数——每天按天聚合(消除 sw_version 展开)，
	// 再取区间 MAX(消除跨天膨胀);event_name 维度由 vehicleWhere 决定(前端传啥查啥)
	sql := `SELECT
		COALESCE(SUM(running_count), 0) AS running_total,
		(
			SELECT COALESCE(MAX(daily_count), 0) FROM (
				SELECT SUM(running_switch_on_vehicle_count) AS daily_count
				FROM ` + tableVehicleDailySummaryAgg + vehicleWhere + `
				GROUP BY dt
			) t
		) AS vehicle_total,
		COALESCE(SUM(switch_on_count), 0) AS switch_on_total,
		COALESCE(SUM(switch_off_count), 0) AS switch_off_total,
		COALESCE(SUM(running_success_count), 0) AS running_success,
		COALESCE(SUM(running_failed_count), 0) AS running_failed,
		COUNT(DISTINCT filter_name) AS filter_count
		FROM ` + tableFffRunningDailySummary + where + `
		AND summary_grain = 'filter'
		AND filter_name != '` + aggAllValue + `' AND filter_name != ''`
	args = append(vehicleArgs, args...)
	return sql, args
}

// GetRunningOverview 筛选器运行健康概览（运行记录数/车辆数/开关占比）
func (r *foDashboardRepo) GetRunningOverview(ctx context.Context, param *biz.FffRunningParam) (*biz.FoRunningOverviewData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
	sql, args := buildRunningOverviewSQL(param)

	type scanRow struct {
		RunningTotal   int64 `gorm:"column:running_total"`
		VehicleTotal   int64 `gorm:"column:vehicle_total"`
		SwitchOnTotal  int64 `gorm:"column:switch_on_total"`
		SwitchOffTotal int64 `gorm:"column:switch_off_total"`
		RunningSuccess int64 `gorm:"column:running_success"`
		RunningFailed  int64 `gorm:"column:running_failed"`
		FilterCount    int64 `gorm:"column:filter_count"`
	}
	var sr scanRow
	if err := db.Raw(sql, args...).Scan(&sr).Error; err != nil {
		return nil, err
	}
	return &biz.FoRunningOverviewData{
		RunningTotal:   sr.RunningTotal,
		VehicleTotal:   sr.VehicleTotal,
		SwitchOnTotal:  sr.SwitchOnTotal,
		SwitchOffTotal: sr.SwitchOffTotal,
		RunningSuccess: sr.RunningSuccess,
		RunningFailed:  sr.RunningFailed,
		FilterCount:    sr.FilterCount,
	}, nil
}

// GetFffOverview FFF 触发概览（触发总数/成功数/成功率）
func (r *foDashboardRepo) GetFffOverview(ctx context.Context, param *biz.FffTriggerParam) (*biz.FoFffOverviewData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
	sql, args := buildFffOverviewSQL(param)

	type scanRow struct {
		TriggerTotal       int64 `gorm:"column:trigger_total"`
		TriggerSuccess     int64 `gorm:"column:trigger_success"`
		TriggerFailed      int64 `gorm:"column:trigger_failed"`
		TriggerFilterCount int64 `gorm:"column:trigger_filter_count"`
		CloseFilterCount   int64 `gorm:"column:close_filter_count"`
	}
	var sr scanRow
	if err := db.Raw(sql, args...).Scan(&sr).Error; err != nil {
		return nil, err
	}
	return &biz.FoFffOverviewData{
		TriggerTotal:       sr.TriggerTotal,
		TriggerSuccess:     sr.TriggerSuccess,
		TriggerFailed:      sr.TriggerFailed,
		TriggerFilterCount: sr.TriggerFilterCount,
		CloseFilterCount:   sr.CloseFilterCount,
	}, nil
}

func buildFffOverviewSQL(param *biz.FffTriggerParam) (string, []interface{}) {
	grain := grainForFilter(param.FilterName)
	// 日期默认与全看板一致：未传时近 7 天
	where, mainArgs := buildAggCommonWhere(
		grain,
		param.FilterName,
		param.EventNames,
		param.ProjectName,
		param.CarTypes,
		param.StartDt,
		param.EndDt,
	)

	triggerFilterWhere, triggerFilterArgs := buildFffFilterCountWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)
	closeFilterWhere, closeFilterArgs := buildFffFilterCountWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT
		SUM(event_count) AS trigger_total,
		SUM(success_count) AS trigger_success,
		SUM(failed_count) AS trigger_failed,
		(SELECT COUNT(DISTINCT filter_name) FROM ` + tableFffTriggerDailySummary + triggerFilterWhere + `) AS trigger_filter_count,
		(SELECT COUNT(DISTINCT filter_name) FROM ` + tableFffCloseDailySummary + closeFilterWhere + `) AS close_filter_count
		FROM ` + tableFffTriggerDailySummary + where

	args := append(triggerFilterArgs, closeFilterArgs...)
	args = append(args, mainArgs...)
	return sql, args
}

// buildFffFilterCountWhere 汇总表 filter 粒度去重 filter_name 计数的 WHERE（日期/筛选器/项目/车型，与主查询口径一致）
func buildFffFilterCountWhere(filterName string, eventNames []string, projectName string, carTypes []string, startDt, endDt string) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if startDt != "" && endDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, startDt, endDt)
	} else {
		// 日期默认与全看板一致：未传时近 7 天
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}
	conds = append(conds, "summary_grain = 'filter'")
	conds = append(conds, "filter_name != '"+aggAllValue+"' AND filter_name != ''")
	if filterName != "" {
		conds = append(conds, "filter_name = ?")
		args = append(args, filterName)
	}
	// 专项分析：event_names 即算子名，映射为 filter_name 过滤
	conds, args = appendFffRunningEventNamesAsFilterNames(conds, args, eventNames)
	if projectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, projectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", carTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

func (r *foDashboardRepo) GetFffFailReason(ctx context.Context, param *biz.FffTriggerParam) ([]*biz.DoFailReasonItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()
	sql, args := buildFffFailReasonSQL(param)

	return r.scanFffFailReason(ctx, db.Raw(sql, args...))
}

func buildFffFailReasonSQL(param *biz.FffTriggerParam) (string, []interface{}) {
	where, args := buildFffTriggerReasonWhere(param)
	sql := `SELECT detail_tag, SUM(failed_count) AS cnt
		FROM ` + tableFffTriggerDailySummary + where + `
		AND summary_grain = 'reason'
		AND detail_tag != '` + aggAllValue + `' AND detail_tag != ''
		GROUP BY detail_tag
		HAVING cnt > 0
		ORDER BY cnt DESC`

	return sql, args
}

func (r *foDashboardRepo) scanFffFailReason(_ context.Context, tx *gorm.DB) ([]*biz.DoFailReasonItem, error) {
	type row struct {
		DetailTag string `gorm:"column:detail_tag"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*biz.DoFailReasonItem, 0, len(rows))
	for _, row := range rows {
		detail := strings.TrimSpace(row.DetailTag)
		if detail == "" || detail == aggAllValue {
			continue
		}
		list = append(list, &biz.DoFailReasonItem{Name: "FFF-" + detail, Value: row.Cnt})
	}
	return list, nil
}
