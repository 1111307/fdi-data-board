package biz

import (
	"context"

	dashboard_api "fdi_data_board/api/dashboard"
)

const (
	defaultPageSize = 50
	maxPageSize     = 500
)

// FoDashboardRepo FO Dashboard 数据仓储接口
type FoDashboardRepo interface {
	ListFffRunning(ctx context.Context, param *FffRunningParam) ([]*dashboard_api.FffRunningItem, int64, error)
	ListFffTrigger(ctx context.Context, param *FffTriggerParam) ([]*dashboard_api.FffTriggerItem, int64, error)
	ListFffClose(ctx context.Context, param *FffCloseParam) ([]*dashboard_api.FffCloseItem, int64, error)
	ListFdrTrigger(ctx context.Context, param *FdrTriggerParam) ([]*dashboard_api.FdrTriggerItem, int64, error)
	ListFclTrigger(ctx context.Context, param *FclTriggerParam) ([]*dashboard_api.FclTriggerItem, int64, error)
	GetDimensions(ctx context.Context) (*FoDimensions, error)
}

// FclTriggerParam FCL 上传明细查询参数
type FclTriggerParam struct {
	FilterName  string
	EventName   string
	ProjectName string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FdrTriggerParam FDR 落盘明细查询参数
type FdrTriggerParam struct {
	FilterName  string
	EventName   string
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
	EventName   string
	ProjectName string
	StartDt     string
	EndDt       string
	Page        int
	PageSize    int
}

// FoDimensions 维度枚举数据
type FoDimensions struct {
	FilterNames  []string
	EventNames   []string
	ProjectNames []string
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
		EventName:   req.EventName,
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
		EventName:   req.EventName,
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
		EventName:   req.EventName,
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
