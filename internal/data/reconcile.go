package data

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"
	"gorm.io/gorm"

	"fdi_data_board/internal/biz"
)

var ReconcileProviderSet = wire.NewSet(NewReconcileRepo)

var _ biz.ReconcileRepo = (*reconcileRepo)(nil)

type reconcileRepo struct {
	*baseRepo
}

func NewReconcileRepo(data *Data) biz.ReconcileRepo {
	return &reconcileRepo{
		baseRepo: &baseRepo{data: data},
	}
}

func reconcilePct(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return math.Round(float64(a)/float64(b)*10000) / 10000
}

// eventLandSelect L2+L3 四段 JOIN 里 matched/convert_failed/landing_failed/missing 的公共表达式。
// matched 必须用 c.status=1 而不是 COUNT(c.uuid)，否则会把 convert_failed/landing_failed
// （consume 有行但 status=2）误算成 matched。
const eventLandSelect = `
	SUM(CASE WHEN d.send_status=1 AND c.status=1 THEN 1 ELSE 0 END) AS matched,
	SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
	SUM(CASE WHEN d.send_status=1 AND c.status=2 AND c.err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
	SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS missing`

const eventLandJoin = `
	FROM fis_decode_detail d
	LEFT JOIN fis_consume_record c
		ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name`

type reconcileBagRow struct {
	Total         int64 `gorm:"column:total"`
	DecodeSuccess int64 `gorm:"column:decode_success"`
	DecodeFailed  int64 `gorm:"column:decode_failed"`
	DecodePartial int64 `gorm:"column:decode_partial"`
}

type reconcileEventParseLandRow struct {
	Expected      int64 `gorm:"column:expected"`
	SendFailed    int64 `gorm:"column:send_failed"`
	ParseFailed   int64 `gorm:"column:parse_failed"`
	Matched       int64 `gorm:"column:matched"`
	ConvertFailed int64 `gorm:"column:convert_failed"`
	LandingFailed int64 `gorm:"column:landing_failed"`
	Missing       int64 `gorm:"column:missing"`
}

type reconcileExtraConsumeRow struct {
	ExtraConsume int64 `gorm:"column:extra_consume"`
}

func (r *reconcileRepo) GetOverview(ctx context.Context, date string) (*biz.ReconcileOverviewData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	bagSql := `SELECT
		COUNT(*) AS total,
		SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) AS decode_success,
		SUM(CASE WHEN status=2 THEN 1 ELSE 0 END) AS decode_failed,
		SUM(CASE WHEN status=3 THEN 1 ELSE 0 END) AS decode_partial
	FROM fis_decode_record
	WHERE dt = ?`

	eventSql := `SELECT
		SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,
		SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed,
		SUM(CASE WHEN d.send_status=3 THEN 1 ELSE 0 END) AS parse_failed,` + eventLandSelect + eventLandJoin + `
	WHERE d.dt = ?`

	extraConsumeSql := `SELECT COUNT(*) AS extra_consume
	FROM fis_consume_record c
	LEFT JOIN fis_decode_detail d
		ON c.dt = d.dt AND c.md5 = d.md5 AND c.uuid = d.uuid AND c.module_name = d.module_name
	WHERE c.dt = ? AND d.uuid IS NULL`

	var (
		bagRow   reconcileBagRow
		eventRow reconcileEventParseLandRow
		extraRow reconcileExtraConsumeRow
		errs     [3]error
	)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		errs[0] = db.Raw(bagSql, date).Scan(&bagRow).Error
	}()
	go func() {
		defer wg.Done()
		errs[1] = db.Raw(eventSql, date).Scan(&eventRow).Error
	}()
	go func() {
		defer wg.Done()
		errs[2] = db.Raw(extraConsumeSql, date).Scan(&extraRow).Error
	}()
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return &biz.ReconcileOverviewData{
		Bag: biz.ReconcileOverviewBag{
			Total:             bagRow.Total,
			DecodeSuccess:     bagRow.DecodeSuccess,
			DecodeFailed:      bagRow.DecodeFailed,
			DecodePartial:     bagRow.DecodePartial,
			DecodeSuccessRate: reconcilePct(bagRow.DecodeSuccess, bagRow.Total),
		},
		EventParse: biz.ReconcileOverviewEventParse{
			Expected:        eventRow.Expected,
			SendFailed:      eventRow.SendFailed,
			ParseFailed:     eventRow.ParseFailed,
			SendFailedRate:  reconcilePct(eventRow.SendFailed, eventRow.Expected+eventRow.SendFailed+eventRow.ParseFailed),
			ParseFailedRate: reconcilePct(eventRow.ParseFailed, eventRow.Expected+eventRow.SendFailed+eventRow.ParseFailed),
		},
		EventLand: biz.ReconcileOverviewEventLand{
			Matched:       eventRow.Matched,
			ConvertFailed: eventRow.ConvertFailed,
			LandingFailed: eventRow.LandingFailed,
			Missing:       eventRow.Missing,
			MatchRate:     reconcilePct(eventRow.Matched, eventRow.Expected),
		},
		ExtraConsume: extraRow.ExtraConsume,
	}, nil
}

