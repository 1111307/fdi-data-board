package biz

import (
	"context"
	"strings"

	dashboard_api "fdi_data_board/api/dashboard"
)

// DoDashboardRepo DO Dashboard 数据仓储接口
type DoDashboardRepo interface {
	GetTrend(ctx context.Context, param *DoTrendParam) (*DoTrendData, error)
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
