package biz

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	dashboard_api "fdi_data_board/api/dashboard"
)

var ErrInvalidMd5Type = errors.New("type must be missing, convert_failed, landing_failed or decode_failed")
var ErrMd5Required = errors.New("md5 is required")
var ErrInvalidEventType = errors.New("type must be missing, extra, send_failed, parse_failed, convert_failed, landing_failed or mismatched")
var ErrInvalidOrderBy = errors.New("order_by must be missing or expected")

const defaultMd5ListLimit = 20
const defaultEventListPageSize = 20
const maxEventListPageSize = 200

// ReconcileRepo FIS 对账监控数据仓储接口
type ReconcileRepo interface {
	GetOverview(ctx context.Context, date string) (*ReconcileOverviewData, error)
	GetTrend(ctx context.Context, startDt, endDt string) (*ReconcileTrendData, error)
	GetModule(ctx context.Context, date, project string) ([]*ReconcileModuleItem, error)
	GetProject(ctx context.Context, date, orderBy string, limit int) ([]*ReconcileProjectItem, error)
	GetDecodeStatus(ctx context.Context, date string) ([]*ReconcileDecodeStatusItem, error)
	ListDiffMd5(ctx context.Context, date, diffType string, limit int) ([]*ReconcileMd5Item, error)
	GetMd5Detail(ctx context.Context, date, md5 string) ([]*ReconcileMd5DetailItem, error)
	ListEventList(ctx context.Context, date, moduleName, project, eventType string, page, pageSize int) ([]*ReconcileEventItem, int64, error)
	GetRecordConsistency(ctx context.Context, date string, limit int) ([]*ReconcileRecordConsistencyItem, error)
	GetUuidSource(ctx context.Context, date string) ([]*ReconcileUuidSourceItem, error)
	GetFailureSummary(ctx context.Context, date string) (*ReconcileFailureSummaryData, error)
	GetPipelineTree(ctx context.Context, date, project, moduleName, md5 string) (*ReconcilePipelineTreeData, error)
}

// ReconcileOverviewBag L1 bag 级解码健康度
type ReconcileOverviewBag struct {
	Total             int64
	DecodeSuccess     int64
	DecodeFailed      int64
	DecodePartial     int64
	DecodeSuccessRate float64
}

// ReconcileOverviewEventParse L2 event 级解析/发送健康度
type ReconcileOverviewEventParse struct {
	Expected        int64
	SendFailed      int64
	SendFailedRate  float64
	ParseFailed     int64
	ParseFailedRate float64
}

// ReconcileOverviewEventLand L3 event 级落库健康度
type ReconcileOverviewEventLand struct {
	Matched       int64
	ConvertFailed int64
	LandingFailed int64
	Missing       int64
	MatchRate     float64
}

// ReconcileOverviewData 对账总览聚合结果（三层嵌套）
type ReconcileOverviewData struct {
	Bag          ReconcileOverviewBag
	EventParse   ReconcileOverviewEventParse
	EventLand    ReconcileOverviewEventLand
	ExtraConsume int64
}

// ReconcileTrendPoint 时间趋势单日数据点
type ReconcileTrendPoint struct {
	Dt            string
	Expected      int64
	Matched       int64
	ConvertFailed int64
	LandingFailed int64
	Missing       int64
	MatchRate     float64
}

// ReconcileTrendData 对账时间趋势聚合结果
type ReconcileTrendData struct {
	Points []*ReconcileTrendPoint
}

// ReconcileModuleItem 模块维度对账统计
type ReconcileModuleItem struct {
	ModuleName    string
	Expected      int64
	Matched       int64
	ConvertFailed int64
	LandingFailed int64
	Missing       int64
	SendFailed    int64
	ParseFailed   int64
	MatchRate     float64
}

// ReconcileProjectItem 项目维度对账统计
type ReconcileProjectItem struct {
	Project       string
	Expected      int64
	Matched       int64
	ConvertFailed int64
	LandingFailed int64
	Missing       int64
	MatchRate     float64
}

// ReconcileDecodeStatusItem 上游解码状态分布单条
type ReconcileDecodeStatusItem struct {
	Status int8
	Stage  int8
	Count  int64
}

