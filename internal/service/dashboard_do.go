package service

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

type DoDashboardService struct {
	uc *biz.DoDashboardUseCase
}

func NewDoDashboardService(uc *biz.DoDashboardUseCase) *DoDashboardService {
	return &DoDashboardService{uc: uc}
}

// GetOverview godoc
//
//	@Summary		DO 事件横向对比
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoOverviewResponse
//	@Router			/dashboard/v1/do/overview [GET]
func (s *DoDashboardService) GetOverview(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoOverviewResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoOverviewRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetOverview(ctx, &req)
	if err != nil {
		log.Errorf("DoGetOverview error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}

// GetProjectCar godoc
//
//	@Summary		DO 项目×车型分布
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoProjectCarResponse
//	@Router			/dashboard/v1/do/project_car [GET]
func (s *DoDashboardService) GetProjectCar(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoProjectCarResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetProjectCar(ctx, &req)
	if err != nil {
		log.Errorf("DoGetProjectCar error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}

// GetTriggerRank godoc
//
//	@Summary		DO 触发频次排行 Top10
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoTriggerRankResponse
//	@Router			/dashboard/v1/do/trigger_rank [GET]
func (s *DoDashboardService) GetTriggerRank(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoTriggerRankResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetTriggerRank(ctx, &req)
	if err != nil {
		log.Errorf("DoGetTriggerRank error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}

// GetSwVersion godoc
//
//	@Summary		DO 触发记录软件版本分布
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoSwVersionResponse
//	@Router			/dashboard/v1/do/sw_version [GET]
func (s *DoDashboardService) GetSwVersion(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoSwVersionResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetSwVersion(ctx, &req)
	if err != nil {
		log.Errorf("DoGetSwVersion error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}

// GetCoolTop godoc
//
//	@Summary		DO 冷却 Top20 筛选器
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoCoolTopResponse
//	@Router			/dashboard/v1/do/cool_top [GET]
func (s *DoDashboardService) GetCoolTop(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoCoolTopResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetCoolTop(ctx, &req)
	if err != nil {
		log.Errorf("DoGetCoolTop error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}

// GetFailReason godoc
//
//	@Summary		DO 失败原因分析（三阶段归因）
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoFailReasonResponse
//	@Router			/dashboard/v1/do/fail_reason [GET]
func (s *DoDashboardService) GetFailReason(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoFailReasonResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoFailReasonRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetFailReason(ctx, &req)
	if err != nil {
		log.Errorf("DoGetFailReason error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}

// GetTrend godoc
//
//	@Summary		DO 数据总览趋势（按天成功次数与成功率）
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			filter_name		query	string	false	"筛选器名称"
//	@Param			event_names		query	string	false	"事件名，多选逗号分隔"
//	@Param			project_name	query	string	false	"项目名称"
//	@Param			car_types		query	string	false	"车型，多选逗号分隔"
//	@Param			start_dt		query	string	false	"开始日期，不传默认近7天"
//	@Param			end_dt			query	string	false	"结束日期"
//	@Success		200				{object}	dashboard_api.DoTrendResponse
//	@Router			/dashboard/v1/do/trend [GET]
// GetTopVehicles godoc
//
//	@Summary		DO Top20活跃车辆
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoTopVehicleResponse
//	@Router			/dashboard/v1/do/top_vehicles [GET]
func (s *DoDashboardService) GetTopVehicles(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoTopVehicleResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()
	var req dashboard_api.DoTopVehicleRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code()); resp.Message = err.Error(); return resp, nil
	}
	result, err := s.uc.GetTopVehicles(ctx, &req)
	if err != nil {
		log.Errorf("DoGetTopVehicles error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code()); resp.Message = err.Error(); return resp, nil
	}
	return result, nil
}

// GetAnomalyVehicles godoc
//
//	@Summary		DO 异常车辆
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoAnomalyResponse
//	@Router			/dashboard/v1/do/anomaly_vehicles [GET]
func (s *DoDashboardService) GetAnomalyVehicles(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoAnomalyResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()
	var req dashboard_api.DoAnomalyRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code()); resp.Message = err.Error(); return resp, nil
	}
	result, err := s.uc.GetAnomalyVehicles(ctx, &req)
	if err != nil {
		log.Errorf("DoGetAnomalyVehicles error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code()); resp.Message = err.Error(); return resp, nil
	}
	return result, nil
}

// GetActiveTrend godoc
//
//	@Summary		DO 活跃车辆趋势
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoActiveTrendResponse
//	@Router			/dashboard/v1/do/active_trend [GET]
func (s *DoDashboardService) GetActiveTrend(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoActiveTrendResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()
	var req dashboard_api.DoTopVehicleRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code()); resp.Message = err.Error(); return resp, nil
	}
	result, err := s.uc.GetActiveTrend(ctx, &req)
	if err != nil {
		log.Errorf("DoGetActiveTrend error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code()); resp.Message = err.Error(); return resp, nil
	}
	return result, nil
}

// GetNetSpeed godoc
//
//	@Summary		DO 各车型平均上传带宽（按天）
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoNetSpeedResponse
//	@Router			/dashboard/v1/do/net_speed [GET]
func (s *DoDashboardService) GetNetSpeed(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoNetSpeedResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetNetSpeed(ctx, &req)
	if err != nil {
		log.Errorf("DoGetNetSpeed error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

// GetFclBw godoc
//
//	@Summary		DO FCL 整体平均上传带宽（按天）
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoFclBwResponse
//	@Router			/dashboard/v1/do/fcl_bw [GET]
func (s *DoDashboardService) GetFclBw(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoFclBwResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetFclBw(ctx, &req)
	if err != nil {
		log.Errorf("DoGetFclBw error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

// GetQuotaTop godoc
//
//	@Summary		DO FCL Quota 超限 Top20
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoQuotaTopResponse
//	@Router			/dashboard/v1/do/quota_top [GET]
func (s *DoDashboardService) GetQuotaTop(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoQuotaTopResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetQuotaTop(ctx, &req)
	if err != nil {
		log.Errorf("DoGetQuotaTop error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

// GetProjectEvent godoc
//
//	@Summary		DO 项目触发回流事件总数
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoProjectEventResponse
//	@Router			/dashboard/v1/do/project_event [GET]
func (s *DoDashboardService) GetProjectEvent(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoProjectEventResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetProjectEvent(ctx, &req)
	if err != nil {
		log.Errorf("DoGetProjectEvent error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

// GetMemTop godoc
//
//	@Summary		DO FDR 内存不足 Top20
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoMemTopResponse
//	@Router			/dashboard/v1/do/mem_top [GET]
func (s *DoDashboardService) GetMemTop(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoMemTopResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetMemTop(ctx, &req)
	if err != nil {
		log.Errorf("DoGetMemTop error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

// GetDiskTop godoc
//
//	@Summary		DO FDR 磁盘不足 Top20
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoDiskTopResponse
//	@Router			/dashboard/v1/do/disk_top [GET]
func (s *DoDashboardService) GetDiskTop(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoDiskTopResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetDiskTop(ctx, &req)
	if err != nil {
		log.Errorf("DoGetDiskTop error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

// GetCloseTop godoc
//
//	@Summary		DO 关闭次数 Top 筛选器
//	@Tags			DoDashboard
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	dashboard_api.DoCloseTopResponse
//	@Router			/dashboard/v1/do/close_top [GET]
func (s *DoDashboardService) GetCloseTop(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoCloseTopResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoCoolTopRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetCloseTop(ctx, &req)
	if err != nil {
		log.Errorf("DoGetCloseTop error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return result, nil
}

func (s *DoDashboardService) GetTrend(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoTrendResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.DoTrendRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetTrend(ctx, &req)
	if err != nil {
		log.Errorf("DoGetTrend error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	return result, nil
}
