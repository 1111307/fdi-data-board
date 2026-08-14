package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	dashboard_api "fdi_data_board/api/dashboard"
)

var ErrInvalidDateRange = errors.New("invalid date format, expected YYYY-MM-DD")
var ErrMissingRequired = errors.New("filter_name, start_dt and end_dt are required")

const (
	defaultPageSize = 50
	maxPageSize     = 500
	maxDateRange    = 365
)

// FoDashboardRepo FO Dashboard 数据仓储接口
type FoDashboardRepo interface {
	GetFunnel(ctx context.Context, param *FunnelParam) (*FunnelData, error)
	ListFffRunning(ctx context.Context, param *FffRunningParam) ([]*FffRunningItem, int64, error)
	ListFffTrigger(ctx context.Context, param *FffTriggerParam) ([]*FffTriggerItem, int64, error)
	ListFffClose(ctx context.Context, param *FffCloseParam) ([]*FffCloseItem, int64, error)
	ListFdrTrigger(ctx context.Context, param *FdrTriggerParam) ([]*FdrTriggerItem, int64, error)
	ListFclTrigger(ctx context.Context, param *FclTriggerParam) ([]*FclTriggerItem, int64, error)
	ListUuidDetail(ctx context.Context, param *UuidDetailParam) ([]*UuidDetailItem, int64, error)
	GetCloseReason(ctx context.Context, param *CloseReasonParam) ([]*CloseReasonItem, error)
	GetStageTrend(ctx context.Context, param *StageTrendParam) (*StageTrendData, error)
	GetDimensions(ctx context.Context) (*FoDimensions, error)
	GetFffRunningTrend(ctx context.Context, param *FffRunningTrendParam) (*FffRunningTrendData, error)
	GetRunningOverview(ctx context.Context, param *FffRunningParam) (*FoRunningOverviewData, error)
	GetFffOverview(ctx context.Context, param *FffTriggerParam) (*FoFffOverviewData, error)
	GetFffFailReason(ctx context.Context, param *FffTriggerParam) ([]*DoFailReasonItem, error)
}

// FunnelParam 数采全链路分析查询参数
type FunnelParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// FunnelData biz 层聚合结果
type FunnelData struct {
	Stat    *FunnelStat
	FffFail []*FunnelFailReason
	FdrFail []*FunnelFailReason
	FclFail []*FunnelFailReason
}

// StageTrendParam 三阶段触发趋势查询参数
type StageTrendParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// StageTrendData biz 层聚合结果
type StageTrendData struct {
	Dates []string
	Fff   []*StageTrendSeries
	Fdr   []*StageTrendSeries
	Fcl   []*StageTrendSeries
}

