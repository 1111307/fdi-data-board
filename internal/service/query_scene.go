package service

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	querySceneApi "fdi_data_board/api/query_scene"
	"fdi_data_board/internal/biz"
)

type QuerySceneService struct {
	uc *biz.QuerySceneUseCase
}

func NewQuerySceneService(uc *biz.QuerySceneUseCase) *QuerySceneService {
	return &QuerySceneService{uc: uc}
}

// ==================== 用户端 ====================

// ListScenes godoc
//
//	@Summary		获取查询场景列表
//	@Tags			QueryScene
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			category	query		string								false	"分类标签"
//	@Param			status		query		int									false	"状态 1=启用 0=禁用，不传则返回全部"
//	@Success		200			{object}	querySceneApi.ListScenesResponse
//	@Router			/query_scene/v1/scenes/list [GET]
func (s *QuerySceneService) ListScenes(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.ListScenesResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.ListScenesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	status := int8(-1)
	if req.Status == 1 {
		status = 1
	} else if req.Status == 0 && ctx.Query("status") == "0" {
		status = 0
	}
	list, err := s.uc.ListScenes(ctx, req.Category, status)
	if err != nil {
		log.Errorf("ListScenes error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, item := range list {
		resp.List = append(resp.List, toProtoSceneItem(item))
	}
	return resp, nil
}

// GetSceneDetail godoc
//
//	@Summary		获取查询场景详情
//	@Tags			QueryScene
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			scene_id	query		int									true	"场景ID"
//	@Success		200			{object}	querySceneApi.GetSceneDetailResponse
//	@Router			/query_scene/v1/scenes/detail [GET]
func (s *QuerySceneService) GetSceneDetail(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.GetSceneDetailResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.SceneIDRequest
	if err := ctx.ShouldBindQuery(&req); err != nil || req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的场景ID"
		return resp, nil
	}
	detail, err := s.uc.GetSceneDetail(ctx, req.SceneId)
	if err != nil {
		log.Errorf("GetSceneDetail scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.Scene = toProtoSceneItem(detail.Scene)
	for _, p := range detail.Params {
		resp.Params = append(resp.Params, toProtoParamItem(p))
	}
	for _, w := range detail.Widgets {
		resp.Widgets = append(resp.Widgets, toProtoWidgetItem(w))
	}
	return resp, nil
}

// ExecuteScene godoc
//
//	@Summary		执行场景查询
//	@Tags			QueryScene
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.ExecuteSceneRequest	true	"场景ID及查询参数"
//	@Success		200		{object}	querySceneApi.ExecuteSceneResponse
//	@Router			/query_scene/v1/scenes/execute [POST]
func (s *QuerySceneService) ExecuteScene(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.ExecuteSceneResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.ExecuteSceneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的场景ID"
		return resp, nil
	}
	results, err := s.uc.ExecuteScene(ctx, req.SceneId, req.Params)
	if err != nil {
		log.Errorf("ExecuteScene scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, r := range results {
		wr := &querySceneApi.WidgetResult{
			WidgetId:     r.WidgetID,
			Title:        r.Title,
			DisplayType:  r.DisplayType,
			ResultConfig: r.ResultConfig,
			Error:        r.Error,
		}
		if r.Data != nil {
			wr.Columns = r.Data.Columns
			wr.Total = int32(r.Data.Total)
			for _, row := range r.Data.Rows {
				b, _ := json.Marshal(row)
				wr.RowsJson = append(wr.RowsJson, string(b))
			}
		}
		resp.Widgets = append(resp.Widgets, wr)
	}
	return resp, nil
}

// ListParams godoc
//
//	@Summary		获取场景参数列表
//	@Tags			QueryScene
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			scene_id	query		int									true	"场景ID"
//	@Success		200			{object}	querySceneApi.ListParamsResponse
//	@Router			/query_scene/v1/params/list [GET]
func (s *QuerySceneService) ListParams(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.ListParamsResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.ListParamsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil || req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的场景ID"
		return resp, nil
	}
	list, err := s.uc.ListParams(ctx, req.SceneId)
	if err != nil {
		log.Errorf("ListParams scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, p := range list {
		resp.List = append(resp.List, toProtoParamItem(p))
	}
	return resp, nil
}

// ListWidgets godoc
//
//	@Summary		获取场景组件列表
//	@Tags			QueryScene
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			scene_id	query		int									true	"场景ID"
//	@Success		200			{object}	querySceneApi.ListWidgetsResponse
//	@Router			/query_scene/v1/widgets/list [GET]
func (s *QuerySceneService) ListWidgets(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.ListWidgetsResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.ListWidgetsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil || req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的场景ID"
		return resp, nil
	}
	list, err := s.uc.ListWidgets(ctx, req.SceneId)
	if err != nil {
		log.Errorf("ListWidgets scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, w := range list {
		resp.List = append(resp.List, toProtoWidgetItem(w))
	}
	return resp, nil
}

// ==================== 管理端 ====================

// CreateScene godoc
//
//	@Summary		创建查询场景（管理端）
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.CreateSceneRequest	true	"场景数据"
//	@Success		200		{object}	querySceneApi.CreateSceneResponse
//	@Router			/query_scene/v1/admin/scenes/create [POST]
func (s *QuerySceneService) CreateScene(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.CreateSceneResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.CreateSceneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.Name == "" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "场景名称不能为空"
		return resp, nil
	}
	operator := ctx.GetHeader("X-User")
	param := protoCreateSceneReqToBizParam(operator, &req)
	sceneID, err := s.uc.SaveScene(ctx, param)
	if err != nil {
		log.Errorf("CreateScene error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.SceneId = sceneID
	return resp, nil
}

// UpdateScene godoc
//
//	@Summary		更新查询场景（管理端）
//	@Description	只更新传入的字段；params/widgets 不传则保留原有，传空数组则清空
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.UpdateSceneRequest	true	"场景数据（scene_id 必填）"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/scenes/update [POST]
func (s *QuerySceneService) UpdateScene(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.UpdateSceneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "scene_id 不能为空"
		return resp, nil
	}

	param := protoUpdateSceneReqToBizParam(&req)
	if err := s.uc.UpdateScene(ctx, param); err != nil {
		log.Errorf("UpdateScene scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return resp, nil
}

// DeleteScene godoc
//
//	@Summary		删除查询场景（管理端）
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.SceneIDRequest	true	"场景ID"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/scenes/delete [POST]
func (s *QuerySceneService) DeleteScene(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.SceneIDRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的场景ID"
		return resp, nil
	}
	if err := s.uc.DeleteScene(ctx, req.SceneId); err != nil {
		log.Errorf("DeleteScene scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return resp, nil
}

// SaveParam godoc
//
//	@Summary		创建或更新场景参数（管理端）
//	@Description	param_id=0 时创建，param_id>0 时更新
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.SaveParamRequest	true	"参数数据"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/params/save [POST]
func (s *QuerySceneService) SaveParam(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.SaveParamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	item := protoSaveParamReqToBizDTO(&req)
	if req.ParamId == 0 {
		if req.SceneId == 0 {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = "创建参数时 scene_id 不能为空"
			return resp, nil
		}
		if err := s.uc.CreateParam(ctx, req.SceneId, item); err != nil {
			log.Errorf("CreateParam scene_id=%d error: %v", req.SceneId, err)
			resp.Code = int32(gcode.CodeInternalError.Code())
			resp.Message = err.Error()
			return resp, nil
		}
	} else {
		if err := s.uc.UpdateParam(ctx, req.ParamId, item); err != nil {
			log.Errorf("UpdateParam param_id=%d error: %v", req.ParamId, err)
			resp.Code = int32(gcode.CodeInternalError.Code())
			resp.Message = err.Error()
			return resp, nil
		}
	}
	return resp, nil
}

// SaveWidget godoc
//
//	@Summary		创建或更新场景组件（管理端）
//	@Description	widget_id=0 时创建，widget_id>0 时更新
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.SaveWidgetRequest	true	"组件数据"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/widgets/save [POST]
func (s *QuerySceneService) SaveWidget(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.SaveWidgetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	item := protoSaveWidgetReqToBizDTO(&req)
	if req.WidgetId == 0 {
		if req.SceneId == 0 {
			resp.Code = int32(gcode.CodeInvalidParameter.Code())
			resp.Message = "创建组件时 scene_id 不能为空"
			return resp, nil
		}
		if err := s.uc.CreateWidget(ctx, req.SceneId, item); err != nil {
			log.Errorf("CreateWidget scene_id=%d error: %v", req.SceneId, err)
			resp.Code = int32(gcode.CodeInternalError.Code())
			resp.Message = err.Error()
			return resp, nil
		}
	} else {
		if err := s.uc.UpdateWidget(ctx, req.WidgetId, item); err != nil {
			log.Errorf("UpdateWidget widget_id=%d error: %v", req.WidgetId, err)
			resp.Code = int32(gcode.CodeInternalError.Code())
			resp.Message = err.Error()
			return resp, nil
		}
	}
	return resp, nil
}

// PreviewWidget godoc
//
//	@Summary		预览 Widget 查询（管理端）
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.PreviewWidgetRequest	true	"Widget 及参数"
//	@Success		200		{object}	querySceneApi.QueryResultResponse
//	@Router			/query_scene/v1/admin/widgets/preview [POST]
func (s *QuerySceneService) PreviewWidget(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.QueryResultResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.PreviewWidgetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.SceneId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的场景ID"
		return resp, nil
	}
	widgetParam := protoWidgetItemToBizDTO(req.Widget)
	result, err := s.uc.PreviewWidget(ctx, req.SceneId, widgetParam, req.Params)
	if err != nil {
		log.Errorf("PreviewWidget scene_id=%d error: %v", req.SceneId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.Columns = result.Columns
	resp.Total = int32(result.Total)
	for _, row := range result.Rows {
		b, _ := json.Marshal(row)
		resp.RowsJson = append(resp.RowsJson, string(b))
	}
	return resp, nil
}

// ==================== proto <-> biz DTO 转换 ====================

func protoCreateSceneReqToBizParam(operator string, req *querySceneApi.CreateSceneRequest) *biz.SaveSceneParam {
	return buildSaveSceneParam(0, operator, req.Name, req.Description, req.Category,
		int8(req.Status), int(req.SortOrder), req.Params, req.Widgets)
}

func protoUpdateSceneReqToBizParam(req *querySceneApi.UpdateSceneRequest) *biz.UpdateSceneParam {
	param := &biz.UpdateSceneParam{
		ID:        req.SceneId,
		Name:      req.Name,
		Description: req.Description,
		Category:  req.Category,
	}
	if req.Status != nil {
		v := int8(*req.Status)
		param.Status = &v
	}
	if req.SortOrder != nil {
		v := int(*req.SortOrder)
		param.SortOrder = &v
	}
	// 只要请求体中包含 params/widgets 字段就做全量替换
	if req.Params != nil {
		param.HasParamsUpdate = true
		for _, p := range req.Params {
			param.Params = append(param.Params, &biz.QuerySceneParamItem{
				KeyName:    p.KeyName,
				Label:      p.Label,
				ParamType:  p.ParamType,
				Required:   int8(p.Required),
				DefaultVal: p.DefaultVal,
				Options:    p.Options,
				DependsOn:  p.DependsOn,
				SortOrder:  int(p.SortOrder),
			})
		}
	}
	if req.Widgets != nil {
		param.HasWidgetsUpdate = true
		for _, w := range req.Widgets {
			param.Widgets = append(param.Widgets, protoWidgetItemToBizDTO(w))
		}
	}
	return param
}

func buildSaveSceneParam(sceneID uint64, operator, name, description, category string,
	status int8, sortOrder int,
	protoParams []*querySceneApi.SceneParamItem,
	protoWidgets []*querySceneApi.SceneWidgetItem,
) *biz.SaveSceneParam {
	params := make([]*biz.QuerySceneParamItem, 0, len(protoParams))
	for _, p := range protoParams {
		params = append(params, &biz.QuerySceneParamItem{
			KeyName:    p.KeyName,
			Label:      p.Label,
			ParamType:  p.ParamType,
			Required:   int8(p.Required),
			DefaultVal: p.DefaultVal,
			Options:    p.Options,
			DependsOn:  p.DependsOn,
			SortOrder:  int(p.SortOrder),
		})
	}
	widgets := make([]*biz.QuerySceneWidgetItem, 0, len(protoWidgets))
	for _, w := range protoWidgets {
		widgets = append(widgets, protoWidgetItemToBizDTO(w))
	}
	return &biz.SaveSceneParam{
		ID:          sceneID,
		Name:        name,
		Description: description,
		Category:    category,
		Status:      status,
		SortOrder:   sortOrder,
		CreatedBy:   operator,
		Params:      params,
		Widgets:     widgets,
	}
}

func protoSaveParamReqToBizDTO(req *querySceneApi.SaveParamRequest) *biz.QuerySceneParamItem {
	return &biz.QuerySceneParamItem{
		KeyName:    req.KeyName,
		Label:      req.Label,
		ParamType:  req.ParamType,
		Required:   int8(req.Required),
		DefaultVal: req.DefaultVal,
		Options:    req.Options,
		DependsOn:  req.DependsOn,
		SortOrder:  int(req.SortOrder),
	}
}

func protoSaveWidgetReqToBizDTO(req *querySceneApi.SaveWidgetRequest) *biz.QuerySceneWidgetItem {
	return &biz.QuerySceneWidgetItem{
		Title:        req.Title,
		SQLTemplate:  req.SqlTemplate,
		DisplayType:  req.DisplayType,
		ResultConfig: req.ResultConfig,
		MaxRows:      int(req.MaxRows),
		TimeoutSec:   int(req.TimeoutSec),
		SortOrder:    int(req.SortOrder),
	}
}

func protoWidgetItemToBizDTO(w *querySceneApi.SceneWidgetItem) *biz.QuerySceneWidgetItem {
	if w == nil {
		return &biz.QuerySceneWidgetItem{}
	}
	return &biz.QuerySceneWidgetItem{
		ID:           w.Id,
		Title:        w.Title,
		SQLTemplate:  w.SqlTemplate,
		DisplayType:  w.DisplayType,
		ResultConfig: w.ResultConfig,
		MaxRows:      int(w.MaxRows),
		TimeoutSec:   int(w.TimeoutSec),
		SortOrder:    int(w.SortOrder),
	}
}

func toProtoSceneItem(s *biz.QuerySceneItem) *querySceneApi.SceneItem {
	if s == nil {
		return nil
	}
	return &querySceneApi.SceneItem{
		Id:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Category:    s.Category,
		Status:      int32(s.Status),
		SortOrder:   int32(s.SortOrder),
		CreatedBy:   s.CreatedBy,
		CreatedAt:   s.CreatedAt,
	}
}

func toProtoParamItem(p *biz.QuerySceneParamItem) *querySceneApi.SceneParamItem {
	return &querySceneApi.SceneParamItem{
		Id:         p.ID,
		KeyName:    p.KeyName,
		Label:      p.Label,
		ParamType:  p.ParamType,
		Required:   int32(p.Required),
		DefaultVal: p.DefaultVal,
		Options:    p.Options,
		DependsOn:  p.DependsOn,
		SortOrder:  int32(p.SortOrder),
	}
}

func toProtoWidgetItem(w *biz.QuerySceneWidgetItem) *querySceneApi.SceneWidgetItem {
	return &querySceneApi.SceneWidgetItem{
		Id:           w.ID,
		Title:        w.Title,
		SqlTemplate:  w.SQLTemplate,
		DisplayType:  w.DisplayType,
		ResultConfig: w.ResultConfig,
		MaxRows:      int32(w.MaxRows),
		TimeoutSec:   int32(w.TimeoutSec),
		SortOrder:    int32(w.SortOrder),
	}
}