// ReconcileMd5Item 差异 md5 单条，字段按 type 不同部分为空
type ReconcileMd5Item struct {
	Md5             string
	ModuleName      string
	Count           int64
	Status          int8
	Stage           int8
	ErrorMsg        string
	ParsedLineCount int
}

// ReconcileMd5DetailItem 单 md5 下单条 event 明细
type ReconcileMd5DetailItem struct {
	Uuid          string
	ModuleName    string
	SendStatus    int8
	ErrDetail     string
	Consumed      bool
	ConsumeStatus int8
	ConsumeErr    string
}

// ReconcileEventItem event/uuid 级明细单条，字段按 type 不同部分为空
type ReconcileEventItem struct {
	Md5            string
	Uuid           string
	ModuleName     string
	Project        string
	Status         int8
	ErrDetail      string
	DetailModule   string
	ConsumeModule  string
	DetailProject  string
	ConsumeProject string
}

// ReconcileRecordConsistencyItem record↔detail 一致性单条（解码阶段是否丢行）
type ReconcileRecordConsistencyItem struct {
	Md5             string
	ParsedLineCount int
	DetailCount     int64
	Diff            int
}

// ReconcileUuidSourceItem uuid 来源细分单条，MatchRate 在 Expected=0 时为 nil
type ReconcileUuidSourceItem struct {
	UuidSource string
	Expected   int64
	Matched    int64
	Missing    int64
	MatchRate  *float64
}

// ReconcileDecodeFailedItem L1 解码失败按 stage 聚合单条
type ReconcileDecodeFailedItem struct {
	Stage          int8
	Count          int64
	SampleErrorMsg string
}

// ReconcileSendFailedItem L2 发送下游失败按 module_name+err_detail 聚合单条
type ReconcileSendFailedItem struct {
	ModuleName string
	ErrDetail  string
	Count      int64
}

// ReconcileParseFailedItem L2 未解析出可对账event按 module_name+err_detail 聚合单条
type ReconcileParseFailedItem struct {
	ModuleName string
	ErrDetail  string
	Count      int64
}

// ReconcileConvertFailedItem L3 转换失败按 module_name+err_detail 聚合单条
type ReconcileConvertFailedItem struct {
	ModuleName string
	ErrDetail  string
	Count      int64
}

// ReconcileLandingFailedItem L3 落库失败按 module_name+err_detail 聚合单条
type ReconcileLandingFailedItem struct {
	ModuleName string
	ErrDetail  string
	Count      int64
}

// ReconcileFailureSummaryData 失败汇总聚合结果
type ReconcileFailureSummaryData struct {
	DecodeFailed  []*ReconcileDecodeFailedItem
	SendFailed    []*ReconcileSendFailedItem
	ParseFailed   []*ReconcileParseFailedItem
	ConvertFailed []*ReconcileConvertFailedItem
	LandingFailed []*ReconcileLandingFailedItem
}

// ReconcilePipelineBagData 全链路树 L1 tar 包层聚合结果
type ReconcilePipelineBagData struct {
	Total           int64
	ParseSuccess    int64
	DecodeSuccess   int64
	DecodePartial   int64
	DecodeFailed    int64
	StatusLineCount int64
	SkipLineCount   int64
	ParsedLineCount int64
}

// ReconcilePipelineEventParseData 全链路树 L2 event 解析层聚合结果
type ReconcilePipelineEventParseData struct {
	ParseSuccess int64
	ParseFailed  int64
	SendSuccess  int64
	SendFailed   int64
}

// ReconcilePipelineEventLandData 全链路树 L3 event 落库层聚合结果（仅 send_status=1 的 event）
type ReconcilePipelineEventLandData struct {
	Matched       int64
	ConvertFailed int64
	LandingFailed int64
	Missing       int64
}

