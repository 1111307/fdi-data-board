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

// eventLandSelect 是 L3 落库层 matched/convert_failed/landing_failed/missing 的公共表达式。
// 合表后 decode_* 与 consume_* 列在同一行，不再需要 JOIN：同一 (dt,md5,uuid,module_name)
// 的两侧由 AGGREGATE KEY + REPLACE_IF_NOT_NULL 自动合并到一行，直接在行内判断即可。
// matched 必须用 consume_status=1 而不是 consume_status IS NOT NULL，否则会把
// convert_failed/landing_failed（consume_status=2）误算成 matched。
// 四项都以 decode_send_status=1 为前提：只有上游发送成功的 event 才会真正到达下游落库。
const eventLandSelect = `
		SUM(CASE WHEN decode_send_status=1 AND consume_status=1 THEN 1 ELSE 0 END) AS matched,
		SUM(CASE WHEN decode_send_status=1 AND consume_status=2 AND consume_err_detail LIKE '3:%' THEN 1 ELSE 0 END) AS convert_failed,
		SUM(CASE WHEN decode_send_status=1 AND consume_status=2 AND consume_err_detail LIKE '6:%' THEN 1 ELSE 0 END) AS landing_failed,
		SUM(CASE WHEN decode_send_status=1 AND consume_status IS NULL THEN 1 ELSE 0 END) AS missing`

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

	// L1 bag 层：uuid='' 的哨兵行，record_* 列由 decode 侧写入。
	bagSql := `SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN record_status=1 THEN 1 ELSE 0 END) AS decode_success,
			SUM(CASE WHEN record_status=2 THEN 1 ELSE 0 END) AS decode_failed,
			SUM(CASE WHEN record_status=3 THEN 1 ELSE 0 END) AS decode_partial
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid = ''`

	// L2+L3：event 行（uuid<>''），decode_send_status/expected 与 consume_* 落库结果同行统计。
	eventSql := `SELECT
			SUM(CASE WHEN decode_send_status=1 THEN 1 ELSE 0 END) AS expected,
			SUM(CASE WHEN decode_send_status=2 THEN 1 ELSE 0 END) AS send_failed,
			SUM(CASE WHEN decode_send_status=3 THEN 1 ELSE 0 END) AS parse_failed,` + eventLandSelect + `
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> ''`

	// extra：consume 侧写了行（consume_status 非空）但 decode 侧没有对应记录（decode_send_status 为空）。
	extraConsumeSql := `SELECT COUNT(*) AS extra_consume
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> '' AND consume_status IS NOT NULL AND decode_send_status IS NULL`

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

	sql := `SELECT dt,
		SUM(CASE WHEN decode_send_status=1 THEN 1 ELSE 0 END) AS expected,` + eventLandSelect + `
		FROM fis_reconcile_record
		WHERE dt BETWEEN ? AND ? AND uuid <> ''
		GROUP BY dt
		ORDER BY dt`

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

	conds := []string{"dt = ?", "uuid <> ''"}
	args := []interface{}{date}
	if project != "" {
		// event 行的 project 来自 decode 侧（与 decode_send_status 同源），用 decode_project。
		conds = append(conds, "decode_project = ?")
		args = append(args, project)
	}

	sql := `SELECT module_name,
		SUM(CASE WHEN decode_send_status=1 THEN 1 ELSE 0 END) AS expected,` + eventLandSelect + `,
		SUM(CASE WHEN decode_send_status=2 THEN 1 ELSE 0 END) AS send_failed,
		SUM(CASE WHEN decode_send_status=3 THEN 1 ELSE 0 END) AS parse_failed
		FROM fis_reconcile_record
		WHERE ` + strings.Join(conds, " AND ") + `
		GROUP BY module_name
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

	sql := `SELECT decode_project AS project,
		SUM(CASE WHEN decode_send_status=1 THEN 1 ELSE 0 END) AS expected,` + eventLandSelect + `
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> ''
		GROUP BY decode_project
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

	sql := `SELECT record_status AS status, record_stage AS stage, COUNT(*) AS cnt
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid = ''
		GROUP BY record_status, record_stage
		ORDER BY record_status, record_stage`

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
		sql := `SELECT md5, record_status AS status, record_stage AS stage, record_error_msg AS error_msg, record_parsed_line_count AS parsed_line_count
			FROM fis_reconcile_record
			WHERE dt = ? AND uuid = '' AND record_status IN (2, 3)
			ORDER BY record_updated_at DESC
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
		sql := `SELECT md5,
			ANY_VALUE(module_name) AS module_name,
			COUNT(*) AS count
			FROM fis_reconcile_record
			WHERE dt = ? AND uuid <> '' AND consume_status = 2 AND consume_err_detail LIKE ?
			GROUP BY md5
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
		// decode 发送成功（decode_send_status=1）但 consume 侧还没写到这行（consume_status IS NULL）。
		// 每行即一条 missing，COUNT(*) 直接是 missing 条数。
		sql := `SELECT md5,
			ANY_VALUE(module_name) AS module_name,
			COUNT(*) AS count
			FROM fis_reconcile_record
			WHERE dt = ? AND uuid <> '' AND decode_send_status = 1 AND consume_status IS NULL
			GROUP BY md5
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

	// 同一 event 行里 decode_* 与 consume_* 并存：consume_status 非空即代表下游已消费到。
	sql := `SELECT
			uuid, module_name, decode_send_status AS send_status, decode_err_detail AS err_detail,
			CASE WHEN consume_status IS NOT NULL THEN 1 ELSE 0 END AS consumed,
			consume_status, consume_err_detail AS consume_err
		FROM fis_reconcile_record
		WHERE dt = ? AND md5 = ? AND uuid <> ''
		ORDER BY module_name, uuid`

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

// listMissingEvents: decode 发送成功但 consume 侧尚未写入（consume_status IS NULL）。
func (r *reconcileRepo) listMissingEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "uuid <> ''", "decode_send_status = 1", "consume_status IS NULL"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "decode_project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_reconcile_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, decode_project AS project, decode_send_status AS status, decode_err_detail AS err_detail
		FROM fis_reconcile_record
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

// listExtraEvents: consume 侧有记录但 decode 侧无对应行（consume_status 非空、decode_send_status 为空）。
// extra 行的 project 取 consume_project（decode 列此时为空）。
func (r *reconcileRepo) listExtraEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "uuid <> ''", "consume_status IS NOT NULL", "decode_send_status IS NULL"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "consume_project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_reconcile_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, consume_project AS project, consume_status AS status, consume_err_detail AS err_detail
		FROM fis_reconcile_record
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

// listSendFailedEvents: 上游发送下游失败（decode_send_status=2）。
func (r *reconcileRepo) listSendFailedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "uuid <> ''", "decode_send_status = 2"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "decode_project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_reconcile_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, decode_project AS project, decode_send_status AS status, decode_err_detail AS err_detail
		FROM fis_reconcile_record
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

// listParseFailedEvents: 未解析出可对账 event（decode_send_status=3）。
func (r *reconcileRepo) listParseFailedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "uuid <> ''", "decode_send_status = 3"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "decode_project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_reconcile_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, decode_project AS project, decode_send_status AS status, decode_err_detail AS err_detail
		FROM fis_reconcile_record
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

// listConsumeFailedEvents: consume_status=2 按 consume_err_detail 前缀过滤（转换失败/落库失败共用）。
func (r *reconcileRepo) listConsumeFailedEvents(db *gorm.DB, date, moduleName, project, errDetailPrefix string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "uuid <> ''", "consume_status = 2", "consume_err_detail LIKE ?"}
	args := []interface{}{date, errDetailPrefix}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "consume_project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_reconcile_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name, consume_project AS project, consume_status AS status, consume_err_detail AS err_detail
		FROM fis_reconcile_record
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

// listMismatchedEvents: decode 与 consume 同行但 project 不一致（module_name 是 AGGREGATE KEY 的一部分，同行必然相等，无需比对）。
func (r *reconcileRepo) listMismatchedEvents(db *gorm.DB, date, moduleName, project string, limit, offset int) ([]*biz.ReconcileEventItem, int64, error) {
	conds := []string{"dt = ?", "uuid <> ''", "decode_project <> consume_project"}
	args := []interface{}{date}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if project != "" {
		conds = append(conds, "decode_project = ?")
		args = append(args, project)
	}
	where := strings.Join(conds, " AND ")

	countSql := `SELECT COUNT(*) AS cnt FROM fis_reconcile_record WHERE ` + where
	var countRow reconcileCountRow
	if err := db.Raw(countSql, args...).Scan(&countRow).Error; err != nil {
		return nil, 0, err
	}
	if countRow.Cnt == 0 {
		return []*biz.ReconcileEventItem{}, 0, nil
	}

	listSql := `SELECT md5, uuid, module_name AS detail_module, module_name AS consume_module,
			decode_project AS detail_project, consume_project AS consume_project
		FROM fis_reconcile_record
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