// CloseReasonParam 算子关闭原因分布查询参数
type CloseReasonParam struct {
	FilterName  string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// UuidDetailParam 全链路明细查询参数
type UuidDetailParam struct {
	FilterName  string
	EventNames  []string // 多选
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
	OnlyFail    bool
	StageFilter string // fff_discard/fdr_discard/fcl_discard/fcl_success
	Page        int
	PageSize    int
}

// FclTriggerParam FCL 上传明细查询参数
type FclTriggerParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FdrTriggerParam FDR 落盘明细查询参数
type FdrTriggerParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FffCloseParam 筛选器关闭明细查询参数
type FffCloseParam struct {
	FilterName  string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FffTriggerParam 筛选器触发明细查询参数
type FffTriggerParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FoDimensions 维度枚举数据（FO/DO 公共）
type FoDimensions struct {
	FilterNames  []string
	EventNames   []string
	ProjectNames []string
	CarTypes     []string
}

// FffRunningParam 筛选器运行明细查询参数
type FffRunningParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FffRunningTrendParam 算子活跃车辆趋势查询参数
type FffRunningTrendParam struct {
	FilterName  string // 必填：算子名称
	ProjectName string // 选填：项目名称
	CarTypes    []string
	StartDt     string // 必填：开始日期
	EndDt       string // 必填：结束日期
}

// FoDashboardUseCase FO Dashboard 业务用例
type FoDashboardUseCase struct {
	repo FoDashboardRepo
}

func NewFoDashboardUseCase(repo FoDashboardRepo) *FoDashboardUseCase {
	return &FoDashboardUseCase{repo: repo}
}

func (uc *FoDashboardUseCase) ListFffTrigger(ctx context.Context, req *dashboard_api.FffTriggerRequest) (*dashboard_api.FffTriggerResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}

	param := &FffTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
		Page:        page,
		PageSize:    pageSize,
	}

	list, total, err := uc.repo.ListFffTrigger(ctx, param)
	if err != nil {
		return nil, err
	}

	return &dashboard_api.FffTriggerResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		List:         toApiFffTriggerItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) ListFffClose(ctx context.Context, req *dashboard_api.FffCloseRequest) (*dashboard_api.FffCloseResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}

	param := &FffCloseParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
		Page:        page,
		PageSize:    pageSize,
	}

	list, total, err := uc.repo.ListFffClose(ctx, param)
	if err != nil {
		return nil, err
	}

	return &dashboard_api.FffCloseResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		List:         toApiFffCloseItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) ListFdrTrigger(ctx context.Context, req *dashboard_api.FdrTriggerRequest) (*dashboard_api.FdrTriggerResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}

	param := &FdrTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
		Page:        page,
		PageSize:    pageSize,
	}

	list, total, err := uc.repo.ListFdrTrigger(ctx, param)
	if err != nil {
		return nil, err
	}

	return &dashboard_api.FdrTriggerResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		List:         toApiFdrTriggerItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) ListFclTrigger(ctx context.Context, req *dashboard_api.FclTriggerRequest) (*dashboard_api.FclTriggerResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}

	param := &FclTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
		Page:        page,
		PageSize:    pageSize,
	}

	list, total, err := uc.repo.ListFclTrigger(ctx, param)
	if err != nil {
		return nil, err
	}

	return &dashboard_api.FclTriggerResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		List:         toApiFclTriggerItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) ListUuidDetail(ctx context.Context, req *dashboard_api.UuidDetailRequest) (*dashboard_api.UuidDetailResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}

	param := &UuidDetailParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
		OnlyFail:    req.OnlyFail == 1,
		StageFilter: req.StageFilter,
		Page:        page,
		PageSize:    pageSize,
	}

	list, total, err := uc.repo.ListUuidDetail(ctx, param)
	if err != nil {
		return nil, err
	}

	return &dashboard_api.UuidDetailResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		List:         toApiUuidDetailItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) GetCloseReason(ctx context.Context, req *dashboard_api.CloseReasonRequest) (*dashboard_api.CloseReasonResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &CloseReasonParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetCloseReason(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.CloseReasonResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiCloseReasonItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) GetFunnel(ctx context.Context, req *dashboard_api.FunnelRequest) (*dashboard_api.FunnelResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &FunnelParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFunnel(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.FunnelResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Stat:         toApiFunnelStat(data.Stat),
		FffFail:      toApiFunnelFailReasons(data.FffFail),
		FdrFail:      toApiFunnelFailReasons(data.FdrFail),
		FclFail:      toApiFunnelFailReasons(data.FclFail),
	}, nil
}

func (uc *FoDashboardUseCase) GetStageTrend(ctx context.Context, req *dashboard_api.StageTrendRequest) (*dashboard_api.StageTrendResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &StageTrendParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetStageTrend(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.StageTrendResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:        data.Dates,
		Fff:          toApiStageTrendSeries(data.Fff),
		Fdr:          toApiStageTrendSeries(data.Fdr),
		Fcl:          toApiStageTrendSeries(data.Fcl),
	}, nil
}

func (uc *FoDashboardUseCase) GetDimensions(ctx context.Context) (*dashboard_api.FoDimensionsResponse, error) {
	dims, err := uc.repo.GetDimensions(ctx)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.FoDimensionsResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		FilterNames:  dims.FilterNames,
		EventNames:   dims.EventNames,
		ProjectNames: dims.ProjectNames,
		CarTypes:     dims.CarTypes,
	}, nil
}

func (uc *FoDashboardUseCase) ListFffRunning(ctx context.Context, req *dashboard_api.FffRunningRequest) (*dashboard_api.FffRunningResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}

	param := &FffRunningParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
		Page:        page,
		PageSize:    pageSize,
	}

	list, total, err := uc.repo.ListFffRunning(ctx, param)
	if err != nil {
		return nil, err
	}

	return &dashboard_api.FffRunningResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
		List:         toApiFffRunningItems(list),
	}, nil
}

