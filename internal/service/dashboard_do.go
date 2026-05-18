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