type reconcileRecordConsistencyRow struct {
	Md5             string `gorm:"column:md5"`
	ParsedLineCount int    `gorm:"column:parsed_line_count"`
	DetailCount     int64  `gorm:"column:detail_count"`
	Diff            int    `gorm:"column:diff"`
}

func (r *reconcileRepo) GetRecordConsistency(ctx context.Context, date string, limit int) ([]*biz.ReconcileRecordConsistencyItem, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	// bag 哨兵行（r, uuid=''）与同 dt+md5 的 event 行（d, uuid<>''）仍是不同行，
	// 需要同表自 JOIN 关联；COUNT(d.decode_send_status) 只计 event 行（bag 行该列为 NULL 不计）。
	sql := `SELECT r.md5, r.record_parsed_line_count AS parsed_line_count, COUNT(d.decode_send_status) AS detail_count,
			r.record_parsed_line_count - COUNT(d.decode_send_status) AS diff
		FROM fis_reconcile_record r
		LEFT JOIN fis_reconcile_record d ON r.dt = d.dt AND r.md5 = d.md5
		WHERE r.dt = ? AND r.uuid = '' AND r.record_status IN (1, 3)
		GROUP BY r.md5, r.record_parsed_line_count
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

	// matched 用 consume_status=1 而不是 consume_status IS NOT NULL，理由同 eventLandSelect：
	// 不能把 convert_failed/landing_failed（consume_status=2）误算成 matched。
	sql := `SELECT
			CASE WHEN uuid LIKE 'gen:%' THEN 'gen_fallback' ELSE 'real' END AS uuid_source,
			COUNT(*) AS expected,
			SUM(CASE WHEN consume_status=1 THEN 1 ELSE 0 END) AS matched,
			SUM(CASE WHEN consume_status IS NULL THEN 1 ELSE 0 END) AS missing
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> '' AND decode_send_status = 1
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

	decodeFailedSql := `SELECT record_stage AS stage, COUNT(*) AS count, ANY_VALUE(record_error_msg) AS sample_error_msg
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid = '' AND record_status IN (2, 3)
		GROUP BY record_stage
		ORDER BY count DESC`

	sendFailedSql := `SELECT module_name, decode_err_detail AS err_detail, COUNT(*) AS count
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> '' AND decode_send_status = 2
		GROUP BY module_name, decode_err_detail
		ORDER BY count DESC
		LIMIT 50`

	parseFailedSql := `SELECT module_name, decode_err_detail AS err_detail, COUNT(*) AS count
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> '' AND decode_send_status = 3
		GROUP BY module_name, decode_err_detail
		ORDER BY count DESC
		LIMIT 50`

	convertFailedSql := `SELECT module_name, consume_err_detail AS err_detail, COUNT(*) AS count
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> '' AND consume_status = 2 AND consume_err_detail LIKE '3:%'
		GROUP BY module_name, consume_err_detail
		ORDER BY count DESC
		LIMIT 50`

	landingFailedSql := `SELECT module_name, consume_err_detail AS err_detail, COUNT(*) AS count
		FROM fis_reconcile_record
		WHERE dt = ? AND uuid <> '' AND consume_status = 2 AND consume_err_detail LIKE '6:%'
		GROUP BY module_name, consume_err_detail
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

// reconcilePipelineBagConds 构造 bag 级哨兵行（uuid=''）的过滤条件，project 用 record_project。
func reconcilePipelineBagConds(date, project, md5 string) ([]string, []interface{}) {
	conds := []string{"dt = ?", "uuid = ''"}
	args := []interface{}{date}
	if project != "" {
		conds = append(conds, "record_project = ?")
		args = append(args, project)
	}
	if md5 != "" {
		conds = append(conds, "md5 = ?")
		args = append(args, md5)
	}
	return conds, args
}

// reconcilePipelineEventConds 构造 event 级行（uuid<>''）的过滤条件。
// 合表后单表查询无需表别名前缀；project 用 decode_project（与 decode_send_status 同源）。
func reconcilePipelineEventConds(date, project, moduleName, md5 string) ([]string, []interface{}) {
	conds := []string{"dt = ?", "uuid <> ''"}
	args := []interface{}{date}
	if project != "" {
		conds = append(conds, "decode_project = ?")
		args = append(args, project)
	}
	if moduleName != "" {
		conds = append(conds, "module_name = ?")
		args = append(args, moduleName)
	}
	if md5 != "" {
		conds = append(conds, "md5 = ?")
		args = append(args, md5)
	}
	return conds, args
}

func (r *reconcileRepo) GetPipelineTree(ctx context.Context, date, project, moduleName, md5 string) (*biz.ReconcilePipelineTreeData, error) {
	db, cancel := r.dorisQuery(ctx)
	defer cancel()

	bagConds, bagArgs := reconcilePipelineBagConds(date, project, md5)
	bagWhere := strings.Join(bagConds, " AND ")

	eventConds, eventArgs := reconcilePipelineEventConds(date, project, moduleName, md5)
	eventWhere := strings.Join(eventConds, " AND ")

	bagSql := `SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN record_status IN (1,3) THEN 1 ELSE 0 END) AS parse_success,
			SUM(CASE WHEN record_status=1 THEN 1 ELSE 0 END) AS decode_success,
			SUM(CASE WHEN record_status=3 THEN 1 ELSE 0 END) AS decode_partial,
			SUM(CASE WHEN record_status=2 THEN 1 ELSE 0 END) AS decode_failed,
			SUM(record_status_line_count) AS status_line_count,
			SUM(record_skip_line_count) AS skip_line_count,
			SUM(record_parsed_line_count) AS parsed_line_count
		FROM fis_reconcile_record
		WHERE ` + bagWhere

	eventParseSql := `SELECT
			SUM(CASE WHEN decode_send_status IN (1,2) THEN 1 ELSE 0 END) AS parse_success,
			SUM(CASE WHEN decode_send_status=3 THEN 1 ELSE 0 END) AS parse_failed,
			SUM(CASE WHEN decode_send_status=1 THEN 1 ELSE 0 END) AS send_success,
			SUM(CASE WHEN decode_send_status=2 THEN 1 ELSE 0 END) AS send_failed
		FROM fis_reconcile_record
		WHERE ` + eventWhere

	eventLandSql := `SELECT` + eventLandSelect + `
		FROM fis_reconcile_record
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
			sql := `SELECT record_stage AS stage, COUNT(*) AS count, ANY_VALUE(record_error_msg) AS sample_error_msg
				FROM fis_reconcile_record
				WHERE ` + bagWhere + ` AND record_status = 2
				GROUP BY record_stage
				ORDER BY count DESC`
			failureErrs[0] = db.Raw(sql, bagArgs...).Scan(&bagFailureRows).Error
		}()
	}
	if eventRow.ParseFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, decode_err_detail AS err_detail, COUNT(*) AS count
				FROM fis_reconcile_record
				WHERE ` + eventWhere + ` AND decode_send_status = 3
				GROUP BY module_name, decode_err_detail
				ORDER BY count DESC
				LIMIT 50`
			failureErrs[1] = db.Raw(sql, eventArgs...).Scan(&parseFailedRows).Error
		}()
	}
	if eventRow.SendFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, decode_err_detail AS err_detail, COUNT(*) AS count
				FROM fis_reconcile_record
				WHERE ` + eventWhere + ` AND decode_send_status = 2
				GROUP BY module_name, decode_err_detail
				ORDER BY count DESC
				LIMIT 50`
			failureErrs[2] = db.Raw(sql, eventArgs...).Scan(&sendFailedRows).Error
		}()
	}
	if landRow.ConvertFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, consume_err_detail AS err_detail, COUNT(*) AS count
				FROM fis_reconcile_record
				WHERE ` + eventWhere + ` AND consume_status = 2 AND consume_err_detail LIKE '3:%'
				GROUP BY module_name, consume_err_detail
				ORDER BY count DESC
				LIMIT 50`
			failureErrs[3] = db.Raw(sql, eventArgs...).Scan(&convertFailedRows).Error
		}()
	}
	if landRow.LandingFailed > 0 {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			sql := `SELECT module_name, consume_err_detail AS err_detail, COUNT(*) AS count
				FROM fis_reconcile_record
				WHERE ` + eventWhere + ` AND consume_status = 2 AND consume_err_detail LIKE '6:%'
				GROUP BY module_name, consume_err_detail
				ORDER BY count DESC
				LIMIT 50`
			failureErrs[4] = db.Raw(sql, eventArgs...).Scan(&landingFailedRows).Error
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
