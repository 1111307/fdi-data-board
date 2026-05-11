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

type SceneGroupService struct {
	uc *biz.SceneGroupUseCase
}

func NewSceneGroupService(uc *biz.SceneGroupUseCase) *SceneGroupService {
	return &SceneGroupService{uc: uc}
}

// ListGroups godoc
//
//	@Summary		获取所有场景集（管理端）
//	@Tags			SceneGroup Admin
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	querySceneApi.ListGroupsResponse
//	@Router			/query_scene/v1/admin/scene_groups/list [GET]
func (s *SceneGroupService) ListGroups(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.ListGroupsResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	list, err := s.uc.List(ctx)
	if err != nil {
		log.Errorf("ListGroups error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, g := range list {
		resp.List = append(resp.List, toGroupProto(g))
	}
	return resp, nil
}

// GetGroupByPageKey godoc
//
//	@Summary		按 page_key 获取场景集（用户端）
//	@Tags			SceneGroup
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			page_key	query		string								true	"页面key"
//	@Success		200			{object}	querySceneApi.GetGroupByPageKeyResponse
//	@Router			/query_scene/v1/scene_groups/page [GET]
func (s *SceneGroupService) GetGroupByPageKey(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.GetGroupByPageKeyResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.GetGroupByPageKeyRequest
	if err := ctx.ShouldBindQuery(&req); err != nil || req.PageKey == "" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "page_key 不能为空"
		return resp, nil
	}
	group, err := s.uc.GetByPageKey(ctx, req.PageKey)
	if err != nil {
		log.Errorf("GetGroupByPageKey page_key=%s error: %v", req.PageKey, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if group != nil {
		resp.Group = toGroupProto(group)
	}
	return resp, nil
}

// GetDimensionValues godoc
//
//	@Summary		获取维度字段的可选值（用户端）
//	@Tags			SceneGroup
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			group_id	query		int									true	"场景集ID"
//	@Param			field_name	query		string								true	"字段名"
//	@Success		200			{object}	querySceneApi.GetDimensionValuesResponse
//	@Router			/query_scene/v1/scene_groups/dimensions [GET]
func (s *SceneGroupService) GetDimensionValues(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.GetDimensionValuesResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.GetDimensionValuesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil || req.GroupId == 0 || req.FieldName == "" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "group_id 和 field_name 不能为空"
		return resp, nil
	}
	values, err := s.uc.GetDimensionValues(ctx, req.GroupId, req.FieldName)
	if err != nil {
		log.Errorf("GetDimensionValues group_id=%d field=%s error: %v", req.GroupId, req.FieldName, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.Values = values
	return resp, nil
}

// CreateGroup godoc
//
//	@Summary		创建场景集（管理端）
//	@Tags			SceneGroup Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.CreateGroupRequest	true	"场景集数据"
//	@Success		200		{object}	querySceneApi.CreateGroupResponse
//	@Router			/query_scene/v1/admin/scene_groups/create [POST]
func (s *SceneGroupService) CreateGroup(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.CreateGroupResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.CreateGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.Name == "" || req.PageKey == "" || req.SourceTable == "" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "name、page_key、source_table 不能为空"
		return resp, nil
	}
	param := &biz.CreateGroupParam{
		Name:           req.Name,
		Description:    req.Description,
		PageKey:        req.PageKey,
		DatasourceID:   req.DatasourceId,
		SourceTable:    req.SourceTable,
		DimFields:      protoDimFieldsToBiz(req.DimFields),
		PartitionField: req.PartitionField,
		LookbackDays:   int(req.LookbackDays),
	}
	id, err := s.uc.Create(ctx, param)
	if err != nil {
		log.Errorf("CreateGroup error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.GroupId = id
	return resp, nil
}

// UpdateGroup godoc
//
//	@Summary		更新场景集（管理端）
//	@Tags			SceneGroup Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.UpdateGroupRequest	true	"更新字段"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/scene_groups/update [POST]
func (s *SceneGroupService) UpdateGroup(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.UpdateGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.GroupId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "group_id 不能为空"
		return resp, nil
	}

	// dim_fields 有传则序列化
	var dimJSON string
	hasDim := len(req.DimFields) > 0
	if hasDim {
		b, _ := json.Marshal(protoDimFieldsToBiz(req.DimFields))
		dimJSON = string(b)
	}

	param := &biz.UpdateGroupParam{
		ID:           req.GroupId,
		Name:         req.Name,
		Description:  req.Description,
		SourceTable:  req.SourceTable,
		HasDimUpdate: hasDim,
	}
	if hasDim {
		param.DimensionFields = dimJSON
	}
	if req.DatasourceId != nil {
		param.DatasourceID = req.DatasourceId
	}
	if req.Status != nil {
		v := int8(*req.Status)
		param.Status = &v
	}
	if req.PartitionField != nil {
		param.PartitionField = req.PartitionField
	}
	if req.LookbackDays != nil {
		v := int(*req.LookbackDays)
		param.LookbackDays = &v
	}

	if err := s.uc.Update(ctx, param); err != nil {
		log.Errorf("UpdateGroup id=%d error: %v", req.GroupId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return resp, nil
}

// DeleteGroup godoc
//
//	@Summary		删除场景集（管理端）
//	@Tags			SceneGroup Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.GroupIDRequest	true	"场景集ID"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/scene_groups/delete [POST]
func (s *SceneGroupService) DeleteGroup(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.GroupIDRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.GroupId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的 group_id"
		return resp, nil
	}
	if err := s.uc.Delete(ctx, req.GroupId); err != nil {
		log.Errorf("DeleteGroup id=%d error: %v", req.GroupId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return resp, nil
}

func toGroupProto(g *biz.SceneGroupItem) *querySceneApi.SceneGroupItem {
	return &querySceneApi.SceneGroupItem{
		Id:             g.ID,
		Name:           g.Name,
		Description:    g.Description,
		PageKey:        g.PageKey,
		DatasourceId:   g.DatasourceID,
		SourceTable:    g.SourceTable,
		DimFields:      bizDimFieldsToProto(g.DimFields),
		Status:         int32(g.Status),
		PartitionField: g.PartitionField,
		LookbackDays:   int32(g.LookbackDays),
	}
}

func protoDimFieldsToBiz(fields []*querySceneApi.DimFieldConfig) []biz.DimField {
	result := make([]biz.DimField, 0, len(fields))
	for _, f := range fields {
		result = append(result, biz.DimField{
			Name:     f.Name,
			Type:     f.Type,
			StartKey: f.StartKey,
			EndKey:   f.EndKey,
			Label:    f.Label,
		})
	}
	return result
}

func bizDimFieldsToProto(fields []biz.DimField) []*querySceneApi.DimFieldConfig {
	result := make([]*querySceneApi.DimFieldConfig, 0, len(fields))
	for _, f := range fields {
		result = append(result, &querySceneApi.DimFieldConfig{
			Name:     f.Name,
			Type:     f.Type,
			StartKey: f.StartKey,
			EndKey:   f.EndKey,
			Label:    f.Label,
		})
	}
	return result
}