// ReconcilePipelineTreeData 全链路树形聚合的原始数据，由 use case 组装成前端可渲染的树
type ReconcilePipelineTreeData struct {
	Bag                      ReconcilePipelineBagData
	BagFailureReasons        []*ReconcileDecodeFailedItem
	EventParse               ReconcilePipelineEventParseData
	EventParseFailureReasons []*ReconcileParseFailedItem
	EventSendFailureReasons  []*ReconcileSendFailedItem
	EventLand                ReconcilePipelineEventLandData
	ConvertFailedReasons     []*ReconcileConvertFailedItem
	LandingFailedReasons     []*ReconcileLandingFailedItem
}

// reconcileStageDesc stage → 描述枚举映射，后端硬编码，不让前端猜数字
var reconcileStageDesc = map[int8]string{
	0: "success",
	1: "下载解包失败",
	2: "文件内容为空",
	3: "模块转换失败",
	4: "转换后无结果",
	5: "发送下游失败",
}

// ReconcileUseCase FIS 对账监控业务用例
type ReconcileUseCase struct {
	repo ReconcileRepo
}

func NewReconcileUseCase(repo ReconcileRepo) *ReconcileUseCase {
	return &ReconcileUseCase{repo: repo}
}

// defaultReconcileDate 缺省取今天，格式 YYYY-MM-DD
func defaultReconcileDate(date string) string {
	if date != "" {
		return date
	}
	return time.Now().Format("2006-01-02")
}

func (uc *ReconcileUseCase) GetOverview(ctx context.Context, req *dashboard_api.ReconcileOverviewRequest) (*dashboard_api.ReconcileOverviewResponse, error) {
	date := defaultReconcileDate(req.Date)
	data, err := uc.repo.GetOverview(ctx, date)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileOverviewResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Bag: dashboard_api.ReconcileOverviewBag{
			Total:             data.Bag.Total,
			DecodeSuccess:     data.Bag.DecodeSuccess,
			DecodeFailed:      data.Bag.DecodeFailed,
			DecodePartial:     data.Bag.DecodePartial,
			DecodeSuccessRate: data.Bag.DecodeSuccessRate,
		},
		EventParse: dashboard_api.ReconcileOverviewEventParse{
			Expected:        data.EventParse.Expected,
			SendFailed:      data.EventParse.SendFailed,
			SendFailedRate:  data.EventParse.SendFailedRate,
			ParseFailed:     data.EventParse.ParseFailed,
			ParseFailedRate: data.EventParse.ParseFailedRate,
		},
		EventLand: dashboard_api.ReconcileOverviewEventLand{
			Matched:       data.EventLand.Matched,
			ConvertFailed: data.EventLand.ConvertFailed,
			LandingFailed: data.EventLand.LandingFailed,
			Missing:       data.EventLand.Missing,
			MatchRate:     data.EventLand.MatchRate,
		},
		ExtraConsume: data.ExtraConsume,
	}, nil
}

func (uc *ReconcileUseCase) GetTrend(ctx context.Context, req *dashboard_api.ReconcileTrendRequest) (*dashboard_api.ReconcileTrendResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	if startDt == "" || endDt == "" {
		endDt = time.Now().Format("2006-01-02")
		startDt = time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	}
	data, err := uc.repo.GetTrend(ctx, startDt, endDt)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileTrendResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Start:        startDt,
		End:          endDt,
		Points:       toApiReconcileTrendPoints(data.Points),
	}, nil
}

func (uc *ReconcileUseCase) GetModule(ctx context.Context, req *dashboard_api.ReconcileModuleRequest) (*dashboard_api.ReconcileModuleResponse, error) {
	date := defaultReconcileDate(req.Date)
	list, err := uc.repo.GetModule(ctx, date, req.Project)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileModuleResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Project:      req.Project,
		Modules:      toApiReconcileModuleItems(list),
	}, nil
}

func (uc *ReconcileUseCase) GetProject(ctx context.Context, req *dashboard_api.ReconcileProjectRequest) (*dashboard_api.ReconcileProjectResponse, error) {
	orderBy := req.OrderBy
	if orderBy == "" {
		orderBy = "missing"
	}
	if orderBy != "missing" && orderBy != "expected" {
		return nil, ErrInvalidOrderBy
	}
	date := defaultReconcileDate(req.Date)
	limit := req.Limit
	if limit <= 0 {
		limit = defaultMd5ListLimit
	}
	list, err := uc.repo.GetProject(ctx, date, orderBy, limit)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileProjectResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Projects:     toApiReconcileProjectItems(list),
	}, nil
}

