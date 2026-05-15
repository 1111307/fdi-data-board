package biz

import (
	"context"
	"strings"

	dashboard_api "fdi_data_board/api/dashboard"
)

const (
	defaultPageSize = 50
	maxPageSize     = 500
)

// FoDashboardRepo FO Dashboard 数据仓储接口
type FoDashboardRepo interface {
	GetFunnel(ctx context.Context, param *FunnelParam) (*FunnelData, error)
	ListFffRunning(ctx context.Context, param *FffRunningParam) ([]*dashboard_api.FffRunningItem, int64, error)
	ListFffTrigger(ctx context.Context, param *FffTriggerParam) ([]*dashboard_api.FffTriggerItem, int64, error)
	ListFffClose(ctx context.Context, param *FffCloseParam) ([]*dashboard_api.FffCloseItem, int64, error)
	ListFdrTrigger(ctx context.Context, param *FdrTriggerParam) ([]*dashboard_api.FdrTriggerItem, int64, error)
	ListFclTrigger(ctx context.Context, param *FclTriggerParam) ([]*dashboard_api.FclTriggerItem, int64, error)
	ListUuidDetail(ctx context.Context, param *UuidDetailParam) ([]*dashboard_api.UuidDetailItem, int64, error)
	GetCloseReason(ctx context.Context, param *CloseReasonParam) ([]*dashboard_api.CloseReasonItem, error)
	GetStageTrend(ctx context.Context, param *StageTrendParam) (*StageTrendData, error)
	GetDimensions(ctx context.Context) (*FoDimensions, error)
}

// FunnelParam 数采全链路分析查询参数
type FunnelParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	StartDt     string
	EndDt       string
}

// FunnelData biz 层聚合结果
type FunnelData struct {
	Stat    *dashboard_api.FunnelStat
	FffFail []*dashboard_api.FunnelFailReason
	FdrFail []*dashboard_api.FunnelFailReason
	FclFail []*dashboard_api.FunnelFailReason
}

// StageTrendParam 三阶段触发趋势查询参数
type StageTrendParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	StartDt     string
	EndDt       string
}

// StageTrendData biz 层聚合结果
type StageTrendData struct {
	Dates []string
	Fff   []*dashboard_api.StageTrendSeries
	Fdr   []*dashboard_api.StageTrendSeries
	Fcl   []*dashboard_api.StageTrendSeries
}

// CloseReasonParam 算子关闭原因分布查询参数
type CloseReasonParam struct {
	FilterName  string
	ProjectName string
	StartDt     string
	EndDt       string
}

// UuidDetailParam 全链路明细查询参数
type UuidDetailParam struct {
	FilterName  string
	EventNames  []string // 多选
	ProjectName string
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
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FffCloseParam 筛选器关闭明细查询参数
type FffCloseParam struct {
	FilterName  string
	ProjectName string
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
	ProjectName string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FoDashboardUseCase FO Dashboard 业务用例
type FoDashboardUseCase struct {
	repo FoDashboardRepo
}

func NewFoDashboardUseCase(repo FoDashboardRepo) *FoDashboardUseCase {
	return &FoDashboardUseCase{repo: repo}
}

func (uc *FoDashboardUseCase) ListFffTrigger(ctx context.Context, req *dashboard_api.FffTriggerRequest) (*dashboard_api.FffTriggerResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	param := &FffTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
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
		List:         list,
	}, nil
}

func (uc *FoDashboardUseCase) ListFffClose(ctx context.Context, req *dashboard_api.FffCloseRequest) (*dashboard_api.FffCloseResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	param := &FffCloseParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
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
		List:         list,
	}, nil
}

func (uc *FoDashboardUseCase) ListFdrTrigger(ctx context.Context, req *dashboard_api.FdrTriggerRequest) (*dashboard_api.FdrTriggerResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	param := &FdrTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
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
		List:         list,
	}, nil
}

func (uc *FoDashboardUseCase) ListFclTrigger(ctx context.Context, req *dashboard_api.FclTriggerRequest) (*dashboard_api.FclTriggerResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	param := &FclTriggerParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
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
		List:         list,
	}, nil
}

func (uc *FoDashboardUseCase) ListUuidDetail(ctx context.Context, req *dashboard_api.UuidDetailRequest) (*dashboard_api.UuidDetailResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	var eventNames []string
	if req.EventNames != "" {
		for _, e := range strings.Split(req.EventNames, ",") {
			if e = strings.TrimSpace(e); e != "" {
				eventNames = append(eventNames, e)
			}
		}
	}

	param := &UuidDetailParam{
		FilterName:  req.FilterName,
		EventNames:  eventNames,
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
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
		List:         list,
	}, nil
}

func (uc *FoDashboardUseCase) GetCloseReason(ctx context.Context, req *dashboard_api.CloseReasonRequest) (*dashboard_api.CloseReasonResponse, error) {
	param := &CloseReasonParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	list, err := uc.repo.GetCloseReason(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.CloseReasonResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         list,
	}, nil
}

func (uc *FoDashboardUseCase) GetFunnel(ctx context.Context, req *dashboard_api.FunnelRequest) (*dashboard_api.FunnelResponse, error) {
	param := &FunnelParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	data, err := uc.repo.GetFunnel(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.FunnelResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Stat:         data.Stat,
		FffFail:      data.FffFail,
		FdrFail:      data.FdrFail,
		FclFail:      data.FclFail,
	}, nil
}

func (uc *FoDashboardUseCase) GetStageTrend(ctx context.Context, req *dashboard_api.StageTrendRequest) (*dashboard_api.StageTrendResponse, error) {
	param := &StageTrendParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	data, err := uc.repo.GetStageTrend(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.StageTrendResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:        data.Dates,
		Fff:          data.Fff,
		Fdr:          data.Fdr,
		Fcl:          data.Fcl,
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
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	param := &FffRunningParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName,
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
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
		List:         list,
	}, nil
}

// splitEventNames 将逗号分隔的事件名字符串拆分为切片，空字符串返回 nil
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
