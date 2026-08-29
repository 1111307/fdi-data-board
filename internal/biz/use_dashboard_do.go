package biz

import (
	"context"

	dashboard_api "fdi_data_board/api/dashboard"
)

// DoDashboardRepo DO Dashboard 数据仓储接口
type DoDashboardRepo interface {
	GetOverview(ctx context.Context, param *DoOverviewParam) ([]*DoOverviewItem, error)
	GetTrend(ctx context.Context, param *DoTrendParam) (*DoTrendData, error)
	GetFailReason(ctx context.Context, param *DoFailReasonParam) ([]*DoFailReasonItem, error)
	GetCoolTop(ctx context.Context, param *DoCommonParam) ([]*DoCoolTopItem, error)
	GetSwVersion(ctx context.Context, param *DoCommonParam) ([]*DoSwVersionItem, error)
	GetTriggerRank(ctx context.Context, param *DoCommonParam) ([]*DoCoolTopItem, error)
	GetProjectCar(ctx context.Context, param *DoCommonParam) (*DoProjectCarData, error)
	GetMemTop(ctx context.Context, param *DoCommonParam) ([]*DoEventTopItem, error)
	GetDiskTop(ctx context.Context, param *DoCommonParam) ([]*DoEventTopItem, error)
	GetCloseTop(ctx context.Context, param *DoCommonParam) ([]*DoCoolTopItem, error)
	GetQuotaTop(ctx context.Context, param *DoCommonParam) ([]*DoEventTopItem, error)
	GetProjectEvent(ctx context.Context, param *DoCommonParam) ([]*DoProjectEventItem, error)
	GetNetSpeed(ctx context.Context, param *DoCommonParam) (*DoNetSpeedData, error)
	GetFclBw(ctx context.Context, param *DoCommonParam) (*DoFclBwData, error)
	GetTopVehicles(ctx context.Context, param *DoVehicleParam) ([]*DoVehicleItem, error)
	GetAnomalyVehicles(ctx context.Context, param *DoAnomalyParam) ([]*DoVehicleItem, error)
	GetActiveTrend(ctx context.Context, param *DoVehicleParam) (*DoActiveTrendData, error)
	GetDoFunnel(ctx context.Context, param *DoCommonParam) (*FunnelData, error)
	GetFdrQuality(ctx context.Context, param *DoCommonParam) (*DoFdrQualityData, error)
	GetFclQuality(ctx context.Context, param *DoCommonParam) (*DoFclQualityData, error)
	GetFdrFragment(ctx context.Context, param *DoCommonParam) (*DoFdrFragmentData, error)
	GetFdrBandwidthTop(ctx context.Context, param *DoCommonParam) (*DoFdrBandwidthTopData, error)
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
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoOverviewParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetOverview(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoOverviewResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoOverviewItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetTrend(ctx context.Context, req *dashboard_api.DoTrendRequest) (*dashboard_api.DoTrendResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoTrendParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
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
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetCoolTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoCoolTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoCoolTopItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetTriggerRank(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoTriggerRankResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetTriggerRank(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoTriggerRankResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoCoolTopItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetSwVersion(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoSwVersionResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetSwVersion(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoSwVersionResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoSwVersionItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetProjectCar(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoProjectCarResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetProjectCar(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoProjectCarResponse{
		BaseResponse:  dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Projects:      data.Projects,
		CarTypes:      data.CarTypes,
		Matrix:        data.Matrix,
		ProjectTotals: data.ProjectTotals,
	}, nil
}

func (uc *DoDashboardUseCase) GetMemTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoMemTopResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitEventNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	list, err := uc.repo.GetMemTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoMemTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoEventTopItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetDiskTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoDiskTopResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitEventNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	list, err := uc.repo.GetDiskTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoDiskTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoEventTopItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetCloseTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoCloseTopResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	list, err := uc.repo.GetCloseTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoCloseTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoCoolTopItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetQuotaTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoQuotaTopResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitEventNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	list, err := uc.repo.GetQuotaTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoQuotaTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoEventTopItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetProjectEvent(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoProjectEventResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName: req.FilterName, EventNames: splitEventNames(req.EventNames),
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	list, err := uc.repo.GetProjectEvent(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoProjectEventResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoProjectEventItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetNetSpeed(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoNetSpeedResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	data, err := uc.repo.GetNetSpeed(ctx, param)
	if err != nil {
		return nil, err
	}
	series := make([]*dashboard_api.DoNetSpeedSeries, 0, len(data.Series))
	for _, s := range data.Series {
		series = append(series, &dashboard_api.DoNetSpeedSeries{CarType: s.CarType, Data: s.Data})
	}
	return &dashboard_api.DoNetSpeedResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:        data.Dates,
		Series:       series,
	}, nil
}

func (uc *DoDashboardUseCase) GetFclBw(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoFclBwResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		ProjectName: req.ProjectName, CarTypes: splitEventNames(req.CarTypes),
		StartDt: startDt, EndDt: endDt,
	}
	data, err := uc.repo.GetFclBw(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFclBwResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:        data.Dates,
		Values:       data.Values,
	}, nil
}

func (uc *DoDashboardUseCase) GetTopVehicles(ctx context.Context, req *dashboard_api.DoTopVehicleRequest) (*dashboard_api.DoTopVehicleResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoVehicleParam{
		EventNames: splitEventNames(req.EventNames), ProjectName: req.ProjectName,
		CarTypes: splitEventNames(req.CarTypes), StartDt: startDt, EndDt: endDt,
	}
	list, err := uc.repo.GetTopVehicles(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoTopVehicleResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoVehicleItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetAnomalyVehicles(ctx context.Context, req *dashboard_api.DoAnomalyRequest) (*dashboard_api.DoAnomalyResponse, error) {
	maxRate := req.MaxRate
	if maxRate == 0 {
		maxRate = 80
	}
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoAnomalyParam{
		DoVehicleParam: DoVehicleParam{
			EventNames: splitEventNames(req.EventNames), ProjectName: req.ProjectName,
			CarTypes: splitEventNames(req.CarTypes), StartDt: startDt, EndDt: endDt,
		},
		MaxRate: maxRate,
	}
	list, err := uc.repo.GetAnomalyVehicles(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoAnomalyResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoVehicleItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetActiveTrend(ctx context.Context, req *dashboard_api.DoTopVehicleRequest) (*dashboard_api.DoActiveTrendResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoVehicleParam{
		EventNames: splitEventNames(req.EventNames), ProjectName: req.ProjectName,
		CarTypes: splitEventNames(req.CarTypes), StartDt: startDt, EndDt: endDt,
	}
	data, err := uc.repo.GetActiveTrend(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoActiveTrendResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		Dates:        data.Dates,
		Counts:       data.Counts,
	}, nil
}

func (uc *DoDashboardUseCase) GetFailReason(ctx context.Context, req *dashboard_api.DoFailReasonRequest) (*dashboard_api.DoFailReasonResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoFailReasonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	list, err := uc.repo.GetFailReason(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFailReasonResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoFailReasonItems(list),
	}, nil
}

func (uc *DoDashboardUseCase) GetDoFunnel(ctx context.Context, req *dashboard_api.DoFunnelRequest) (*dashboard_api.FunnelResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetDoFunnel(ctx, param)
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

func (uc *DoDashboardUseCase) GetFdrQuality(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoFdrQualityResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFdrQuality(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFdrQualityResponse{
		BaseResponse:  dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		TdMbP95:       data.TdMbP95,
		TmMbP95:       data.TmMbP95,
		TimeCostMsP95: data.TimeCostMsP95,
		FdrTotal:      data.FdrTotal,
		FdrSuccess:    data.FdrSuccess,
	}, nil
}

func (uc *DoDashboardUseCase) GetFclQuality(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoFclQualityResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFclQuality(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFclQualityResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		BagSizeP95:   data.BagSizeP95,
		BagSizeAvg:   data.BagSizeAvg,
		BagSizeMax:   data.BagSizeMax,
		UploadTotal:  data.UploadTotal,
	}, nil
}

func (uc *DoDashboardUseCase) GetFdrFragment(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoFdrFragmentResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFdrFragment(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFdrFragmentResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		FragmentP95:  data.FragmentP95,
		FragmentAvg:  data.FragmentAvg,
		FragmentMax:  data.FragmentMax,
	}, nil
}

func (uc *DoDashboardUseCase) GetFdrBandwidthTop(ctx context.Context, req *dashboard_api.DoCoolTopRequest) (*dashboard_api.DoFdrBandwidthTopResponse, error) {
	startDt, endDt, err := normalizeDateRange(req.StartDt, req.EndDt)
	if err != nil {
		return nil, err
	}
	param := &DoCommonParam{
		FilterName:  req.FilterName,
		EventNames:  splitEventNames(req.EventNames),
		ProjectName: req.ProjectName,
		CarTypes:    splitEventNames(req.CarTypes),
		StartDt:     startDt,
		EndDt:       endDt,
	}
	data, err := uc.repo.GetFdrBandwidthTop(ctx, param)
	if err != nil {
		return nil, err
	}
	return &dashboard_api.DoFdrBandwidthTopResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: 0, Message: "OK"},
		List:         toApiDoBandwidthTopItems(data.List),
	}, nil
}

// ---------- biz domain → api DTO 映射函数 ----------

func toApiDoBandwidthTopItems(list []*DoBandwidthTopItem) []*dashboard_api.DoBandwidthTopItem {
	out := make([]*dashboard_api.DoBandwidthTopItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoBandwidthTopItem{
			BandwidthField: v.BandwidthField,
			BwSum:          v.BwSum,
			BwAvg:          v.BwAvg,
			BwP95:          v.BwP95,
			BwMax:          v.BwMax,
			SampleCount:    v.SampleCount,
		})
	}
	return out
}

func toApiDoOverviewItems(list []*DoOverviewItem) []*dashboard_api.DoOverviewItem {
	out := make([]*dashboard_api.DoOverviewItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoOverviewItem{
			EventName:    v.EventName,
			VehicleCount: v.VehicleCount,
			TriggerCount: v.TriggerCount,
			CfdiRate:     v.CfdiRate,
			FffCount:     v.FffCount,
			FffRate:      v.FffRate,
			FdrCount:     v.FdrCount,
			FdrRate:      v.FdrRate,
			FclCount:     v.FclCount,
			FclRate:      v.FclRate,
		})
	}
	return out
}

func toApiDoCoolTopItems(list []*DoCoolTopItem) []*dashboard_api.DoCoolTopItem {
	out := make([]*dashboard_api.DoCoolTopItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoCoolTopItem{FilterName: v.FilterName, Count: v.Count})
	}
	return out
}

func toApiDoSwVersionItems(list []*DoSwVersionItem) []*dashboard_api.DoSwVersionItem {
	out := make([]*dashboard_api.DoSwVersionItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoSwVersionItem{SwVersion: v.SwVersion, Count: v.Count})
	}
	return out
}

func toApiDoEventTopItems(list []*DoEventTopItem) []*dashboard_api.DoEventTopItem {
	out := make([]*dashboard_api.DoEventTopItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoEventTopItem{EventName: v.EventName, Count: v.Count})
	}
	return out
}

func toApiDoProjectEventItems(list []*DoProjectEventItem) []*dashboard_api.DoProjectEventItem {
	out := make([]*dashboard_api.DoProjectEventItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoProjectEventItem{ProjectName: v.ProjectName, EventCount: v.EventCount})
	}
	return out
}

func toApiDoVehicleItems(list []*DoVehicleItem) []*dashboard_api.DoVehicleItem {
	out := make([]*dashboard_api.DoVehicleItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoVehicleItem{
			AnonymousId:  v.AnonymousId,
			CarType:      v.CarType,
			ProjectName:  v.ProjectName,
			TriggerCount: v.TriggerCount,
			SuccessCount: v.SuccessCount,
			CfdiRate:     v.CfdiRate,
			MainReason:   v.MainReason,
		})
	}
	return out
}

func toApiDoFailReasonItems(list []*DoFailReasonItem) []*dashboard_api.DoFailReasonItem {
	out := make([]*dashboard_api.DoFailReasonItem, 0, len(list))
	for _, v := range list {
		out = append(out, &dashboard_api.DoFailReasonItem{Name: v.Name, Value: v.Value})
	}
	return out
}