func (uc *ReconcileUseCase) GetDecodeStatus(ctx context.Context, req *dashboard_api.ReconcileDecodeStatusRequest) (*dashboard_api.ReconcileDecodeStatusResponse, error) {
	date := defaultReconcileDate(req.Date)
	list, err := uc.repo.GetDecodeStatus(ctx, date)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileDecodeStatusResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Items:        toApiReconcileDecodeStatusItems(list),
	}, nil
}

func (uc *ReconcileUseCase) ListMd5(ctx context.Context, req *dashboard_api.ReconcileMd5ListRequest) (*dashboard_api.ReconcileMd5ListResponse, error) {
	diffType := req.Type
	if diffType == "" {
		diffType = "missing"
	}
	switch diffType {
	case "missing", "convert_failed", "landing_failed", "decode_failed":
	default:
		return nil, ErrInvalidMd5Type
	}
	date := defaultReconcileDate(req.Date)
	limit := req.Limit
	if limit <= 0 {
		limit = defaultMd5ListLimit
	}
	list, err := uc.repo.ListDiffMd5(ctx, date, diffType, limit)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileMd5ListResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Type:         diffType,
		Items:        toApiReconcileMd5Items(list),
	}, nil
}

func (uc *ReconcileUseCase) GetMd5Detail(ctx context.Context, req *dashboard_api.ReconcileMd5DetailRequest) (*dashboard_api.ReconcileMd5DetailResponse, error) {
	if req.Md5 == "" {
		return nil, ErrMd5Required
	}
	date := defaultReconcileDate(req.Date)
	list, err := uc.repo.GetMd5Detail(ctx, date, req.Md5)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileMd5DetailResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Md5:          req.Md5,
		Date:         date,
		Items:        toApiReconcileMd5DetailItems(list),
	}, nil
}

func (uc *ReconcileUseCase) ListEventList(ctx context.Context, req *dashboard_api.ReconcileEventListRequest) (*dashboard_api.ReconcileEventListResponse, error) {
	eventType := req.Type
	if eventType == "" {
		eventType = "missing"
	}
	switch eventType {
	case "missing", "extra", "send_failed", "parse_failed", "convert_failed", "landing_failed", "mismatched":
	default:
		return nil, ErrInvalidEventType
	}
	date := defaultReconcileDate(req.Date)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultEventListPageSize
	} else if pageSize > maxEventListPageSize {
		pageSize = maxEventListPageSize
	}
	list, total, err := uc.repo.ListEventList(ctx, date, req.ModuleName, req.Project, eventType, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileEventListResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Type:         eventType,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		Items:        toApiReconcileEventItems(list),
	}, nil
}

func (uc *ReconcileUseCase) GetRecordConsistency(ctx context.Context, req *dashboard_api.ReconcileRecordConsistencyRequest) (*dashboard_api.ReconcileRecordConsistencyResponse, error) {
	date := defaultReconcileDate(req.Date)
	limit := req.Limit
	if limit <= 0 {
		limit = defaultMd5ListLimit
	}
	list, err := uc.repo.GetRecordConsistency(ctx, date, limit)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileRecordConsistencyResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Items:        toApiReconcileRecordConsistencyItems(list),
	}, nil
}

func (uc *ReconcileUseCase) GetUuidSource(ctx context.Context, req *dashboard_api.ReconcileUuidSourceRequest) (*dashboard_api.ReconcileUuidSourceResponse, error) {
	date := defaultReconcileDate(req.Date)
	list, err := uc.repo.GetUuidSource(ctx, date)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileUuidSourceResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Sources:      toApiReconcileUuidSourceItems(list),
	}, nil
}