type reconcileTrendRow struct {
	Dt            time.Time `gorm:"column:dt"`
	Expected      int64     `gorm:"column:expected"`
	Matched       int64     `gorm:"column:matched"`
	ConvertFailed int64     `gorm:"column:convert_failed"`
	LandingFailed int64     `gorm:"column:landing_failed"`
	Missing       int64     `gorm:"column:missing"`
}

func (r *reconcileRepo) GetTrend(ctx context.Context, startDt, endDt string) (*biz.ReconcileTrendData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	sql := `SELECT d.dt,
		SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,` + eventLandSelect + eventLandJoin + `
	WHERE d.dt BETWEEN ? AND ?
	GROUP BY d.dt
	ORDER BY d.dt`

	var rows []*reconcileTrendRow
	if err := db.Raw(sql, startDt, endDt).Scan(&rows).Error; err != nil {
		return nil, err
	}

	points := make([]*biz.ReconcileTrendPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, &biz.ReconcileTrendPoint{
			Dt:            row.Dt.Format("2006-01-02"),
			Expected:      row.Expected,
			Matched:       row.Matched,
			ConvertFailed: row.ConvertFailed,
			LandingFailed: row.LandingFailed,
			Missing:       row.Missing,
			MatchRate:     reconcilePct(row.Matched, row.Expected),
		})
	}
	return &biz.ReconcileTrendData{Points: points}, nil
}

type reconcileModuleRow struct {
	ModuleName    string `gorm:"column:module_name"`
	Expected      int64  `gorm:"column:expected"`
	Matched       int64  `gorm:"column:matched"`
	ConvertFailed int64  `gorm:"column:convert_failed"`
	LandingFailed int64  `gorm:"column:landing_failed"`
	Missing       int64  `gorm:"column:missing"`
	SendFailed    int64  `gorm:"column:send_failed"`
	ParseFailed   int64  `gorm:"column:parse_failed"`
}

func (r *reconcileRepo) GetModule(ctx context.Context, date, project string) ([]*biz.ReconcileModuleItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	conds := []string{"d.dt = ?"}
	args := []interface{}{date}
	if project != "" {
		conds = append(conds, "d.project = ?")
		args = append(args, project)
	}

	sql := `SELECT d.module_name,
		SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,` + eventLandSelect + `,
		SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed,
		SUM(CASE WHEN d.send_status=3 THEN 1 ELSE 0 END) AS parse_failed` + eventLandJoin + `
	WHERE ` + strings.Join(conds, " AND ") + `
	GROUP BY d.module_name
	ORDER BY missing DESC`

	var rows []*reconcileModuleRow
	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*biz.ReconcileModuleItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.ReconcileModuleItem{
			ModuleName:    row.ModuleName,
			Expected:      row.Expected,
			Matched:       row.Matched,
			ConvertFailed: row.ConvertFailed,
			LandingFailed: row.LandingFailed,
			Missing:       row.Missing,
			SendFailed:    row.SendFailed,
			ParseFailed:   row.ParseFailed,
			MatchRate:     reconcilePct(row.Matched, row.Expected),
		})
	}
	return list, nil
}

type reconcileProjectRow struct {
	Project       string `gorm:"column:project"`
	Expected      int64  `gorm:"column:expected"`
	Matched       int64  `gorm:"column:matched"`
	ConvertFailed int64  `gorm:"column:convert_failed"`
	LandingFailed int64  `gorm:"column:landing_failed"`
	Missing       int64  `gorm:"column:missing"`
}

func (r *reconcileRepo) GetProject(ctx context.Context, date, orderBy string, limit int) ([]*biz.ReconcileProjectItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	orderCol := "missing"
	if orderBy == "expected" {
		orderCol = "expected"
	}

	sql := `SELECT d.project,
		SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS expected,` + eventLandSelect + eventLandJoin + `
	WHERE d.dt = ?
	GROUP BY d.project
	ORDER BY ` + orderCol + ` DESC
	LIMIT ?`

	var rows []*reconcileProjectRow
	if err := db.Raw(sql, date, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*biz.ReconcileProjectItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.ReconcileProjectItem{
			Project:       row.Project,
			Expected:      row.Expected,
			Matched:       row.Matched,
			ConvertFailed: row.ConvertFailed,
			LandingFailed: row.LandingFailed,
			Missing:       row.Missing,
			MatchRate:     reconcilePct(row.Matched, row.Expected),
		})
	}
	return list, nil
}

type reconcileDecodeStatusRow struct {
	Status int8  `gorm:"column:status"`
	Stage  int8  `gorm:"column:stage"`
	Count  int64 `gorm:"column:cnt"`
}

