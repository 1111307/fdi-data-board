package biz

import (
	"context"
	"strings"

	dashboard_api "fdi_data_board/api/dashboard"
)

// DoDashboardRepo DO Dashboard 数据仓储接口
type DoDashboardRepo interface {
	GetOverview(ctx context.Context, param *DoOverviewParam) ([]*dashboard_api.DoOverviewItem, error)
	GetTrend(ctx context.Context, param *DoTrendParam) (*DoTrendData, error)
	GetFailReason(ctx context.Context, param *DoFailReasonParam) ([]*dashboard_api.DoFailReasonItem, error)
	GetCoolTop(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoCoolTopItem, error)
}

// DoCommonParam 通用查询参数（无特殊字段的 Top 类接口复用）
type DoCommonParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// DoFailReasonParam 失败原因分析查询参数
type DoFailReasonParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// DoOverviewParam 事件横向对比查询参数
type DoOverviewParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// DoTrendParam 趋势查询参数
type DoTrendParam struct {
	FilterName  string
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// DoTrendData 趋势聚合结果
type DoTrendData struct {
	Dates         []string
	SuccessCounts []int64
	SuccessRates  []float64
}

// DoDashboardUseCase DO Dashboard 业务用例
type DoDashboardUseCase struct {
	repo DoDashboardRepo
}

func NewDoDashboardUseCase(repo DoDashboardRepo) *DoDashboardUseCase {
	return &DoDashboardUseCase{repo: repo}
}

func (uc *DoDashboardUseCase) GetOverview(ctx context.Context, req *dashboard_api.DoOverviewRequest) (*dashboard_api.DoOverviewResponse, error) {
	param := &DoOverviewParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	list, err := uc.repo.GetOverview(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoOverviewResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         list,
	}, nil
}

func (uc *DoDashboardUseCase) GetTrend(ctx context.Context, req *dashboard_api.DoTrendRequest) (*dashboard_api.DoTrendResponse, error) {
	param := &DoTrendParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	data, err := uc.repo.GetTrend(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoTrendResponse{
		BaseResponse:  dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:         data.Dates,
		SuccessCounts: data.SuccessCounts,
		SuccessRates:  data.SuccessRates,
	}, nil
}

func (uc *DoDashboardUseCase) GetCoolTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoCoolTopResponse, error) {
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	list, err := uc.repo.GetCoolTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoCoolTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         list,
	}, nil
}

func (uc *DoDashboardUseCase) GetFailReason(ctx context.Context, req *dashboard_api.DoFailReasonRequest) (*dashboard_api.DoFailReasonResponse, error) {
	param := &DoFailReasonParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	list, err := uc.repo.GetFailReason(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFailReasonResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         list,
	}, nil
}

func splitNames(raw string) []string {
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