func (uc *ReconcileUseCase) GetFailureSummary(ctx context.Context, req *dashboard_api.ReconcileFailureSummaryRequest) (*dashboard_api.ReconcileFailureSummaryResponse, error) {
	date := defaultReconcileDate(req.Date)
	data, err := uc.repo.GetFailureSummary(ctx, date)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcileFailureSummaryResponse{
		BaseResponse:  dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:          date,
		DecodeFailed:  toApiReconcileDecodeFailedItems(data.DecodeFailed),
		SendFailed:    toApiReconcileSendFailedItems(data.SendFailed),
		ParseFailed:   toApiReconcileParseFailedItems(data.ParseFailed),
		ConvertFailed: toApiReconcileConvertFailedItems(data.ConvertFailed),
		LandingFailed: toApiReconcileLandingFailedItems(data.LandingFailed),
	}, nil
}

func (uc *ReconcileUseCase) GetPipelineTree(ctx context.Context, req *dashboard_api.ReconcilePipelineTreeRequest) (*dashboard_api.ReconcilePipelineTreeResponse, error) {
	date := defaultReconcileDate(req.Date)
	data, err := uc.repo.GetPipelineTree(ctx, date, req.Project, req.ModuleName, req.Md5)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.ReconcilePipelineTreeResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Date:         date,
		Filters:      toApiReconcilePipelineTreeFilters(req),
		Tree:         buildReconcilePipelineTree(data),
	}, nil
}

// ---------- biz domain → api DTO 映射函数 ----------

func toApiReconcileTrendPoints(list []*ReconcileTrendPoint) []*dashboard_api.ReconcileTrendPoint {
	out := make([]*dashboard_api.ReconcileTrendPoint, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileTrendPoint{
			Dt:            v.Dt,
			Expected:      v.Expected,
			Matched:       v.Matched,
			ConvertFailed: v.ConvertFailed,
			LandingFailed: v.LandingFailed,
			Missing:       v.Missing,
			MatchRate:     v.MatchRate,
		})
	}
	return out
}

func toApiReconcileModuleItems(list []*ReconcileModuleItem) []*dashboard_api.ReconcileModuleItem {
	out := make([]*dashboard_api.ReconcileModuleItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileModuleItem{
			ModuleName:    v.ModuleName,
			Expected:      v.Expected,
			Matched:       v.Matched,
			ConvertFailed: v.ConvertFailed,
			LandingFailed: v.LandingFailed,
			Missing:       v.Missing,
			SendFailed:    v.SendFailed,
			ParseFailed:   v.ParseFailed,
			MatchRate:     v.MatchRate,
		})
	}
	return out
}

func toApiReconcileProjectItems(list []*ReconcileProjectItem) []*dashboard_api.ReconcileProjectItem {
	out := make([]*dashboard_api.ReconcileProjectItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileProjectItem{
			Project:       v.Project,
			Expected:      v.Expected,
			Matched:       v.Matched,
			ConvertFailed: v.ConvertFailed,
			LandingFailed: v.LandingFailed,
			Missing:       v.Missing,
			MatchRate:     v.MatchRate,
		})
	}
	return out
}

func toApiReconcileDecodeStatusItems(list []*ReconcileDecodeStatusItem) []*dashboard_api.ReconcileDecodeStatusItem {
	out := make([]*dashboard_api.ReconcileDecodeStatusItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileDecodeStatusItem{
			Status: v.Status,
			Stage:  v.Stage,
			Count:  v.Count,
		})
	}
	return out
}

func toApiReconcileMd5Items(list []*ReconcileMd5Item) []*dashboard_api.ReconcileMd5Item {
	out := make([]*dashboard_api.ReconcileMd5Item, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileMd5Item{
			Md5:             v.Md5,
			ModuleName:      v.ModuleName,
			Count:           v.Count,
			Status:          v.Status,
			Stage:           v.Stage,
			ErrorMsg:        v.ErrorMsg,
			ParsedLineCount: v.ParsedLineCount,
		})
	}
	return out
}