func (r *reconcileRepo) GetDecodeStatus(ctx context.Context, date string) ([]*biz.ReconcileDecodeStatusItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	sql := `SELECT status, stage, COUNT(*) AS cnt
	FROM fis_decode_record
	WHERE dt = ?
	GROUP BY status, stage
	ORDER BY status, stage`

	var rows []*reconcileDecodeStatusRow
	if err := db.Raw(sql, date).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*biz.ReconcileDecodeStatusItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.ReconcileDecodeStatusItem{
			Status: row.Status,
			Stage:  row.Stage,
			Count:  row.Count,
		})
	}
	return list, nil
}

type reconcileMd5CountRow struct {
	Md5        string `gorm:"column:md5"`
	ModuleName string `gorm:"column:module_name"`
	Count      int64  `gorm:"column:count"`
}

type reconcileDecodeFailedMd5Row struct {
	Md5             string `gorm:"column:md5"`
	Status          int8   `gorm:"column:status"`
	Stage           int8   `gorm:"column:stage"`
	ErrorMsg        string `gorm:"column:error_msg"`
	ParsedLineCount int    `gorm:"column:parsed_line_count"`
}

func (r *reconcileRepo) ListDiffMd5(ctx context.Context, date, diffType string, limit int) ([]*biz.ReconcileMd5Item, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	switch diffType {
	case "decode_failed":
		sql := `SELECT md5, status, stage, error_msg, parsed_line_count
		FROM fis_decode_record
		WHERE dt = ? AND status IN (2, 3)
		ORDER BY updated_at DESC
		LIMIT ?`
		var rows []*reconcileDecodeFailedMd5Row
		if err := db.Raw(sql, date, limit).Scan(&rows).Error; err != nil {
			return nil, err
		}
		list := make([]*biz.ReconcileMd5Item, 0, len(rows))
		for _, row := range rows {
			list = append(list, &biz.ReconcileMd5Item{
				Md5:             row.Md5,
				Status:          row.Status,
				Stage:           row.Stage,
				ErrorMsg:        row.ErrorMsg,
				ParsedLineCount: row.ParsedLineCount,
			})
		}
		return list, nil

	case "convert_failed", "landing_failed":
		prefix := "3:%"
		if diffType == "landing_failed" {
			prefix = "6:%"
		}
		sql := `SELECT c.md5,
			ANY_VALUE(c.module_name) AS module_name,
			COUNT(*) AS count
		FROM fis_consume_record c
		WHERE c.dt = ? AND c.status = 2 AND c.err_detail LIKE ?
		GROUP BY c.md5
		ORDER BY count DESC
		LIMIT ?`
		var rows []*reconcileMd5CountRow
		if err := db.Raw(sql, date, prefix, limit).Scan(&rows).Error; err != nil {
			return nil, err
		}
		list := make([]*biz.ReconcileMd5Item, 0, len(rows))
		for _, row := range rows {
			list = append(list, &biz.ReconcileMd5Item{
				Md5:        row.Md5,
				ModuleName: row.ModuleName,
				Count:      row.Count,
			})
		}
		return list, nil

	default: // "missing"
		sql := `SELECT d.md5,
			ANY_VALUE(d.module_name) AS module_name,
			SUM(CASE WHEN d.send_status=1 AND c.uuid IS NULL THEN 1 ELSE 0 END) AS count
		FROM fis_decode_detail d
		LEFT JOIN fis_consume_record c
			ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
		WHERE d.dt = ?
		GROUP BY d.md5
		HAVING count > 0
		ORDER BY count DESC
		LIMIT ?`
		var rows []*reconcileMd5CountRow
		if err := db.Raw(sql, date, limit).Scan(&rows).Error; err != nil {
			return nil, err
		}
		list := make([]*biz.ReconcileMd5Item, 0, len(rows))
		for _, row := range rows {
			list = append(list, &biz.ReconcileMd5Item{
				Md5:        row.Md5,
				ModuleName: row.ModuleName,
				Count:      row.Count,
			})
		}
		return list, nil
	}
}

type reconcileMd5DetailRow struct {
	Uuid          string `gorm:"column:uuid"`
	ModuleName    string `gorm:"column:module_name"`
	SendStatus    int8   `gorm:"column:send_status"`
	ErrDetail     string `gorm:"column:err_detail"`
	Consumed      int8   `gorm:"column:consumed"`
	ConsumeStatus int8   `gorm:"column:consume_status"`
	ConsumeErr    string `gorm:"column:consume_err"`
}