func (uc *FoDashboardUseCase) GetFffRunningTrend(ctx context.Context, req *dashboard_api.FffRunningTrendRequest) (*dashboard_api.FffRunningTrendResponse, error) {
	if req.FilterName == "" {
		return nil, ErrMissingRequired
	}
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	if startDt == "" || endDt == "" {
		return nil, ErrMissingRequired
	}

	param := &FffRunningTrendParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFffRunningTrend(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.FffRunningTrendResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:        data.Dates,
		Counts:       data.Counts,
	}, nil
}

func (uc *FoDashboardUseCase) GetRunningOverview(ctx context.Context, req *dashboard_api.FffRunningRequest) (*dashboard_api.FoRunningOverviewResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &FffRunningParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetRunningOverview(ctx, param)
	if err != nil {
		return nil, err
	}
	switchOnRatio := 0.0
	if total := data.SwitchOnTotal + data.SwitchOffTotal; total > 0 {
		switchOnRatio = float64(data.SwitchOnTotal) / float64(total) * 100
	}
	return &dashboard_api.FoRunningOverviewResponse{
		BaseResponse:   dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		RunningTotal:   data.RunningTotal,
		VehicleTotal:   data.VehicleTotal,
		SwitchOnTotal:  data.SwitchOnTotal,
		SwitchOffTotal: data.SwitchOffTotal,
		SwitchOnRatio:  switchOnRatio,
		RunningSuccess: data.RunningSuccess,
		RunningFailed:  data.RunningFailed,
		FilterCount:    data.FilterCount,
	}, nil
}

func (uc *FoDashboardUseCase) GetFffOverview(ctx context.Context, req *dashboard_api.FffTriggerRequest) (*dashboard_api.FoFffOverviewResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &FffTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFffOverview(ctx, param)
	if err != nil {
		return nil, err
	}
	successRate := 0.0
	if data.TriggerTotal > 0 {
		successRate = float64(data.TriggerSuccess) / float64(data.TriggerTotal) * 100
	}
	return &dashboard_api.FoFffOverviewResponse{
		BaseResponse:       dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		TriggerTotal:       data.TriggerTotal,
		TriggerSuccess:     data.TriggerSuccess,
		TriggerFailed:      data.TriggerFailed,
		TriggerSuccessRate: successRate,
	}, nil
}

func (uc *FoDashboardUseCase) GetFffFailReason(ctx context.Context, req *dashboard_api.FffTriggerRequest) (*dashboard_api.DoFailReasonResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &FffTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetFffFailReason(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFailReasonResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoFailReasonItems(list),
	}, nil
}

// normalizePage 统一校正分页参数
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// normalizeDateRange 校验日期范围：格式校验、顺序校正、跨度限制
func normalizeDateRange(startDt, endDt string) (string, string, error) {
	const layout = "2006-01-02"

	if startDt == "" && endDt == "" {
		return "", "", nil
	}

	start, err1 := time.Parse(layout, startDt)
	end, err2 := time.Parse(layout, endDt)

	if err1 != nil || err2 != nil {
		return "", "", ErrInvalidDateRange
	}

	if start.After(end) {
		start, end = end, start
	}

	if int(end.Sub(start).Hours()/24) > maxDateRange {
		end = start.AddDate(0, 0, maxDateRange)
	}

	return start.Format(layout), end.Format(layout), nil
}

// splitEventNames 将逗号分隔的字符串拆分为切片，空字符串返回 nil
func splitEventNames(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	for _, e := range strings.Split(raw, ",") {
		if e = strings.TrimSpace(e); e != "" {
			result = append(result, e)
		}
	}
	return result
}

// ---------- biz domain → api DTO 映射函数 ----------