func toApiReconcileMd5DetailItems(list []*ReconcileMd5DetailItem) []*dashboard_api.ReconcileMd5DetailItem {
	out := make([]*dashboard_api.ReconcileMd5DetailItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileMd5DetailItem{
			Uuid:          v.Uuid,
			ModuleName:    v.ModuleName,
			SendStatus:    v.SendStatus,
			ErrDetail:     v.ErrDetail,
			Consumed:      v.Consumed,
			ConsumeStatus: v.ConsumeStatus,
			ConsumeErr:    v.ConsumeErr,
		})
	}
	return out
}

func toApiReconcileEventItems(list []*ReconcileEventItem) []*dashboard_api.ReconcileEventItem {
	out := make([]*dashboard_api.ReconcileEventItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileEventItem{
			Md5:            v.Md5,
			Uuid:           v.Uuid,
			ModuleName:     v.ModuleName,
			Project:        v.Project,
			Status:         v.Status,
			ErrDetail:      v.ErrDetail,
			DetailModule:   v.DetailModule,
			ConsumeModule:  v.ConsumeModule,
			DetailProject:  v.DetailProject,
			ConsumeProject: v.ConsumeProject,
		})
	}
	return out
}

func toApiReconcileRecordConsistencyItems(list []*ReconcileRecordConsistencyItem) []*dashboard_api.ReconcileRecordConsistencyItem {
	out := make([]*dashboard_api.ReconcileRecordConsistencyItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileRecordConsistencyItem{
			Md5:             v.Md5,
			ParsedLineCount: v.ParsedLineCount,
			DetailCount:     v.DetailCount,
			Diff:            v.Diff,
		})
	}
	return out
}

func toApiReconcileUuidSourceItems(list []*ReconcileUuidSourceItem) []*dashboard_api.ReconcileUuidSourceItem {
	out := make([]*dashboard_api.ReconcileUuidSourceItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileUuidSourceItem{
			UuidSource: v.UuidSource,
			Expected:   v.Expected,
			Matched:    v.Matched,
			Missing:    v.Missing,
			MatchRate:  v.MatchRate,
		})
	}
	return out
}

func toApiReconcileDecodeFailedItems(list []*ReconcileDecodeFailedItem) []*dashboard_api.ReconcileDecodeFailedItem {
	out := make([]*dashboard_api.ReconcileDecodeFailedItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileDecodeFailedItem{
			Stage:          v.Stage,
			StageDesc:      reconcileStageDesc[v.Stage],
			Count:          v.Count,
			SampleErrorMsg: v.SampleErrorMsg,
		})
	}
	return out
}