func (r *reconcileRepo) GetMd5Detail(ctx context.Context, date, md5 string) ([]*biz.ReconcileMd5DetailItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	sql := `SELECT
		d.uuid, d.module_name, d.send_status, d.err_detail,
		CASE WHEN c.uuid IS NOT NULL THEN 1 ELSE 0 END AS consumed,
		c.status AS consume_status, c.err_detail AS consume_err
	FROM fis_decode_detail d
	LEFT JOIN fis_consume_record c
		ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
	WHERE d.dt = ? AND d.md5 = ?
	ORDER BY d.module_name, d.uuid`

	var rows []*reconcileMd5DetailRow
	if err := db.Raw(sql, date, md5).Scan(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]*biz.ReconcileMd5DetailItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.ReconcileMd5DetailItem{
			Uuid:          row.Uuid,
			ModuleName:    row.ModuleName,
			SendStatus:    row.SendStatus,
			ErrDetail:     row.ErrDetail,
			Consumed:      row.Consumed == 1,
			ConsumeStatus: row.ConsumeStatus,
			ConsumeErr:    row.ConsumeErr,
		})
	}
	return list, nil
}

type reconcileEventRow struct {
	Md5            string `gorm:"column:md5"`
	Uuid           string `gorm:"column:uuid"`
	ModuleName     string `gorm:"column:module_name"`
	Project        string `gorm:"column:project"`
	Status         int8   `gorm:"column:status"`
	ErrDetail      string `gorm:"column:err_detail"`
	DetailModule   string `gorm:"column:detail_module"`
	ConsumeModule  string `gorm:"column:consume_module"`
	DetailProject  string `gorm:"column:detail_project"`
	ConsumeProject string `gorm:"column:consume_project"`
}

type reconcileCountRow struct {
	Cnt int64 `gorm:"column:cnt"`
}

func toBizReconcileEventItems(rows []*reconcileEventRow) []*biz.ReconcileEventItem {
	list := make([]*biz.ReconcileEventItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.ReconcileEventItem{
			Md5:            row.Md5,
			Uuid:           row.Uuid,
			ModuleName:     row.ModuleName,
			Project:        row.Project,
			Status:         row.Status,
			ErrDetail:      row.ErrDetail,
			DetailModule:   row.DetailModule,
			ConsumeModule:  row.ConsumeModule,
			DetailProject:  row.DetailProject,
			ConsumeProject: row.ConsumeProject,
		})
	}
	return list
}

func (r *reconcileRepo) ListEventList(ctx context.Context, date, moduleName, project, eventType string, page, pageSize int) ([]*biz.ReconcileEventItem, int64, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	offset := (page - 1) * pageSize
	switch eventType {
	case "missing":
		return r.listMissingEvents(db, date, moduleName, project, pageSize, offset)
	case "extra":
		return r.listExtraEvents(db, date, moduleName, project, pageSize, offset)
	case "send_failed":
		return r.listSendFailedEvents(db, date, moduleName, project, pageSize, offset)
	case "parse_failed":
		return r.listParseFailedEvents(db, date, moduleName, project, pageSize, offset)
	case "convert_failed":
		return r.listConvertFailedEvents(db, date, moduleName, project, pageSize, offset)
	case "landing_failed":
		return r.listLandingFailedEvents(db, date, moduleName, project, pageSize, offset)
	case "mismatched":
		return r.listMismatchedEvents(db, date, moduleName, project, pageSize, offset)
	default:
		return nil, 0, biz.ErrInvalidEventType
	}
}

// listMissingEvents: detail 有、consume 没有
func (r *reconcileRepo) listMissingEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"d.dt = ?", "d.send_status = 1", "c.uuid IS NULL"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "d.module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "d.project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt
	FROM fis_decode_detail d
	LEFT JOIN fis_consume_record c ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
	WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT d.md5, d.uuid, d.module_name, d.project, d.send_status AS status, d.err_detail
	FROM fis_decode_detail d
	LEFT JOIN fis_consume_record c ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
	WHERE ` + where + `
	ORDER BY d.md5, d.uuid
	LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*reconcileEventRow
	if err := db.Raw(listSql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toBizReconcileEventItems(rows), countRow.Cnt, nil
}

// listExtraEvents: consume 有、detail 没有
func (r *reconcileRepo) listExtraEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"c.dt = ?", "d.uuid IS NULL"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "c.module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "c.project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt
	FROM fis_consume_record c
	LEFT JOIN fis_decode_detail d ON c.dt = d.dt AND c.md5 = d.md5 AND c.uuid = d.uuid AND c.module_name = d.module_name
	WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT c.md5, c.uuid, c.module_name, c.project, c.status, c.err_detail
	FROM fis_consume_record c
	LEFT JOIN fis_decode_detail d ON c.dt = d.dt AND c.md5 = d.md5 AND c.uuid = d.uuid AND c.module_name = d.module_name
	WHERE ` + where + `
	ORDER BY c.md5, c.uuid
	LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*reconcileEventRow
	if err := db.Raw(listSql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toBizReconcileEventItems(rows), countRow.Cnt, nil
}

// listSendFailedEvents: 上游发送下游失败（fis_decode_detail.send_status=2）
func (r *reconcileRepo) listSendFailedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "send_status = 2"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_decode_detail WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, project, send_status AS status, err_detail
	FROM fis_decode_detail
	WHERE ` + where + `
	ORDER BY md5, uuid
	LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*reconcileEventRow
	if err := db.Raw(listSql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toBizReconcileEventItems(rows), countRow.Cnt, nil
}

