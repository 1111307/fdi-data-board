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