func toApiReconcileSendFailedItems(list []*ReconcileSendFailedItem) []*dashboard_api.ReconcileSendFailedItem {
	out := make([]*dashboard_api.ReconcileSendFailedItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileSendFailedItem{
			ModuleName: v.ModuleName,
			ErrDetail:  v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcileParseFailedItems(list []*ReconcileParseFailedItem) []*dashboard_api.ReconcileParseFailedItem {
	out := make([]*dashboard_api.ReconcileParseFailedItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileParseFailedItem{
			ModuleName: v.ModuleName,
			ErrDetail:  v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcileConvertFailedItems(list []*ReconcileConvertFailedItem) []*dashboard_api.ReconcileConvertFailedItem {
	out := make([]*dashboard_api.ReconcileConvertFailedItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileConvertFailedItem{
			ModuleName: v.ModuleName,
			ErrDetail:  v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcileLandingFailedItems(list []*ReconcileLandingFailedItem) []*dashboard_api.ReconcileLandingFailedItem {
	out := make([]*dashboard_api.ReconcileLandingFailedItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileLandingFailedItem{
			ModuleName: v.ModuleName,
			ErrDetail:  v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcilePipelineTreeFilters(req *dashboard_api.ReconcilePipelineTreeRequest) dashboard_api.ReconcilePipelineTreeFilters {
	f := dashboard_api.ReconcilePipelineTreeFilters{}
	if req.Project != "" {
		project := req.Project
		f.Project = &project
	}
	if req.ModuleName != "" {
		moduleName := req.ModuleName
		f.ModuleName = &moduleName
	}
	if req.Md5 != "" {
		md5 := req.Md5
		f.Md5 = &md5
	}
	return f
}

// reconcilePipelineRate 计算子节点相对于同一分支总量的占比，分母为 0 时返回 0
func reconcilePipelineRate(a, b int64) *float64 {
	rate := 0.0
	if b != 0 {
		rate = math.Round(float64(a)/float64(b)*10000) / 10000
	}
	return &rate
}

func toApiReconcilePipelineReasonsFromDecodeFailed(list []*ReconcileDecodeFailedItem) []*dashboard_api.ReconcileFailureReason {
	if len(list) == 0 {
		return nil
	}
	out := make([]*dashboard_api.ReconcileFailureReason, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileFailureReason{
			Code:   strconv.Itoa(int(v.Stage)),
			Desc:   reconcileStageDesc[v.Stage],
			Sample: v.SampleErrorMsg,
			Count:  v.Count,
		})
	}
	return out
}

func toApiReconcilePipelineReasonsFromSendFailed(list []*ReconcileSendFailedItem) []*dashboard_api.ReconcileFailureReason {
	if len(list) == 0 {
		return nil
	}
	out := make([]*dashboard_api.ReconcileFailureReason, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileFailureReason{
			Code:       "5",
			Desc:       "发送下游Kafka失败",
			ModuleName: v.ModuleName,
			Sample:     v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcilePipelineReasonsFromParseFailed(list []*ReconcileParseFailedItem) []*dashboard_api.ReconcileFailureReason {
	if len(list) == 0 {
		return nil
	}
	out := make([]*dashboard_api.ReconcileFailureReason, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileFailureReason{
			Code:       "4",
			Desc:       "未解析出可对账event",
			ModuleName: v.ModuleName,
			Sample:     v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcilePipelineReasonsFromConvertFailed(list []*ReconcileConvertFailedItem) []*dashboard_api.ReconcileFailureReason {
	if len(list) == 0 {
		return nil
	}
	out := make([]*dashboard_api.ReconcileFailureReason, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileFailureReason{
			Code:       "3",
			Desc:       "下游转换失败",
			ModuleName: v.ModuleName,
			Sample:     v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

func toApiReconcilePipelineReasonsFromLandingFailed(list []*ReconcileLandingFailedItem) []*dashboard_api.ReconcileFailureReason {
	if len(list) == 0 {
		return nil
	}
	out := make([]*dashboard_api.ReconcileFailureReason, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.ReconcileFailureReason{
			Code:       "6",
			Desc:       "落库失败(DLQ/丢弃)",
			ModuleName: v.ModuleName,
			Sample:     v.ErrDetail,
			Count:      v.Count,
		})
	}
	return out
}

// buildReconcilePipelineTree 把 L1/L2/L3 三层的扁平聚合数据组装成前端可直接渲染的树形结构。
// tar 包（L1）与 event（L2/L3）是不同粒度的统计单位，event 分支的 rate 相对其自身分支总量计算，
// 而不是相对父节点 tar_parse_success.count（两者数值不可比，仅用于视觉分组）。
func buildReconcilePipelineTree(data *ReconcilePipelineTreeData) *dashboard_api.ReconcilePipelineNode {
	eventTotal := data.EventParse.ParseSuccess + data.EventParse.ParseFailed

	eventLandMatched := &dashboard_api.ReconcilePipelineNode{
		Key:    "event_land_matched",
		Label:  "落库成功",
		Status: "success",
		Count:  data.EventLand.Matched,
		Rate:   reconcilePipelineRate(data.EventLand.Matched, data.EventParse.SendSuccess),
	}
	eventLandConvertFailed := &dashboard_api.ReconcilePipelineNode{
		Key:            "event_land_convert_failed",
		Label:          "转换失败",
		Status:         "failed",
		Count:          data.EventLand.ConvertFailed,
		Rate:           reconcilePipelineRate(data.EventLand.ConvertFailed, data.EventParse.SendSuccess),
		FailureReasons: toApiReconcilePipelineReasonsFromConvertFailed(data.ConvertFailedReasons),
	}
	eventLandLandingFailed := &dashboard_api.ReconcilePipelineNode{
		Key:            "event_land_landing_failed",
		Label:          "落库失败",
		Status:         "failed",
		Count:          data.EventLand.LandingFailed,
		Rate:           reconcilePipelineRate(data.EventLand.LandingFailed, data.EventParse.SendSuccess),
		FailureReasons: toApiReconcilePipelineReasonsFromLandingFailed(data.LandingFailedReasons),
	}
	eventLandMissing := &dashboard_api.ReconcilePipelineNode{
		Key:    "event_land_missing",
		Label:  "待落库/丢库",
		Status: "unknown",
		Count:  data.EventLand.Missing,
		Rate:   reconcilePipelineRate(data.EventLand.Missing, data.EventParse.SendSuccess),
	}

	eventSendSuccess := &dashboard_api.ReconcilePipelineNode{
		Key:      "event_send_success",
		Label:    "发送kafka成功",
		Status:   "success",
		Count:    data.EventParse.SendSuccess,
		Rate:     reconcilePipelineRate(data.EventParse.SendSuccess, data.EventParse.ParseSuccess),
		Children: []*dashboard_api.ReconcilePipelineNode{eventLandMatched, eventLandConvertFailed, eventLandLandingFailed, eventLandMissing},
	}
	eventSendFailed := &dashboard_api.ReconcilePipelineNode{
		Key:            "event_send_failed",
		Label:          "发送kafka失败",
		Status:         "failed",
		Count:          data.EventParse.SendFailed,
		Rate:           reconcilePipelineRate(data.EventParse.SendFailed, data.EventParse.ParseSuccess),
		FailureReasons: toApiReconcilePipelineReasonsFromSendFailed(data.EventSendFailureReasons),
	}

	eventParseSuccess := &dashboard_api.ReconcilePipelineNode{
		Key:      "event_parse_success",
		Label:    "解析成功event",
		Status:   "success",
		Count:    data.EventParse.ParseSuccess,
		Rate:     reconcilePipelineRate(data.EventParse.ParseSuccess, eventTotal),
		Children: []*dashboard_api.ReconcilePipelineNode{eventSendSuccess, eventSendFailed},
	}
	eventParseFailed := &dashboard_api.ReconcilePipelineNode{
		Key:            "event_parse_failed",
		Label:          "解析失败event",
		Status:         "failed",
		Count:          data.EventParse.ParseFailed,
		Rate:           reconcilePipelineRate(data.EventParse.ParseFailed, eventTotal),
		FailureReasons: toApiReconcilePipelineReasonsFromParseFailed(data.EventParseFailureReasons),
	}

	tarParseSuccess := &dashboard_api.ReconcilePipelineNode{
		Key:    "tar_parse_success",
		Label:  "解析成功tar包",
		Status: "success",
		Count:  data.Bag.ParseSuccess,
		Rate:   reconcilePipelineRate(data.Bag.ParseSuccess, data.Bag.Total),
		Meta: map[string]int64{
			"decode_success":    data.Bag.DecodeSuccess,
			"decode_partial":    data.Bag.DecodePartial,
			"status_line_count": data.Bag.StatusLineCount,
			"skip_line_count":   data.Bag.SkipLineCount,
			"parsed_line_count": data.Bag.ParsedLineCount,
		},
		Children: []*dashboard_api.ReconcilePipelineNode{eventParseSuccess, eventParseFailed},
	}
	tarParseFailed := &dashboard_api.ReconcilePipelineNode{
		Key:            "tar_parse_failed",
		Label:          "解析失败tar包",
		Status:         "failed",
		Count:          data.Bag.DecodeFailed,
		Rate:           reconcilePipelineRate(data.Bag.DecodeFailed, data.Bag.Total),
		FailureReasons: toApiReconcilePipelineReasonsFromDecodeFailed(data.BagFailureReasons),
	}

	return &dashboard_api.ReconcilePipelineNode{
		Key:      "tar_received",
		Label:    "收到的tar包",
		Status:   "root",
		Count:    data.Bag.Total,
		Children: []*dashboard_api.ReconcilePipelineNode{tarParseSuccess, tarParseFailed},
	}
}
