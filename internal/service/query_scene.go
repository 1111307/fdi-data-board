package service

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	"fdi_data_board/internal/biz"
)

type QuerySceneService struct {
	uc *biz.QuerySceneUseCase
}

func NewQuerySceneService(uc *biz.QuerySceneUseCase) *QuerySceneService {
	return &QuerySceneService{uc: uc}
}

// ==================== 通用响应 ====================

type querySceneResp struct {
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *querySceneResp) GetCode() int32    { return r.Code }
func (r *querySceneResp) GetMessage() string { return r.Message }

func okResp(data interface{}) *querySceneResp {
	return &querySceneResp{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
		Data:    data,
	}
}

func errResp(code gcode.Code, msg string) *querySceneResp {
	return &querySceneResp{
		Code:    int32(code.Code()),
		Message: msg,
	}
}

// ==================== 用户端接口 ====================

// ListScenes godoc
//
//	@Summary		获取查询场景列表
//	@Description	获取查询场景列表
//	@Tags			QueryScene
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			category	query		string	false	"分类标签"
//	@Param			status		query		int		false	"状态 1=启用 0=禁用，不传则返回全部"
//	@Success		200			{object}	querySceneResp
//	@Failure		400			{object}	querySceneResp
//	@Router			/query_scene/v1/scenes [GET]
func (s *QuerySceneService) ListScenes(ctx *gin.Context) (api.HttpResponse, error) {
	category := ctx.Query("category")
	status := int8(-1)
	if v := ctx.Query("status"); v == "0" {
		status = 0
	} else if v == "1" {
		status = 1
	}

	list, err := s.uc.ListScenes(ctx, category, status)
	if err != nil {
		log.Errorf("ListScenes error: %v", err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(map[string]interface{}{"list": list}), nil
}

// GetSceneDetail godoc
//
//	@Summary		获取查询场景详情
//	@Description	获取查询场景详情（含参数定义和组件列表）
//	@Tags			QueryScene
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			id	path		int	true	"场景ID"
//	@Success		200	{object}	querySceneResp
//	@Failure		400	{object}	querySceneResp
//	@Router			/query_scene/v1/scenes/{id} [GET]
func (s *QuerySceneService) GetSceneDetail(ctx *gin.Context) (api.HttpResponse, error) {
	sceneID, err := parseUintParam(ctx, "id")
	if err != nil {
		return errResp(gcode.CodeInvalidParameter, "无效的场景ID"), nil
	}

	detail, err := s.uc.GetSceneDetail(ctx, sceneID)
	if err != nil {
		log.Errorf("GetSceneDetail scene_id=%d error: %v", sceneID, err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(detail), nil
}

// ExecuteScene godoc
//
//	@Summary		执行场景查询
//	@Description	执行场景下所有 Widget 查询，返回结果集
//	@Tags			QueryScene
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			id		path		int					true	"场景ID"
//	@Param			body	body		executeSceneRequest	true	"查询参数"
//	@Success		200		{object}	querySceneResp
//	@Failure		400		{object}	querySceneResp
//	@Router			/query_scene/v1/scenes/{id}/execute [POST]
func (s *QuerySceneService) ExecuteScene(ctx *gin.Context) (api.HttpResponse, error) {
	sceneID, err := parseUintParam(ctx, "id")
	if err != nil {
		return errResp(gcode.CodeInvalidParameter, "无效的场景ID"), nil
	}

	var req executeSceneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return errResp(gcode.CodeInvalidParameter, err.Error()), nil
	}

	results, err := s.uc.ExecuteScene(ctx, sceneID, req.Params)
	if err != nil {
		log.Errorf("ExecuteScene scene_id=%d error: %v", sceneID, err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(map[string]interface{}{"widgets": results}), nil
}

// ==================== 管理端接口 ====================

// CreateScene godoc
//
//	@Summary		创建查询场景（管理端）
//	@Description	创建查询场景，含参数定义和组件
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		saveSceneRequest	true	"场景数据"
//	@Success		200		{object}	querySceneResp
//	@Failure		400		{object}	querySceneResp
//	@Router			/query_scene/v1/admin/scenes [POST]
func (s *QuerySceneService) CreateScene(ctx *gin.Context) (api.HttpResponse, error) {
	var req saveSceneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return errResp(gcode.CodeInvalidParameter, err.Error()), nil
	}
	if req.Name == "" {
		return errResp(gcode.CodeInvalidParameter, "场景名称不能为空"), nil
	}

	operator := ctx.GetHeader("X-User")
	if err := s.uc.SaveScene(ctx, req.toBizParam(0, operator)); err != nil {
		log.Errorf("CreateScene error: %v", err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(nil), nil
}

// UpdateScene godoc
//
//	@Summary		更新查询场景（管理端）
//	@Description	更新查询场景，含参数定义和组件（全量覆盖）
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			id		path		int					true	"场景ID"
//	@Param			body	body		saveSceneRequest	true	"场景数据"
//	@Success		200		{object}	querySceneResp
//	@Failure		400		{object}	querySceneResp
//	@Router			/query_scene/v1/admin/scenes/{id} [PUT]
func (s *QuerySceneService) UpdateScene(ctx *gin.Context) (api.HttpResponse, error) {
	sceneID, err := parseUintParam(ctx, "id")
	if err != nil {
		return errResp(gcode.CodeInvalidParameter, "无效的场景ID"), nil
	}

	var req saveSceneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return errResp(gcode.CodeInvalidParameter, err.Error()), nil
	}

	operator := ctx.GetHeader("X-User")
	if err := s.uc.SaveScene(ctx, req.toBizParam(sceneID, operator)); err != nil {
		log.Errorf("UpdateScene scene_id=%d error: %v", sceneID, err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(nil), nil
}

// DeleteScene godoc
//
//	@Summary		删除查询场景（管理端）
//	@Description	软删除查询场景
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			id	path		int	true	"场景ID"
//	@Success		200	{object}	querySceneResp
//	@Failure		400	{object}	querySceneResp
//	@Router			/query_scene/v1/admin/scenes/{id} [DELETE]
func (s *QuerySceneService) DeleteScene(ctx *gin.Context) (api.HttpResponse, error) {
	sceneID, err := parseUintParam(ctx, "id")
	if err != nil {
		return errResp(gcode.CodeInvalidParameter, "无效的场景ID"), nil
	}

	if err := s.uc.DeleteScene(ctx, sceneID); err != nil {
		log.Errorf("DeleteScene scene_id=%d error: %v", sceneID, err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(nil), nil
}

// PreviewWidget godoc
//
//	@Summary		预览 Widget 查询（管理端）
//	@Description	管理员预览单个 Widget 的查询结果
//	@Tags			QueryScene Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			id		path		int						true	"场景ID"
//	@Param			body	body		previewWidgetRequest	true	"Widget 及参数"
//	@Success		200		{object}	querySceneResp
//	@Failure		400		{object}	querySceneResp
//	@Router			/query_scene/v1/admin/scenes/{id}/preview [POST]
func (s *QuerySceneService) PreviewWidget(ctx *gin.Context) (api.HttpResponse, error) {
	sceneID, err := parseUintParam(ctx, "id")
	if err != nil {
		return errResp(gcode.CodeInvalidParameter, "无效的场景ID"), nil
	}

	var req previewWidgetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return errResp(gcode.CodeInvalidParameter, err.Error()), nil
	}

	result, err := s.uc.PreviewWidget(ctx, sceneID, req.Widget, req.Params)
	if err != nil {
		log.Errorf("PreviewWidget scene_id=%d error: %v", sceneID, err)
		return errResp(gcode.CodeInternalError, err.Error()), nil
	}
	return okResp(result), nil
}

// ==================== 请求结构 ====================

type executeSceneRequest struct {
	Params map[string]string `json:"params"`
}

type saveSceneRequest struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Category    string                      `json:"category"`
	Status      int8                        `json:"status"`
	SortOrder   int                         `json:"sort_order"`
	Params      []*biz.QuerySceneParamItem  `json:"params"`
	Widgets     []*biz.QuerySceneWidgetItem `json:"widgets"`
}

func (r *saveSceneRequest) toBizParam(sceneID uint64, operator string) *biz.SaveSceneParam {
	return &biz.SaveSceneParam{
		ID:          sceneID,
		Name:        r.Name,
		Description: r.Description,
		Category:    r.Category,
		Status:      r.Status,
		SortOrder:   r.SortOrder,
		CreatedBy:   operator,
		Params:      r.Params,
		Widgets:     r.Widgets,
	}
}

type previewWidgetRequest struct {
	Widget *biz.QuerySceneWidgetItem `json:"widget"`
	Params map[string]string         `json:"params"`
}

// ==================== 工具函数 ====================

func parseUintParam(ctx *gin.Context, key string) (uint64, error) {
	var id uint64
	_, err := fmt.Sscanf(ctx.Param(key), "%d", &id)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}
