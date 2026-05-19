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
	GetSwVersion(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoSwVersionItem, error)
	GetTriggerRank(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoCoolTopItem, error)
	GetProjectCar(ctx context.Context, param *DoCommonParam) (*dashboard_api.DoProjectCarResponse, error)
	GetMemTop(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoEventTopItem, error)
	GetDiskTop(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoEventTopItem, error)
	GetCloseTop(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoCoolTopItem, error)
	GetQuotaTop(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoEventTopItem, error)
	GetProjectEvent(ctx context.Context, param *DoCommonParam) ([]*dashboard_api.DoProjectEventItem, error)
	GetNetSpeed(ctx context.Context, param *DoCommonParam) (*dashboard_api.DoNetSpeedResponse, error)
	GetFclBw(ctx context.Context, param *DoCommonParam) (*dashboard_api.DoFclBwResponse, error)
	GetTopVehicles(ctx context.Context, param *DoVehicleParam) ([]*dashboard_api.DoVehicleItem, error)
	GetAnomalyVehicles(ctx context.Context, param *DoAnomalyParam) ([]*dashboard_api.DoVehicleItem, error)
	GetActiveTrend(ctx context.Context, param *DoVehicleParam) (*dashboard_api.DoActiveTrendResponse, error)
}

// DoVehicleParam 车辆维度分析通用查询参数
type DoVehicleParam struct {
	EventNames  []string
	ProjectName string
	CarTypes    []string
	StartDt     string
	EndDt       string
}

// DoAnomalyParam 异常车辆查询参数
type DoAnomalyParam struct {
	DoVehicleParam
	MaxRate int
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

func (uc *DoDashboardUseCase) GetTriggerRank(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoTriggerRankResponse, error) {
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	list, err := uc.repo.GetTriggerRank(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoTriggerRankResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         list,
	}, nil
}

func (uc *DoDashboardUseCase) GetSwVersion(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoSwVersionResponse, error) {
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	list, err := uc.repo.GetSwVersion(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoSwVersionResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         list,
	}, nil
}

func (uc *DoDashboardUseCase) GetProjectCar(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoProjectCarResponse, error) {
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitNames(req.CarTypes),
		StartDt:     req.StartDt,
		EndDt:       req.EndDt,
	}
	return uc.repo.GetProjectCar(ctx, param)
}

func (uc *DoDashboardUseCase) GetMemTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoMemTopResponse, error) {
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	list, err := uc.repo.GetMemTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoMemTopResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetDiskTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoDiskTopResponse, error) {
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	list, err := uc.repo.GetDiskTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoDiskTopResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetCloseTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoCloseTopResponse, error) {
	param := &DoCommonParam{
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	list, err := uc.repo.GetCloseTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoCloseTopResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetQuotaTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoQuotaTopResponse, error) {
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	list, err := uc.repo.GetQuotaTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoQuotaTopResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetProjectEvent(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoProjectEventResponse, error) {
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	list, err := uc.repo.GetProjectEvent(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoProjectEventResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetNetSpeed(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoNetSpeedResponse, error) {
	param := &DoCommonParam{
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	return uc.repo.GetNetSpeed(ctx, param)
}

func (uc *DoDashboardUseCase) GetFclBw(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoFclBwResponse, error) {
	param := &DoCommonParam{
		ProjectName: req.ProjectName, CarTypes: splitNames(req.CarTypes),
		StartDt: req.StartDt, EndDt: req.EndDt,
	}
	return uc.repo.GetFclBw(ctx, param)
}

func (uc *DoDashboardUseCase) GetTopVehicles(ctx context.Context, req *dashboard_api.DoTopVehicleRequest) (*dashboard_api.DoTopVehicleResponse, error) {
	param := &DoVehicleParam{
		EventNames: splitNames(req.EventNames), ProjectName: req.ProjectName,
		CarTypes: splitNames(req.CarTypes), StartDt: req.StartDt, EndDt: req.EndDt,
	}
	list, err := uc.repo.GetTopVehicles(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoTopVehicleResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetAnomalyVehicles(ctx context.Context, req *dashboard_api.DoAnomalyRequest) (*dashboard_api.DoAnomalyResponse, error) {
	maxRate := req.MaxRate
	if maxRate == 0 {
		maxRate = 80
	}
	param := &DoAnomalyParam{
		DoVehicleParam: DoVehicleParam{
			EventNames: splitNames(req.EventNames), ProjectName: req.ProjectName,
			CarTypes: splitNames(req.CarTypes), StartDt: req.StartDt, EndDt: req.EndDt,
		},
		MaxRate: maxRate,
	}
	list, err := uc.repo.GetAnomalyVehicles(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoAnomalyResponse{BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"}, List: list}, nil
}

func (uc *DoDashboardUseCase) GetActiveTrend(ctx context.Context, req *dashboard_api.DoTopVehicleRequest) (*dashboard_api.DoActiveTrendResponse, error) {
	param := &DoVehicleParam{
		EventNames: splitNames(req.EventNames), ProjectName: req.ProjectName,
		CarTypes: splitNames(req.CarTypes), StartDt: req.StartDt, EndDt: req.EndDt,
	}
	return uc.repo.GetActiveTrend(ctx, param)
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