func toApiFffRunningItems(list []*FffRunningItem) []*dashboard_api.FffRunningItem {
	out := make([]*dashboard_api.FffRunningItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.FffRunningItem{
			Dt:              v.Dt,
			FilterName:      v.FilterName,
			AnonymousId:     v.AnonymousId,
			TimestampUtc:    v.TimestampUtc,
			CreateAt:        v.CreateAt,
			CollectType:     v.CollectType,
			SwVersion:       v.SwVersion,
			ProjectName:     v.ProjectName,
			CarType:         v.CarType,
			VehicleSource:   v.VehicleSource,
			SwitchOn:        v.SwitchOn,
			Version:         v.Version,
			OnAutopilot:     v.OnAutopilot,
			FunctionMode:    v.FunctionMode,
			Status:          v.Status,
			FdiProjectName:  v.FdiProjectName,
			ProjectCarType:  v.ProjectCarType,
			VehicleSourceCn: v.VehicleSourceCn,
		})
	}
	return out
}

func toApiFffTriggerItems(list []*FffTriggerItem) []*dashboard_api.FffTriggerItem {
	out := make([]*dashboard_api.FffTriggerItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.FffTriggerItem{
			Dt:              v.Dt,
			Uuid:            v.Uuid,
			EventName:       v.EventName,
			AnonymousId:     v.AnonymousId,
			TimestampUtc:    v.TimestampUtc,
			CreateAt:        v.CreateAt,
			TriggerTime:     v.TriggerTime,
			UtcDiffUs:       v.UtcDiffUs,
			Before:          v.Before,
			After:           v.After,
			FilterName:      v.FilterName,
			TriggerType:     v.TriggerType,
			CollectType:     v.CollectType,
			Status:          v.Status,
			OnAutopilot:     v.OnAutopilot,
			FunctionMode:    v.FunctionMode,
			SwVersion:       v.SwVersion,
			ProjectName:     v.ProjectName,
			CarType:         v.CarType,
			VehicleSource:   v.VehicleSource,
			Bj02Lat:         v.Bj02Lat,
			Bj02Lon:         v.Bj02Lon,
			RoadType:        v.RoadType,
			FdiProjectName:  v.FdiProjectName,
			ProjectCarType:  v.ProjectCarType,
			VehicleSourceCn: v.VehicleSourceCn,
			Tags:            v.Tags,
			Detail:          v.Detail,
		})
	}
	return out
}

func toApiFffCloseItems(list []*FffCloseItem) []*dashboard_api.FffCloseItem {
	out := make([]*dashboard_api.FffCloseItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.FffCloseItem{
			Dt:              v.Dt,
			FilterName:      v.FilterName,
			Version:         v.Version,
			Reason:          v.Reason,
			AnonymousId:     v.AnonymousId,
			CreateAt:        v.CreateAt,
			SwVersion:       v.SwVersion,
			TimestampUtc:    v.TimestampUtc,
			ProjectName:     v.ProjectName,
			CarType:         v.CarType,
			VehicleSource:   v.VehicleSource,
			FdiProjectName:  v.FdiProjectName,
			ProjectCarType:  v.ProjectCarType,
			VehicleSourceCn: v.VehicleSourceCn,
		})
	}
	return out
}

func toApiFdrTriggerItems(list []*FdrTriggerItem) []*dashboard_api.FdrTriggerItem {
	out := make([]*dashboard_api.FdrTriggerItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.FdrTriggerItem{
			Dt:                v.Dt,
			Uuid:              v.Uuid,
			EventName:         v.EventName,
			AnonymousId:       v.AnonymousId,
			TimestampUtc:      v.TimestampUtc,
			CreateAt:          v.CreateAt,
			SwVersion:         v.SwVersion,
			Dse:               v.Dse,
			TdMb:              v.TdMb,
			TmMb:              v.TmMb,
			TriggerTimestamp:  v.TriggerTimestamp,
			BeginTimestampUts: v.BeginTimestampUts,
			EndTimestampUts:   v.EndTimestampUts,
			DumpTimestamp:     v.DumpTimestamp,
			Status:            v.Status,
			Detail:            v.Detail,
			TimeCostMs:        v.TimeCostMs,
			RecordType:        v.RecordType,
			ProjectName:       v.ProjectName,
			CarType:           v.CarType,
			VehicleSource:     v.VehicleSource,
			FdiProjectName:    v.FdiProjectName,
			ProjectCarType:    v.ProjectCarType,
			VehicleSourceCn:   v.VehicleSourceCn,
		})
	}
	return out
}