// listParseFailedEvents: 未解析出可对账event（fis_decode_detail.send_status=3）
func (r *reconcileRepo) listParseFailedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "send_status = 3"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_decode_detail WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, project, send_status AS status, err_detail
	FROM fis_decode_detail
	WHERE ` + where + `
	ORDER BY md5, uuid
	LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*reconcileEventRow
	if err := db.Raw(listSql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toBizReconcileEventItems(rows), countRow.Cnt, nil
}

// listConsumeFailedEvents: fis_consume_record.status=2 按 err_detail 前缀过滤（转换失败/落库失败共用）
func (r *reconcileRepo) listConsumeFailedEvents(db *gorm.DB, date, moduleName, project, errDetailPrefix string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "status = 2", "err_detail LIKE ?"}
	args := []interface{}{date, errDetailPrefix}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_consume_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, project, status, err_detail
	FROM fis_consume_record
	WHERE ` + where + `
	ORDER BY md5, uuid
	LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*reconcileEventRow
	if err := db.Raw(listSql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toBizReconcileEventItems(rows), countRow.Cnt, nil
}

func (r *reconcileRepo) listConvertFailedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	return r.listConsumeFailedEvents(db, date, moduleName, project, "3:%", limit, offset)
}

func (r *reconcileRepo) listLandingFailedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	return r.listConsumeFailedEvents(db, date, moduleName, project, "6:%", limit, offset)
}

