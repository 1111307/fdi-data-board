package data

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"

	"fdi_data_board/internal/biz"
)

var DashboardDoProviderSet = wire.NewSet(NewDoDashboardRepo)

var _ biz.DoDashboardRepo = (*doDashboardRepo)(nil)

type doDashboardRepo struct {
	*baseRepo
	slowLog *slowQueryLogger
}

func NewDoDashboardRepo(data *Data) biz.DoDashboardRepo {
	return &doDashboardRepo{
		baseRepo: &baseRepo{data: data},
		slowLog:  newSlowQueryLogger(data.mysqlDB),
	}
}

// doOverviewRow 事件横向对比聚合行
type doOverviewRow struct {
	EventName    string `gorm:"column:event_name"`
	TriggerCount int64  `gorm:"column:trigger_count"`
	FffCount     int64  `gorm:"column:fff_count"`
	FdrCount     int64  `gorm:"column:fdr_count"`
	FclCount     int64  `gorm:"column:fcl_count"`
}

func (r *doDashboardRepo) GetOverview(ctx context.Context, param *biz.DoOverviewParam) ([]*biz.DoOverviewItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	adsSql := `SELECT event_name,
		SUM(cnt) AS trigger_count,
		SUM(CASE WHEN fff_status='success' OR fdr_status='success' OR fcl_status='success' THEN cnt ELSE 0 END) AS fff_count,
		SUM(CASE WHEN fdr_status='success' OR fcl_status='success' THEN cnt ELSE 0 END) AS fdr_count,
		SUM(CASE WHEN fcl_status='success' THEN cnt ELSE 0 END) AS fcl_count
		FROM ads_do_cfdi_daily` + where + `
		AND event_name != 'Forever_log'
		GROUP BY event_name ORDER BY trigger_count DESC`

	type vehicleRow struct {
		EventName    string `gorm:"column:event_name"`
		VehicleCount int64  `gorm:"column:vehicle_count"`
	}
	dwdSql := `SELECT event_name, COUNT(DISTINCT anonymous_id) AS vehicle_count
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		AND event_name != 'Forever_log'
		GROUP BY event_name`

	var (
		rows   []*doOverviewRow
		vRows  []*vehicleRow
		adsErr error
		dwdErr error
	)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		adsErr = r.slowLog.Observe(slowQueryMeta{
			QueryType: "GetOverview", StartDt: param.StartDt, EndDt: param.EndDt,
			EventNames: param.EventNames, FilterName: param.FilterName,
		}, func() error {
			return db.Raw(adsSql, args...).Scan(&rows).Error
		})
	}()
	go func() {
		defer wg.Done()
		dwdErr = db.Raw(dwdSql, args...).Scan(&vRows).Error
	}()
	wg.Wait()
	if adsErr != nil {
		return nil, adsErr
	}
	if dwdErr != nil {
		return nil, dwdErr
	}

	vehicleMap := make(map[string]int64, len(vRows))
	for _, v := range vRows {
		vehicleMap[v.EventName] = v.VehicleCount
	}

	pct := func(a, b int64) float64 {
		if b == 0 {
			return 0
		}
		return math.Round(float64(a)/float64(b)*1000) / 10
	}

	list := make([]*biz.DoOverviewItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.DoOverviewItem{
			EventName:    row.EventName,
			VehicleCount: vehicleMap[row.EventName],
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
		SUM(cnt) AS total,
		SUM(CASE WHEN fcl_status='success' THEN cnt ELSE 0 END) AS success
		FROM ads_do_cfdi_daily` + where + `
		GROUP BY dt ORDER BY dt ASC`

	var rows []*doTrendRow
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetTrend", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames, FilterName: param.FilterName,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
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
	Detail string `gorm:"column:detail_tag"`
	Cnt    int64  `gorm:"column:cnt"`
}

func (r *doDashboardRepo) GetFailReason(ctx context.Context, param *biz.DoFailReasonParam) ([]*biz.DoFailReasonItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT stage, detail_tag, SUM(cnt) AS cnt
	FROM (
		SELECT 'FFF' AS stage, fff_detail_tag AS detail_tag, cnt
		FROM ads_do_cfdi_daily` + where + ` AND fff_status = 'discard' AND fff_detail_tag IS NOT NULL AND fff_detail_tag != ''
		UNION ALL
		SELECT 'FDR' AS stage, fdr_detail_tag AS detail_tag, cnt
		FROM ads_do_cfdi_daily` + where + ` AND fdr_status = 'discard' AND fdr_detail_tag IS NOT NULL AND fdr_detail_tag != ''
		UNION ALL
		SELECT 'FCL' AS stage, fcl_detail_tag AS detail_tag, cnt
		FROM ads_do_cfdi_daily` + where + ` AND fcl_status = 'discard' AND fcl_detail_tag IS NOT NULL AND fcl_detail_tag != ''
	) t
	GROUP BY stage, detail_tag
	ORDER BY stage, cnt DESC`

	tripleArgs := append(append(append([]interface{}{}, args...), args...), args...)
	var rows []*failReasonRow
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetFailReason", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames, FilterName: param.FilterName,
	}, func() error {
		return db.Raw(sql, tripleArgs...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	list := make([]*biz.DoFailReasonItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.DoFailReasonItem{
			Name:  row.Stage + "-" + row.Detail,
			Value: row.Cnt,
		})
	}
	return list, nil
}

func (r *doDashboardRepo) GetCoolTop(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoCoolTopItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere("", param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT filter_name, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + ` AND fff_status != 'success' AND fff_detail_tag = 'cooldown' AND filter_name IS NOT NULL
		GROUP BY filter_name
		ORDER BY cnt DESC
		LIMIT 20`

	type row struct {
		FilterName string `gorm:"column:filter_name"`
		Cnt        int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetCoolTop", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	list := make([]*biz.DoCoolTopItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoCoolTopItem{FilterName: r.FilterName, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetTriggerRank(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoCoolTopItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere("", param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT filter_name, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + ` AND filter_name IS NOT NULL
		GROUP BY filter_name
		ORDER BY cnt DESC
		LIMIT 10`

	type row struct {
		FilterName string `gorm:"column:filter_name"`
		Cnt        int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetTriggerRank", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	list := make([]*biz.DoCoolTopItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoCoolTopItem{FilterName: r.FilterName, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetSwVersion(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoSwVersionItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT fff_sw_version AS sw_version, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + ` AND fff_sw_version IS NOT NULL AND fff_sw_version != ''
		GROUP BY fff_sw_version
		ORDER BY cnt DESC
		LIMIT 20`

	type row struct {
		SwVersion string `gorm:"column:sw_version"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetSwVersion", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames, FilterName: param.FilterName,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	list := make([]*biz.DoSwVersionItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoSwVersionItem{SwVersion: r.SwVersion, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetProjectCar(ctx context.Context, param *biz.DoCommonParam) (*biz.DoProjectCarData, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT project_name, car_type, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + `
		AND project_name IS NOT NULL AND project_name != ''
		AND car_type IS NOT NULL AND car_type != ''
		GROUP BY project_name, car_type
		ORDER BY project_name, car_type`

	type pcRow struct {
		ProjectName string `gorm:"column:project_name"`
		CarType     string `gorm:"column:car_type"`
		Cnt         int64  `gorm:"column:cnt"`
	}
	var rows []*pcRow
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetProjectCar", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames, FilterName: param.FilterName,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	projTotal := map[string]int64{}
	carTypeSet := map[string]struct{}{}
	for _, row := range rows {
		projTotal[row.ProjectName] += row.Cnt
		carTypeSet[row.CarType] = struct{}{}
	}

	projects := make([]string, 0, len(projTotal))
	for p := range projTotal {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projTotal[projects[i]] > projTotal[projects[j]]
	})

	carTypes := make([]string, 0, len(carTypeSet))
	for ct := range carTypeSet {
		carTypes = append(carTypes, ct)
	}
	sort.Strings(carTypes)

	projIdx := make(map[string]int, len(projects))
	for i, p := range projects {
		projIdx[p] = i
	}

	matrix := make(map[string][]int64, len(carTypes))
	for _, ct := range carTypes {
		matrix[ct] = make([]int64, len(projects))
	}
	for _, row := range rows {
		matrix[row.CarType][projIdx[row.ProjectName]] = row.Cnt
	}

	totals := make([]int64, len(projects))
	for i, p := range projects {
		totals[i] = projTotal[p]
	}

	return &biz.DoProjectCarData{
		Projects:      projects,
		CarTypes:      carTypes,
		Matrix:        matrix,
		ProjectTotals: totals,
	}, nil
}

func (r *doDashboardRepo) GetMemTop(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoEventTopItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere("", nil, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT event_name, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + `
		AND fdr_status != 'success'
		AND fdr_detail_tag = 'memory'
		AND event_name IS NOT NULL AND event_name != ''
		GROUP BY event_name ORDER BY cnt DESC LIMIT 20`

	type row struct {
		EventName string `gorm:"column:event_name"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetMemTop", StartDt: param.StartDt, EndDt: param.EndDt,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}
	list := make([]*biz.DoEventTopItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoEventTopItem{EventName: r.EventName, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetDiskTop(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoEventTopItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere("", nil, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT event_name, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + `
		AND fdr_status != 'success'
		AND fdr_detail_tag = 'disk'
		AND event_name IS NOT NULL AND event_name != ''
		GROUP BY event_name ORDER BY cnt DESC LIMIT 20`

	type row struct {
		EventName string `gorm:"column:event_name"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetDiskTop", StartDt: param.StartDt, EndDt: param.EndDt,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}
	list := make([]*biz.DoEventTopItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoEventTopItem{EventName: r.EventName, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetCloseTop(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoCoolTopItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere("", nil, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT filter_name, COUNT(*) AS cnt
		FROM dwd_cfdi_basic_fff_close` + where + `
		AND filter_name IS NOT NULL AND filter_name != ''
		GROUP BY filter_name ORDER BY cnt DESC LIMIT 20`

	type row struct {
		FilterName string `gorm:"column:filter_name"`
		Cnt        int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetCloseTop", StartDt: param.StartDt, EndDt: param.EndDt,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}
	list := make([]*biz.DoCoolTopItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoCoolTopItem{FilterName: r.FilterName, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetQuotaTop(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoEventTopItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere("", nil, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT event_name, SUM(cnt) AS cnt
		FROM ads_do_cfdi_daily` + where + `
		AND fcl_status = 'discard'
		AND fcl_detail_tag = 'quota_exceeded'
		AND event_name IS NOT NULL AND event_name != ''
		GROUP BY event_name ORDER BY cnt DESC LIMIT 20`

	type row struct {
		EventName string `gorm:"column:event_name"`
		Cnt       int64  `gorm:"column:cnt"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetQuotaTop", StartDt: param.StartDt, EndDt: param.EndDt,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}
	list := make([]*biz.DoEventTopItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoEventTopItem{EventName: r.EventName, Count: r.Cnt})
	}
	return list, nil
}

func (r *doDashboardRepo) GetProjectEvent(ctx context.Context, param *biz.DoCommonParam) ([]*biz.DoProjectEventItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildDoCommonWhere(param.FilterName, param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT project_name, COUNT(DISTINCT event_name) AS event_count
		FROM ads_do_cfdi_daily` + where + `
		AND fcl_status = 'success'
		AND project_name IS NOT NULL AND project_name != ''
		GROUP BY project_name ORDER BY event_count DESC`

	type row struct {
		ProjectName string `gorm:"column:project_name"`
		EventCount  int64  `gorm:"column:event_count"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetProjectEvent", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames, FilterName: param.FilterName,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}
	list := make([]*biz.DoProjectEventItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, &biz.DoProjectEventItem{ProjectName: r.ProjectName, EventCount: r.EventCount})
	}
	return list, nil
}

func (r *doDashboardRepo) GetNetSpeed(ctx context.Context, param *biz.DoCommonParam) (*biz.DoNetSpeedData, error) {
	db := r.dorisDB(ctx)
	where, args := buildFclUploadWhere(param)

	sql := `SELECT DATE(create_at) AS dt, car_type,
		ROUND(AVG((package_size / 1024.0 / 1024.0) / (total_cost / 1000.0)), 2) AS avg_bw
		FROM dwd_cfdi_basic_fcl_uploadinfo` + where + `
		AND package_size > 10485760 AND total_cost > 0
		AND car_type IS NOT NULL AND car_type != ''
		GROUP BY DATE(create_at), car_type ORDER BY dt, car_type`

	type row struct {
		Dt      string  `gorm:"column:dt"`
		CarType string  `gorm:"column:car_type"`
		AvgBw   float64 `gorm:"column:avg_bw"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetNetSpeed", StartDt: param.StartDt, EndDt: param.EndDt,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	dateSet := map[string]struct{}{}
	carSet := map[string]struct{}{}
	for _, r := range rows {
		dateSet[r.Dt] = struct{}{}
		carSet[r.CarType] = struct{}{}
	}
	dates := sortedKeys(dateSet)
	carTypes := sortedKeys(carSet)

	dateIdx := make(map[string]int, len(dates))
	for i, d := range dates {
		dateIdx[d] = i
	}

	matrix := make(map[string][]float64, len(carTypes))
	for _, ct := range carTypes {
		matrix[ct] = make([]float64, len(dates))
	}
	for _, r := range rows {
		matrix[r.CarType][dateIdx[r.Dt]] = r.AvgBw
	}

	series := make([]*biz.DoNetSpeedSeries, 0, len(carTypes))
	for _, ct := range carTypes {
		series = append(series, &biz.DoNetSpeedSeries{CarType: ct, Data: matrix[ct]})
	}

	return &biz.DoNetSpeedData{
		Dates:  dates,
		Series: series,
	}, nil
}

func (r *doDashboardRepo) GetFclBw(ctx context.Context, param *biz.DoCommonParam) (*biz.DoFclBwData, error) {
	db := r.dorisDB(ctx)
	where, args := buildFclUploadWhere(param)

	sql := `SELECT DATE(create_at) AS dt,
		ROUND(AVG((package_size / 1024.0 / 1024.0) / (total_cost / 1000.0)), 2) AS avg_bw
		FROM dwd_cfdi_basic_fcl_uploadinfo` + where + `
		AND package_size > 10485760 AND total_cost > 0
		GROUP BY DATE(create_at) ORDER BY dt`

	type row struct {
		Dt    string  `gorm:"column:dt"`
		AvgBw float64 `gorm:"column:avg_bw"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetFclBw", StartDt: param.StartDt, EndDt: param.EndDt,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	dates := make([]string, 0, len(rows))
	values := make([]float64, 0, len(rows))
	for _, r := range rows {
		dates = append(dates, r.Dt)
		values = append(values, r.AvgBw)
	}

	return &biz.DoFclBwData{
		Dates:  dates,
		Values: values,
	}, nil
}

// vehicleStatsRow 车辆统计聚合行
type vehicleStatsRow struct {
	AnonymousId  string  `gorm:"column:anonymous_id"`
	CarType      string  `gorm:"column:car_type"`
	ProjectName  string  `gorm:"column:project_name"`
	TriggerCount int64   `gorm:"column:trigger_count"`
	SuccessCount int64   `gorm:"column:success_count"`
	CfdiRate     float64 `gorm:"column:cfdi_rate"`
}

// vehicleFailRow 车辆失败原因聚合行
type vehicleFailRow struct {
	AnonymousId string `gorm:"column:anonymous_id"`
	FailReason  string `gorm:"column:fail_reason"`
	Cnt         int64  `gorm:"column:cnt"`
}

func (r *doDashboardRepo) vehicleFailReasons(ctx context.Context, ids []string, where string, args []interface{}) map[string]string {
	if len(ids) == 0 {
		return nil
	}
	db := r.dorisDB(ctx)
	idArgs := make([]interface{}, 0, len(args)+len(ids))
	idArgs = append(idArgs, args...)
	for _, id := range ids {
		idArgs = append(idArgs, id)
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	sql := `SELECT anonymous_id,
		CASE
			WHEN fcl_status != 'success' AND fdr_status = 'success' AND fff_status = 'success'
				THEN CONCAT('FCL-', COALESCE(fcl_detail,''))
			WHEN fdr_status != 'success' AND fff_status = 'success'
				THEN CONCAT('FDR-', COALESCE(fdr_detail,''))
			ELSE CONCAT('FFF-', COALESCE(fff_detail,''))
		END AS fail_reason,
		COUNT(*) AS cnt
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		AND (fcl_status != 'success' OR fdr_status != 'success' OR fff_status != 'success')
		AND anonymous_id IN (` + placeholders + `)
		GROUP BY anonymous_id, fail_reason
		ORDER BY anonymous_id, cnt DESC`

	var rows []*vehicleFailRow
	if err := db.Raw(sql, idArgs...).Scan(&rows).Error; err != nil {
		return nil
	}
	result := make(map[string]string, len(ids))
	for _, row := range rows {
		if _, exists := result[row.AnonymousId]; !exists {
			result[row.AnonymousId] = row.FailReason
		}
	}
	return result
}

func (r *doDashboardRepo) GetTopVehicles(ctx context.Context, param *biz.DoVehicleParam) ([]*biz.DoVehicleItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildVehicleWhere(param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT anonymous_id, car_type, project_name,
		COUNT(*) AS trigger_count,
		SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS success_count,
		ROUND(SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END)*100.0/COUNT(*), 1) AS cfdi_rate
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		AND anonymous_id IS NOT NULL AND anonymous_id != ''
		GROUP BY anonymous_id, car_type, project_name
		ORDER BY trigger_count DESC
		LIMIT 20`

	var rows []*vehicleStatsRow
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetTopVehicles", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.AnonymousId)
	}
	reasons := r.vehicleFailReasons(ctx, ids, where, args)

	list := make([]*biz.DoVehicleItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.DoVehicleItem{
			AnonymousId: row.AnonymousId, CarType: row.CarType, ProjectName: row.ProjectName,
			TriggerCount: row.TriggerCount, SuccessCount: row.SuccessCount, CfdiRate: row.CfdiRate,
			MainReason: reasons[row.AnonymousId],
		})
	}
	return list, nil
}

func (r *doDashboardRepo) GetAnomalyVehicles(ctx context.Context, param *biz.DoAnomalyParam) ([]*biz.DoVehicleItem, error) {
	db := r.dorisDB(ctx)
	where, args := buildVehicleWhere(param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	args = append(args, param.MaxRate)
	sql := `SELECT anonymous_id, car_type, project_name,
		COUNT(*) AS trigger_count,
		SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END) AS success_count,
		ROUND(SUM(CASE WHEN fcl_status='success' THEN 1 ELSE 0 END)*100.0/COUNT(*), 1) AS cfdi_rate
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		AND anonymous_id IS NOT NULL AND anonymous_id != ''
		GROUP BY anonymous_id, car_type, project_name
		HAVING cfdi_rate < ?
		ORDER BY cfdi_rate ASC
		LIMIT 100`

	var rows []*vehicleStatsRow
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetAnomalyVehicles", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.AnonymousId)
	}
	reasons := r.vehicleFailReasons(ctx, ids, where, args)

	list := make([]*biz.DoVehicleItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.DoVehicleItem{
			AnonymousId: row.AnonymousId, CarType: row.CarType, ProjectName: row.ProjectName,
			TriggerCount: row.TriggerCount, SuccessCount: row.SuccessCount, CfdiRate: row.CfdiRate,
			MainReason: reasons[row.AnonymousId],
		})
	}
	return list, nil
}

func (r *doDashboardRepo) GetActiveTrend(ctx context.Context, param *biz.DoVehicleParam) (*biz.DoActiveTrendData, error) {
	db := r.dorisDB(ctx)
	where, args := buildVehicleWhere(param.EventNames, param.ProjectName, param.CarTypes, param.StartDt, param.EndDt)

	sql := `SELECT dt, COUNT(DISTINCT anonymous_id) AS active_count
		FROM dwd_cfdi_status_monitor_analysis` + where + `
		GROUP BY dt ORDER BY dt`

	type row struct {
		Dt          string `gorm:"column:dt"`
		ActiveCount int64  `gorm:"column:active_count"`
	}
	var rows []*row
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetActiveTrend", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames,
	}, func() error {
		return db.Raw(sql, args...).Scan(&rows).Error
	}); err != nil {
		return nil, err
	}

	dates := make([]string, 0, len(rows))
	counts := make([]int64, 0, len(rows))
	for _, r := range rows {
		dates = append(dates, r.Dt)
		counts = append(counts, r.ActiveCount)
	}

	return &biz.DoActiveTrendData{
		Dates:  dates,
		Counts: counts,
	}, nil
}

// buildVehicleWhere 构建车辆维度分析 WHERE 子句
func buildVehicleWhere(eventNames []string, projectName string, carTypes []string, startDt, endDt string) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if startDt != "" && endDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, startDt, endDt)
	} else {
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}
	conds, args = appendMultiCond(conds, args, "event_name", eventNames)
	if projectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, projectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", carTypes)
	return " WHERE " + strings.Join(conds, " AND "), args
}

// buildFclUploadWhere 构建 dwd_cfdi_basic_fcl_uploadinfo 的 WHERE 子句
func buildFclUploadWhere(param *biz.DoCommonParam) (string, []interface{}) {
	var conds []string
	var args []interface{}

	if param.StartDt != "" && param.EndDt != "" {
		conds = append(conds, "dt BETWEEN ? AND ?")
		args = append(args, param.StartDt, param.EndDt)
	} else {
		conds = append(conds, "dt >= DATE_SUB(CURDATE(), INTERVAL 6 DAY)")
	}
	if param.ProjectName != "" {
		conds = append(conds, "project_name = ?")
		args = append(args, param.ProjectName)
	}
	conds, args = appendMultiCond(conds, args, "car_type", param.CarTypes)

	return " WHERE " + strings.Join(conds, " AND "), args
}

// sortedKeys 返回 map 的 key 按字母升序排列
func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

func (r *doDashboardRepo) GetDoFunnel(ctx context.Context, param *biz.DoCommonParam) (*biz.FunnelData, error) {
	db := r.dorisDB(ctx)
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
		SUM(CASE WHEN fff_status='success' OR fdr_status='success' OR fcl_status='success' THEN cnt ELSE 0 END) AS fff_allow,
		SUM(CASE WHEN fdr_status='success' OR fcl_status='success' THEN cnt ELSE 0 END) AS fdr_success,
		SUM(CASE WHEN fdr_status='discard' THEN cnt ELSE 0 END) AS fdr_fail,
		SUM(CASE WHEN fcl_status='success' THEN cnt ELSE 0 END) AS fcl_success,
		SUM(CASE WHEN fcl_status='discard' THEN cnt ELSE 0 END) AS fcl_fail` + base

	fffFailSQL := `SELECT fff_detail_tag AS name, SUM(cnt) AS cnt` + base +
		` AND fff_status='discard' GROUP BY fff_detail_tag ORDER BY cnt DESC LIMIT 10`

	fdrFailSQL := `SELECT fdr_detail_tag AS name, SUM(cnt) AS cnt` + base +
		` AND fdr_status='discard' GROUP BY fdr_detail_tag ORDER BY cnt DESC LIMIT 10`

	fclFailSQL := `SELECT fcl_detail_tag AS name, SUM(cnt) AS cnt` + base +
		` AND fcl_status='discard' GROUP BY fcl_detail_tag ORDER BY cnt DESC LIMIT 10`

	var (
		stat                      statRow
		fffFail, fdrFail, fclFail []*detailRow
		statErr, f1, f2, f3       error
	)
	if err := r.slowLog.Observe(slowQueryMeta{
		QueryType: "GetDoFunnel", StartDt: param.StartDt, EndDt: param.EndDt,
		EventNames: param.EventNames, FilterName: param.FilterName,
	}, func() error {
		var wg sync.WaitGroup
		wg.Add(4)
		go func() { defer wg.Done(); statErr = db.Raw(statSQL, args...).Scan(&stat).Error }()
		go func() { defer wg.Done(); f1 = db.Raw(fffFailSQL, args...).Scan(&fffFail).Error }()
		go func() { defer wg.Done(); f2 = db.Raw(fdrFailSQL, args...).Scan(&fdrFail).Error }()
		go func() { defer wg.Done(); f3 = db.Raw(fclFailSQL, args...).Scan(&fclFail).Error }()
		wg.Wait()
		for _, e := range []error{statErr, f1, f2, f3} {
			if e != nil {
				return e
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	pct := func(a, b int64) float64 {
		if b == 0 {
			return 0
		}
		return math.Round(float64(a)/float64(b)*1000) / 10
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
			CfdiRate:   pct(stat.FclSuccess, stat.FffTotal),
		},
		FffFail: toReasons(fffFail),
		FdrFail: toReasons(fdrFail),
		FclFail: toReasons(fclFail),
	}, nil
}
