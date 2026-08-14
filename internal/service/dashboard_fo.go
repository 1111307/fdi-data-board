package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

type FoDashboardService struct {
	uc *biz.FoDashboardUseCase
}

func NewFoDashboardService(uc *biz.FoDashboardUseCase) *FoDashboardService {
	return &FoDashboardService{uc: uc}
}

// ListFffTrigger godoc
//
//	@Summary	筛选器触发明细
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		event_name		query		string	false	"事件名"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Param		page			query		int		false	"页码，默认1"
//	@Param		page_size		query		int		false	"每页条数，默认50，最大500"
//	@Success	200				{object}	dashboard_api.FffTriggerResponse
//	@Router		/dashboard/v1/fo/detail/trigger [GET]
func (s *FoDashboardService) ListFffTrigger(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FffTriggerResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffTriggerRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListFffTrigger(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ListFffTrigger error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListFffClose godoc
//
//	@Summary	筛选器关闭明细
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Param		page			query		int		false	"页码，默认1"
//	@Param		page_size		query		int		false	"每页条数，默认50，最大500"
//	@Success	200				{object}	dashboard_api.FffCloseResponse
//	@Router		/dashboard/v1/fo/detail/close [GET]
func (s *FoDashboardService) ListFffClose(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FffCloseResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffCloseRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListFffClose(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ListFffClose error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListFdrTrigger godoc
//
//	@Summary	FDR 落盘明细
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		event_name		query		string	false	"事件名"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Param		page			query		int		false	"页码，默认1"
//	@Param		page_size		query		int		false	"每页条数，默认50，最大500"
//	@Success	200				{object}	dashboard_api.FdrTriggerResponse
//	@Router		/dashboard/v1/fo/detail/fdr [GET]
func (s *FoDashboardService) ListFdrTrigger(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FdrTriggerResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FdrTriggerRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListFdrTrigger(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ListFdrTrigger error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListFclTrigger godoc
//
//	@Summary	FCL 上传明细
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		event_name		query		string	false	"事件名"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Param		page			query		int		false	"页码，默认1"
//	@Param		page_size		query		int		false	"每页条数，默认50，最大500"
//	@Success	200				{object}	dashboard_api.FclTriggerResponse
//	@Router		/dashboard/v1/fo/detail/fcl [GET]
func (s *FoDashboardService) ListFclTrigger(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FclTriggerResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FclTriggerRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListFclTrigger(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ListFclTrigger error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListUuidDetail godoc
//
//	@Summary	全链路明细
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		event_names		query		string	false	"事件名，多选逗号分隔"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Param		only_fail		query		int		false	"1=仅看失败记录"
//	@Param		stage_filter	query		string	false	"阶段过滤: fff_discard/fdr_discard/fcl_discard/fcl_success"
//	@Param		page			query		int		false	"页码，默认1"
//	@Param		page_size		query		int		false	"每页条数，默认50，最大500"
//	@Success	200				{object}	dashboard_api.UuidDetailResponse
//	@Router		/dashboard/v1/fo/detail/uuid [GET]
func (s *FoDashboardService) ListUuidDetail(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.UuidDetailResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.UuidDetailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListUuidDetail(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ListUuidDetail error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetFunnel godoc
//
//	@Summary	数采全链路分析（节点统计 + 失败原因）
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		event_names		query		string	false	"事件名，多选逗号分隔"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD，不传默认当天"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{object}	dashboard_api.FunnelResponse
//	@Router		/dashboard/v1/fo/diag/funnel [GET]
func (s *FoDashboardService) GetFunnel(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FunnelResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FunnelRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetFunnel(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetFunnel error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetStageTrend godoc
//
//	@Summary	三阶段触发趋势（FFF/FDR/FCL 按日期聚合，堆叠柱图）
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		event_names		query		string	false	"事件名，多选逗号分隔"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD，不传默认近7天"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{object}	dashboard_api.StageTrendResponse
//	@Router		/dashboard/v1/fo/diag/stage_trend [GET]
func (s *FoDashboardService) GetStageTrend(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.StageTrendResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.StageTrendRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetStageTrend(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetStageTrend error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetCloseReason godoc
//
//	@Summary	算子关闭原因分布
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{object}	dashboard_api.CloseReasonResponse
//	@Router		/dashboard/v1/fo/diag/close_reason [GET]
func (s *FoDashboardService) GetCloseReason(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.CloseReasonResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.CloseReasonRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetCloseReason(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetCloseReason error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetDimensions godoc
//
//	@Summary	FO Dashboard 下拉维度（近7天活跃数据，30分钟缓存）
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Success	200	{object}	dashboard_api.FoDimensionsResponse
//	@Router		/dashboard/v1/fo/dimensions [GET]
func (s *FoDashboardService) GetDimensions(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FoDimensionsResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	result, err := s.uc.GetDimensions(ctx)
	if err != nil {
		log.Errorf("GetDimensions error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListFffRunning godoc
//
//	@Summary	筛选器运行明细
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Param		page			query		int		false	"页码，默认1"
//	@Param		page_size		query		int		false	"每页条数，默认50，最大500"
//	@Success	200				{object}	dashboard_api.FffRunningResponse
//	@Router		/dashboard/v1/fo/detail/running [GET]
func (s *FoDashboardService) ListFffRunning(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FffRunningResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffRunningRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListFffRunning(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ListFffRunning error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetFffRunningTrend godoc
//
//	@Summary	算子活跃车辆趋势（按天）
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	true	"算子名称（必填）"
//	@Param		start_dt		query		string	true	"开始日期 YYYY-MM-DD（必填）"
//	@Param		end_dt			query		string	true	"结束日期 YYYY-MM-DD（必填）"
//	@Param		project_name	query		string	false	"项目名称（选填）"
//	@Param		car_types		query		string	false	"车型，多选逗号分隔"
//	@Success	200				{object}	dashboard_api.FffRunningTrendResponse
//	@Router		/dashboard/v1/fo/running/trend [GET]
func (s *FoDashboardService) GetFffRunningTrend(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FffRunningTrendResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffRunningTrendRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetFffRunningTrend(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) || errors.Is(err, biz.ErrMissingRequired) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetFffRunningTrend error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetRunningOverview godoc
//
//	@Summary	筛选器运行健康概览（运行记录数/车辆数/开关占比）
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		car_types		query		string	false	"车型，多选逗号分隔"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{object}	dashboard_api.FoRunningOverviewResponse
//	@Router		/dashboard/v1/fo/running/overview [GET]
func (s *FoDashboardService) GetRunningOverview(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FoRunningOverviewResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffRunningRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetRunningOverview(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetRunningOverview error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}
	return result, nil
}

// GetFffOverview godoc
//
//	@Summary	FFF 触发概览（触发总数/成功数/成功率）
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		event_name		query		string	false	"事件名"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		car_types		query		string	false	"车型，多选逗号分隔"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{object}	dashboard_api.FoFffOverviewResponse
//	@Router		/dashboard/v1/fo/fff/overview [GET]
func (s *FoDashboardService) GetFffOverview(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.FoFffOverviewResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffTriggerRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetFffOverview(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetFffOverview error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}
	return result, nil
}

// GetFffFailReason godoc
//
//	@Summary	FFF 触发失败原因分布
//	@Tags		FoDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		filter_name		query		string	false	"筛选器名称"
//	@Param		event_names		query		string	false	"事件名，多选逗号分隔"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		car_types		query		string	false	"车型，多选逗号分隔"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{object}	dashboard_api.DoFailReasonResponse
//	@Router		/dashboard/v1/fo/fff/fail_reason [GET]
func (s *FoDashboardService) GetFffFailReason(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.DoFailReasonResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.FffTriggerRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	result, err := s.uc.GetFffFailReason(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("GetFffFailReason error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}
	return result, nil
}