func toApiFclTriggerItems(list []*FclTriggerItem) []*dashboard_api.FclTriggerItem {
	out := make([]*dashboard_api.FclTriggerItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.FclTriggerItem{
			Dt:              v.Dt,
			Uuid:            v.Uuid,
			EventName:       v.EventName,
			AnonymousId:     v.AnonymousId,
			TimestampUtc:    v.TimestampUtc,
			CreateAt:        v.CreateAt,
			Status:          v.Status,
			Detail:          v.Detail,
			CompletePercent: v.CompletePercent,
			LocalFile:       v.LocalFile,
			UploadFailTimes: v.UploadFailTimes,
			PrefixStitch:    v.PrefixStitch,
			TriggerSource:   v.TriggerSource,
			SwVersion:       v.SwVersion,
			ProjectName:     v.ProjectName,
			CarType:         v.CarType,
			VehicleSource:   v.VehicleSource,
			FdiProjectName:  v.FdiProjectName,
			ProjectCarType:  v.ProjectCarType,
			VehicleSourceCn: v.VehicleSourceCn,
		})
	}
	return out
}

func toApiUuidDetailItems(list []*UuidDetailItem) []*dashboard_api.UuidDetailItem {
	out := make([]*dashboard_api.UuidDetailItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.UuidDetailItem{
			Dt:                v.Dt,
			AnonymousId:       v.AnonymousId,
			EventName:         v.EventName,
			Uuid:              v.Uuid,
			CreateAt:          v.CreateAt,
			FilterName:        v.FilterName,
			FffSwVersion:      v.FffSwVersion,
			FdrSwVersion:      v.FdrSwVersion,
			FclSwVersion:      v.FclSwVersion,
			TriggerType:       v.TriggerType,
			CollectType:       v.CollectType,
			FffUpdatedAt:      v.FffUpdatedAt,
			FdrUpdatedAt:      v.FdrUpdatedAt,
			FclUpdatedAt:      v.FclUpdatedAt,
			FffStatus:         v.FffStatus,
			FdrStatus:         v.FdrStatus,
			FclStatus:         v.FclStatus,
			FffDetail:         v.FffDetail,
			FdrDetail:         v.FdrDetail,
			FclDetail:         v.FclDetail,
			BeginTimestampUts: v.BeginTimestampUts,
			DumpTimestamp:     v.DumpTimestamp,
			EndTimestampUts:   v.EndTimestampUts,
			Md5:               v.Md5,
			BagName:           v.BagName,
			CompletePercent:   v.CompletePercent,
			ProjectName:       v.ProjectName,
			CarType:           v.CarType,
			VehicleSource:     v.VehicleSource,
			TimestampUtc:      v.TimestampUtc,
			FdiProjectName:    v.FdiProjectName,
			ProjectCarType:    v.ProjectCarType,
			VehicleSourceCn:   v.VehicleSourceCn,
			Dse:               v.Dse,
		})
	}
	return out
}

func toApiCloseReasonItems(list []*CloseReasonItem) []*dashboard_api.CloseReasonItem {
	out := make([]*dashboard_api.CloseReasonItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.CloseReasonItem{Name: v.Name, Value: v.Value})
	}
	return out
}

func toApiFunnelStat(s *FunnelStat) *dashboard_api.FunnelStat {
	if s == nil {
		return nil
	}
	return &dashboard_api.FunnelStat{
		FffTotal:   s.FffTotal,
		FffAllow:   s.FffAllow,
		FdrSuccess: s.FdrSuccess,
		FdrFail:    s.FdrFail,
		FclSuccess: s.FclSuccess,
		FclFail:    s.FclFail,
		CfdiRate:   s.CfdiRate,
	}
}

func toApiFunnelFailReasons(list []*FunnelFailReason) []*dashboard_api.FunnelFailReason {
	out := make([]*dashboard_api.FunnelFailReason, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.FunnelFailReason{Name: v.Name, Count: v.Count})
	}
	return out
}

func toApiStageTrendSeries(list []*StageTrendSeries) []*dashboard_api.StageTrendSeries {
	out := make([]*dashboard_api.StageTrendSeries, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.StageTrendSeries{Name: v.Name, Data: v.Data})
	}
	return out
}