// listMismatchedEvents: 解码明细与消费记录都存在，但 project 不一致（module_name 已经是 JOIN 等值条件）
func (r *reconcileRepo) listMismatchedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"d.dt = ?", "d.project <> c.project"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "d.module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "d.project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt
	FROM fis_decode_detail d
	JOIN fis_consume_record c ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
	WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT d.md5, d.uuid, d.module_name AS detail_module, c.module_name AS consume_module,
		d.project AS detail_project, c.project AS consume_project
	FROM fis_decode_detail d
	JOIN fis_consume_record c ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
	WHERE ` + where + `
	ORDER BY d.md5, d.uuid
	LIMIT ? OFFSET ?`
	listArgs := append(append([]interface{}{}, args...), limit, offset)
	var rows []*reconcileEventRow
	if err := db.Raw(listSql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return toBizReconcileEventItems(rows), countRow.Cnt, nil
}

type reconcileRecordConsistencyRow struct {
	Md5             string `gorm:"column:md5"`
	ParsedLineCount int    `gorm:"column:parsed_line_count"`
	DetailCount     int64  `gorm:"column:detail_count"`
	Diff            int    `gorm:"column:diff"`
}

func (r *reconcileRepo) GetRecordConsistency(ctx context.Context, date string, limit int) ([]*biz.ReconcileRecordConsistencyItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	sql := `SELECT r.md5, r.parsed_line_count, COUNT(d.uuid) AS detail_count, r.parsed_line_count - COUNT(d.uuid) AS diff
	FROM fis_decode_record r
	LEFT JOIN fis_decode_detail d ON r.dt = d.dt AND r.md5 = d.md5
	WHERE r.dt = ? AND r.status IN (1, 3)
	GROUP BY r.md5, r.parsed_line_count
	HAVING diff <> 0
	ORDER BY ABS(diff) DESC
	LIMIT ?`

	var rows []*reconcileRecordConsistencyRow
	if err := db.Raw(sql, date, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*biz.ReconcileRecordConsistencyItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, &biz.ReconcileRecordConsistencyItem{
			Md5:             row.Md5,
			ParsedLineCount: row.ParsedLineCount,
			DetailCount:     row.DetailCount,
			Diff:            row.Diff,
		})
	}
	return list, nil
}

type reconcileUuidSourceRow struct {
	UuidSource string `gorm:"column:uuid_source"`
	Expected   int64  `gorm:"column:expected"`
	Matched    int64  `gorm:"column:matched"`
	Missing    int64  `gorm:"column:missing"`
}

func (r *reconcileRepo) GetUuidSource(ctx context.Context, date string) ([]*biz.ReconcileUuidSourceItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	// matched 用 c.status=1 而不是 COUNT(c.uuid)，理由同 eventLandSelect：不能把
	// convert_failed/landing_failed 的 consume 行误算成 matched。
	sql := `SELECT
		CASE WHEN d.uuid LIKE 'gen:%' THEN 'gen_fallback' ELSE 'real' END AS uuid_source,
		COUNT(*) AS expected,
		SUM(CASE WHEN c.status=1 THEN 1 ELSE 0 END) AS matched,
		SUM(CASE WHEN c.uuid IS NULL THEN 1 ELSE 0 END) AS missing
	FROM fis_decode_detail d
	LEFT JOIN fis_consume_record c ON d.dt = c.dt AND d.md5 = c.md5 AND d.uuid = c.uuid AND d.module_name = c.module_name
	WHERE d.dt = ? AND d.send_status = 1
	GROUP BY uuid_source`

	var rows []*reconcileUuidSourceRow
	if err := db.Raw(sql, date).Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*biz.ReconcileUuidSourceItem, 0, len(rows))
	for _, row := range rows {
		item := &biz.ReconcileUuidSourceItem{
			UuidSource: row.UuidSource,
			Expected:   row.Expected,
			Matched:    row.Matched,
			Missing:    row.Missing,
		}
		if row.Expected > 0 {
			rate := reconcilePct(row.Matched, row.Expected)
			item.MatchRate = &rate
		}
		list = append(list, item)
	}
	return list, nil
}

type reconcileDecodeFailedRow struct {
	Stage          int8   `gorm:"column:stage"`
	Count          int64  `gorm:"column:count"`
	SampleErrorMsg string `gorm:"column:sample_error_msg"`
}

type reconcileModuleErrDetailRow struct {
	ModuleName string `gorm:"column:module_name"`
	ErrDetail  string `gorm:"column:err_detail"`
	Count      int64  `gorm:"column:count"`
}

func (r *reconcileRepo) GetFailureSummary(ctx context.Context, date string) (*biz.ReconcileFailureSummaryData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	decodeFailedSql := `SELECT stage, COUNT(*) AS count, ANY_VALUE(error_msg) AS sample_error_msg
	FROM fis_decode_record
	WHERE dt = ? AND status IN (2, 3)
	GROUP BY stage
	ORDER BY count DESC`

	sendFailedSql := `SELECT module_name, err_detail, COUNT(*) AS count
	FROM fis_decode_detail
	WHERE dt = ? AND send_status = 2
	GROUP BY module_name, err_detail
	ORDER BY count DESC
	LIMIT 50`

	parseFailedSql := `SELECT module_name, err_detail, COUNT(*) AS count
	FROM fis_decode_detail
	WHERE dt = ? AND send_status = 3
	GROUP BY module_name, err_detail
	ORDER BY count DESC
	LIMIT 50`

	convertFailedSql := `SELECT module_name, err_detail, COUNT(*) AS count
	FROM fis_consume_record
	WHERE dt = ? AND status = 2 AND err_detail LIKE '3:%'
	GROUP BY module_name, err_detail
	ORDER BY count DESC
	LIMIT 50`

	landingFailedSql := `SELECT module_name, err_detail, COUNT(*) AS count
	FROM fis_consume_record
	WHERE dt = ? AND status = 2 AND err_detail LIKE '6:%'
	GROUP BY module_name, err_detail
	ORDER BY count DESC
	LIMIT 50`

	var (
		decodeFailedRows  []*reconcileDecodeFailedRow
		sendFailedRows    []*reconcileModuleErrDetailRow
		parseFailedRows   []*reconcileModuleErrDetailRow
		convertFailedRows []*reconcileModuleErrDetailRow
		landingFailedRows []*reconcileModuleErrDetailRow
		errs              [5]error
	)
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		errs[0] = db.Raw(decodeFailedSql, date).Scan(&decodeFailedRows).Error
	}()
	go func() {
		defer wg.Done()
		errs[1] = db.Raw(sendFailedSql, date).Scan(&sendFailedRows).Error
	}()
	go func() {
		defer wg.Done()
		errs[2] = db.Raw(convertFailedSql, date).Scan(&convertFailedRows).Error
	}()
	go func() {
		defer wg.Done()
		errs[3] = db.Raw(landingFailedSql, date).Scan(&landingFailedRows).Error
	}()
	go func() {
		defer wg.Done()
		errs[4] = db.Raw(parseFailedSql, date).Scan(&parseFailedRows).Error
	}()
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	data := &biz.ReconcileFailureSummaryData{
		DecodeFailed:  make([]*biz.ReconcileDecodeFailedItem, 0, len(decodeFailedRows)),
		SendFailed:    make([]*biz.ReconcileSendFailedItem, 0, len(sendFailedRows)),
		ParseFailed:   make([]*biz.ReconcileParseFailedItem, 0, len(parseFailedRows)),
		ConvertFailed: make([]*biz.ReconcileConvertFailedItem, 0, len(convertFailedRows)),
		LandingFailed: make([]*biz.ReconcileLandingFailedItem, 0, len(landingFailedRows)),
	}
	for _, row := range decodeFailedRows {
		data.DecodeFailed = append(data.DecodeFailed, &biz.ReconcileDecodeFailedItem{
			Stage:          row.Stage,
			Count:          row.Count,
			SampleErrorMsg: row.SampleErrorMsg,
		})
	}
	for _, row := range sendFailedRows {
		data.SendFailed = append(data.SendFailed, &biz.ReconcileSendFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	for _, row := range parseFailedRows {
		data.ParseFailed = append(data.ParseFailed, &biz.ReconcileParseFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	for _, row := range convertFailedRows {
		data.ConvertFailed = append(data.ConvertFailed, &biz.ReconcileConvertFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	for _, row := range landingFailedRows {
		data.LandingFailed = append(data.LandingFailed, &biz.ReconcileLandingFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	return data, nil
}

type reconcilePipelineBagRow struct {
	Total           int64 `gorm:"column:total"`
	ParseSuccess    int64 `gorm:"column:parse_success"`
	DecodeSuccess   int64 `gorm:"column:decode_success"`
	DecodePartial   int64 `gorm:"column:decode_partial"`
	DecodeFailed    int64 `gorm:"column:decode_failed"`
	StatusLineCount int64 `gorm:"column:status_line_count"`
	SkipLineCount   int64 `gorm:"column:skip_line_count"`
	ParsedLineCount int64 `gorm:"column:parsed_line_count"`
}

type reconcilePipelineEventParseRow struct {
	ParseSuccess int64 `gorm:"column:parse_success"`
	ParseFailed  int64 `gorm:"column:parse_failed"`
	SendSuccess  int64 `gorm:"column:send_success"`
	SendFailed   int64 `gorm:"column:send_failed"`
}

type reconcilePipelineEventLandRow struct {
	Matched       int64 `gorm:"column:matched"`
	ConvertFailed int64 `gorm:"column:convert_failed"`
	LandingFailed int64 `gorm:"column:landing_failed"`
	Missing       int64 `gorm:"column:missing"`
}

// reconcilePipelineBagConds 构造 fis_decode_record（bag 级，无 module_name 列）的过滤条件
func reconcilePipelineBagConds(date, project, md5 string) ([]string, []interface{}) {
	conds := []string{"dt = ?"}
	args := []interface{}{date}
	if project != "" {
		conds = append(conds, "project = ?")
		args = append(args, project)
	}
	if md5 != "" {
		conds = append(conds, "md5 = ?")
		args = append(args, md5)
	}
	return conds, args
}

// reconcilePipelineEventConds 构造 fis_decode_detail/fis_consume_record（event 级）的过滤条件，
// prefix 用于区分是否要给列名加表别名（如 JOIN 查询里的 "d."）
func reconcilePipelineEventConds(prefix, date, project, moduleName, md5 string) ([]string, []interface{}) {
	conds := []string{prefix + "dt = ?"}
	args := []interface{}{date}
	if project != "" {
		conds = append(conds, prefix+"project = ?")
		args = append(args, project)
	}
	if moduleName != "" {
		conds = append(conds, prefix+"module_name = ?")
		args = append(args, moduleName)
	}
	if md5 != "" {
		conds = append(conds, prefix+"md5 = ?")
		args = append(args, md5)
	}
	return conds, args
}

func (r *reconcileRepo) GetPipelineTree(ctx context.Context, date, project, moduleName, md5 string) (*biz.ReconcilePipelineTreeData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	bagConds, bagArgs := reconcilePipelineBagConds(date, project, md5)
	bagWhere := strings.Join(bagConds, " AND ")

	eventConds, eventArgs := reconcilePipelineEventConds("d.", date, project, moduleName, md5)
	eventWhere := strings.Join(eventConds, " AND ")

	plainConds, plainArgs := reconcilePipelineEventConds("", date, project, moduleName, md5)
	plainWhere := strings.Join(plainConds, " AND ")

	bagSql := `SELECT
		COUNT(*) AS total,
		SUM(CASE WHEN status IN (1,3) THEN 1 ELSE 0 END) AS parse_success,
		SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) AS decode_success,
		SUM(CASE WHEN status=3 THEN 1 ELSE 0 END) AS decode_partial,
		SUM(CASE WHEN status=2 THEN 1 ELSE 0 END) AS decode_failed,
		SUM(status_line_count) AS status_line_count,
		SUM(skip_line_count) AS skip_line_count,
		SUM(parsed_line_count) AS parsed_line_count
	FROM fis_decode_record
	WHERE ` + bagWhere

	eventParseSql := `SELECT
		SUM(CASE WHEN d.send_status IN (1,2) THEN 1 ELSE 0 END) AS parse_success,
		SUM(CASE WHEN d.send_status=3 THEN 1 ELSE 0 END) AS parse_failed,
		SUM(CASE WHEN d.send_status=1 THEN 1 ELSE 0 END) AS send_success,
		SUM(CASE WHEN d.send_status=2 THEN 1 ELSE 0 END) AS send_failed
	FROM fis_decode_detail d
	WHERE ` + eventWhere

	eventLandSql := `SELECT` + eventLandSelect + eventLandJoin + `
	WHERE ` + eventWhere

	var (
		bagRow   reconcilePipelineBagRow
		eventRow reconcilePipelineEventParseRow
		landRow  reconcilePipelineEventLandRow
		errs     [3]error
	)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		errs[0] = db.Raw(bagSql, bagArgs...).Scan(&bagRow).Error
	}()
	go func() {
		defer wg.Done()
		errs[1] = db.Raw(eventParseSql, eventArgs...).Scan(&eventRow).Error
	}()
	go func() {
		defer wg.Done()
		errs[2] = db.Raw(eventLandSql, eventArgs...).Scan(&landRow).Error
	}()
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	data := &biz.ReconcilePipelineTreeData{
		Bag: biz.ReconcilePipelineBagData{
			Total:           bagRow.Total,
			ParseSuccess:    bagRow.ParseSuccess,
			DecodeSuccess:   bagRow.DecodeSuccess,
			DecodePartial:   bagRow.DecodePartial,
			DecodeFailed:    bagRow.DecodeFailed,
			StatusLineCount: bagRow.StatusLineCount,
			SkipLineCount:   bagRow.SkipLineCount,
			ParsedLineCount: bagRow.ParsedLineCount,
		},
		EventParse: biz.ReconcilePipelineEventParseData{
			ParseSuccess: eventRow.ParseSuccess,
			ParseFailed:  eventRow.ParseFailed,
			SendSuccess:  eventRow.SendSuccess,
			SendFailed:   eventRow.SendFailed,
		},
		EventLand: biz.ReconcilePipelineEventLandData{
			Matched:       landRow.Matched,
			ConvertFailed: landRow.ConvertFailed,
			LandingFailed: landRow.LandingFailed,
			Missing:       landRow.Missing,
		},
	}

	var (
		bagFailureRows    []*reconcileDecodeFailedRow
		parseFailedRows   []*reconcileModuleErrDetailRow
		sendFailedRows    []*reconcileModuleErrDetailRow
		convertFailedRows []*reconcileModuleErrDetailRow
		landingFailedRows []*reconcileModuleErrDetailRow
		failureErrs       [5]error
	)
	var fwg sync.WaitGroup
	if bagRow.DecodeFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT stage, COUNT(*) AS count, ANY_VALUE(error_msg) AS sample_error_msg
			FROM fis_decode_record
			WHERE ` + bagWhere + ` AND status = 2
			GROUP BY stage
			ORDER BY count DESC`
			failureErrs[0] = db.Raw(sql, bagArgs...).Scan(&bagFailureRows).Error
		}()
	}
	if eventRow.ParseFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, err_detail, COUNT(*) AS count
			FROM fis_decode_detail
			WHERE ` + plainWhere + ` AND send_status = 3
			GROUP BY module_name, err_detail
			ORDER BY count DESC
			LIMIT 50`
			failureErrs[1] = db.Raw(sql, plainArgs...).Scan(&parseFailedRows).Error
		}()
	}
	if eventRow.SendFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, err_detail, COUNT(*) AS count
			FROM fis_decode_detail
			WHERE ` + plainWhere + ` AND send_status = 2
			GROUP BY module_name, err_detail
			ORDER BY count DESC
			LIMIT 50`
			failureErrs[2] = db.Raw(sql, plainArgs...).Scan(&sendFailedRows).Error
		}()
	}
	if landRow.ConvertFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, err_detail, COUNT(*) AS count
			FROM fis_consume_record
			WHERE ` + plainWhere + ` AND status = 2 AND err_detail LIKE '3:%'
			GROUP BY module_name, err_detail
			ORDER BY count DESC
			LIMIT 50`
			failureErrs[3] = db.Raw(sql, plainArgs...).Scan(&convertFailedRows).Error
		}()
	}
	if landRow.LandingFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, err_detail, COUNT(*) AS count
			FROM fis_consume_record
			WHERE ` + plainWhere + ` AND status = 2 AND err_detail LIKE '6:%'
			GROUP BY module_name, err_detail
			ORDER BY count DESC
			LIMIT 50`
			failureErrs[4] = db.Raw(sql, plainArgs...).Scan(&landingFailedRows).Error
		}()
	}
	fwg.Wait()
	for _, err := range failureErrs {
		if err != nil {
			return nil, err
		}
	}

	for _, row := range bagFailureRows {
		data.BagFailureReasons = append(data.BagFailureReasons, &biz.ReconcileDecodeFailedItem{
			Stage:          row.Stage,
			Count:          row.Count,
			SampleErrorMsg: row.SampleErrorMsg,
		})
	}
	for _, row := range parseFailedRows {
		data.EventParseFailureReasons = append(data.EventParseFailureReasons, &biz.ReconcileParseFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	for _, row := range sendFailedRows {
		data.EventSendFailureReasons = append(data.EventSendFailureReasons, &biz.ReconcileSendFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	for _, row := range convertFailedRows {
		data.ConvertFailedReasons = append(data.ConvertFailedReasons, &biz.ReconcileConvertFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}
	for _, row := range landingFailedRows {
		data.LandingFailedReasons = append(data.LandingFailedReasons, &biz.ReconcileLandingFailedItem{
			ModuleName: row.ModuleName,
			ErrDetail:  row.ErrDetail,
			Count:      row.Count,
		})
	}

	return data, nil
}
