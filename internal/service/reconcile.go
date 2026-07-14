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

type ReconcileService struct {
	uc *biz.ReconcileUseCase
}

func NewReconcileService(uc *biz.ReconcileUseCase) *ReconcileService {
	return &ReconcileService{uc: uc}
}

// GetOverview godoc
//
//	@Summary	对账总览（当日健康度）
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Success	200		{object}	dashboard_api.ReconcileOverviewResponse
//	@Router		/dashboard/v1/reconcile/overview [GET]
func (s *ReconcileService) GetOverview(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileOverviewResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileOverviewRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetOverview(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetOverview error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetTrend godoc
//
//	@Summary	对账时间趋势
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		start_dt	query		string	false	"开始日期，不传默认近7天"
//	@Param		end_dt		query		string	false	"结束日期"
//	@Success	200			{object}	dashboard_api.ReconcileTrendResponse
//	@Router		/dashboard/v1/reconcile/trend [GET]
func (s *ReconcileService) GetTrend(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileTrendResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileTrendRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetTrend(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidDateRange) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ReconcileGetTrend error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetModule godoc
//
//	@Summary	对账模块维度
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Param		project	query		string	false	"可选，交叉钻取到项目内模块分布"
//	@Success	200		{object}	dashboard_api.ReconcileModuleResponse
//	@Router		/dashboard/v1/reconcile/module [GET]
func (s *ReconcileService) GetModule(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileModuleResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileModuleRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetModule(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetModule error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetProject godoc
//
//	@Summary	对账项目维度
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date		query		string	false	"日期，不传默认今天"
//	@Param		order_by	query		string	false	"missing(默认) / expected"
//	@Param		limit		query		int		false	"返回条数，默认20"
//	@Success	200			{object}	dashboard_api.ReconcileProjectResponse
//	@Router		/dashboard/v1/reconcile/project [GET]
func (s *ReconcileService) GetProject(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileProjectResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileProjectRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetProject(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidOrderBy) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ReconcileGetProject error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetDecodeStatus godoc
//
//	@Summary	上游解码状态分布
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Success	200		{object}	dashboard_api.ReconcileDecodeStatusResponse
//	@Router		/dashboard/v1/reconcile/decode_status [GET]
func (s *ReconcileService) GetDecodeStatus(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileDecodeStatusResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileDecodeStatusRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetDecodeStatus(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetDecodeStatus error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListMd5 godoc
//
//	@Summary	差异 md5 排行
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Param		type	query		string	false	"missing(漏消费Top，默认) / convert_failed / landing_failed / decode_failed"
//	@Param		limit	query		int		false	"返回条数，默认20"
//	@Success	200		{object}	dashboard_api.ReconcileMd5ListResponse
//	@Router		/dashboard/v1/reconcile/md5 [GET]
func (s *ReconcileService) ListMd5(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileMd5ListResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileMd5ListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListMd5(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidMd5Type) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ReconcileListMd5 error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetMd5Detail godoc
//
//	@Summary	单 md5 event 级明细
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Param		md5		query		string	true	"bag 文件唯一标识"
//	@Success	200		{object}	dashboard_api.ReconcileMd5DetailResponse
//	@Router		/dashboard/v1/reconcile/md5_detail [GET]
func (s *ReconcileService) GetMd5Detail(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileMd5DetailResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileMd5DetailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetMd5Detail(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrMd5Required) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ReconcileGetMd5Detail error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// ListEventList godoc
//
//	@Summary	event/uuid 级明细列表（不依赖先选 md5）
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date		query		string	false	"日期，不传默认今天"
//	@Param		module_name	query		string	false	"按模块过滤"
//	@Param		type		query		string	false	"missing(默认) / extra / send_failed / convert_failed / landing_failed / mismatched"
//	@Param		page		query		int		false	"页码，默认1"
//	@Param		page_size	query		int		false	"每页条数，默认20，最大200"
//	@Success	200			{object}	dashboard_api.ReconcileEventListResponse
//	@Router		/dashboard/v1/reconcile/event_list [GET]
func (s *ReconcileService) ListEventList(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileEventListResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileEventListRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.ListEventList(ctx, &req)
	if err != nil {
		if errors.Is(err, biz.ErrInvalidEventType) {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = err.Error()
			return resp, nil
		}
		log.Errorf("ReconcileListEventList error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetRecordConsistency godoc
//
//	@Summary	record↔detail 内部一致性（解码阶段是否丢行）
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Param		limit	query		int		false	"返回条数，默认20"
//	@Success	200		{object}	dashboard_api.ReconcileRecordConsistencyResponse
//	@Router		/dashboard/v1/reconcile/record_consistency [GET]
func (s *ReconcileService) GetRecordConsistency(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileRecordConsistencyResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileRecordConsistencyRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetRecordConsistency(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetRecordConsistency error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetUuidSource godoc
//
//	@Summary	uuid 来源细分（real / gen_fallback）
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Success	200		{object}	dashboard_api.ReconcileUuidSourceResponse
//	@Router		/dashboard/v1/reconcile/uuid_source [GET]
func (s *ReconcileService) GetUuidSource(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileUuidSourceResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileUuidSourceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetUuidSource(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetUuidSource error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetFailureSummary godoc
//
//	@Summary	失败原因汇总（解码失败/发送失败/转换失败/落库失败）
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date	query		string	false	"日期，不传默认今天"
//	@Success	200		{object}	dashboard_api.ReconcileFailureSummaryResponse
//	@Router		/dashboard/v1/reconcile/failure_summary [GET]
func (s *ReconcileService) GetFailureSummary(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcileFailureSummaryResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcileFailureSummaryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetFailureSummary(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetFailureSummary error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}

// GetPipelineTree godoc
//
//	@Summary	全链路树形聚合（tar包→event解析→event落库，看分流去哪了）
//	@Tags		ReconcileDashboard
//	@Produce	json
//	@Security	OAuth2Password
//	@Param		date		query		string	false	"日期，不传默认今天"
//	@Param		project		query		string	false	"可选，过滤到单个项目"
//	@Param		module_name	query		string	false	"可选，只影响 event 解析/落库分支"
//	@Param		md5			query		string	false	"可选，单包下钻模式"
//	@Success	200			{object}	dashboard_api.ReconcilePipelineTreeResponse
//	@Router		/dashboard/v1/reconcile/pipeline_tree [GET]
func (s *ReconcileService) GetPipelineTree(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &dashboard_api.ReconcilePipelineTreeResponse{}
	resp.Code = int32(gcode.CodeOK.Code())
	resp.Message = gcode.CodeOK.Message()

	var req dashboard_api.ReconcilePipelineTreeRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}

	result, err := s.uc.GetPipelineTree(ctx, &req)
	if err != nil {
		log.Errorf("ReconcileGetPipelineTree error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = "internal server error"
		return resp, nil
	}

	return result, nil
}
