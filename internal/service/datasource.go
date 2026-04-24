package service

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	querySceneApi "fdi_data_board/api/query_scene"
	"fdi_data_board/internal/biz"
)

type DatasourceService struct {
	uc *biz.DatasourceUseCase
}

func NewDatasourceService(uc *biz.DatasourceUseCase) *DatasourceService {
	return &DatasourceService{uc: uc}
}

// ListDatasources godoc
//
//	@Summary		获取数据源列表（用户端）
//	@Tags			Datasource
//	@Produce		json
//	@Security		OAuth2Password
//	@Success		200	{object}	querySceneApi.ListDatasourcesResponse
//	@Router			/query_scene/v1/datasources/list [GET]
func (s *DatasourceService) ListDatasources(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.ListDatasourcesResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	// all=true 时返回全部（含禁用），用于管理端；默认只返回启用的，用于用户端
	onlyEnabled := ctx.Query("all") != "true"
	list, err := s.uc.List(ctx, onlyEnabled)
	if err != nil {
		log.Errorf("ListDatasources error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, d := range list {
		resp.List = append(resp.List, toDatasourceProto(d))
	}
	return resp, nil
}

// CreateDatasource godoc
//
//	@Summary		创建数据源（管理端）
//	@Tags			Datasource Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.CreateDatasourceRequest	true	"数据源配置"
//	@Success		200		{object}	querySceneApi.CreateDatasourceResponse
//	@Router			/query_scene/v1/admin/datasources/create [POST]
func (s *DatasourceService) CreateDatasource(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.CreateDatasourceResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.CreateDatasourceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.Name == "" || req.DsType == "" || req.Host == "" || req.Port == "" || req.DatabaseName == "" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "name、ds_type、host、port、database_name 不能为空"
		return resp, nil
	}
	if req.DsType != "doris" && req.DsType != "mysql" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "ds_type 只支持 doris / mysql"
		return resp, nil
	}

	param := &biz.CreateDatasourceParam{
		Name:         req.Name,
		Description:  req.Description,
		DsType:       req.DsType,
		Host:         req.Host,
		Port:         req.Port,
		Username:     req.Username,
		Password:     req.Password,
		DatabaseName: req.DatabaseName,
		MaxIdl:       int(req.MaxIdl),
		MaxOpen:      int(req.MaxOpen),
	}
	id, err := s.uc.Create(ctx, param)
	if err != nil {
		log.Errorf("CreateDatasource error: %v", err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.DatasourceId = id
	return resp, nil
}

// UpdateDatasource godoc
//
//	@Summary		更新数据源（管理端）
//	@Tags			Datasource Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.UpdateDatasourceRequest	true	"更新字段"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/datasources/update [POST]
func (s *DatasourceService) UpdateDatasource(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.UpdateDatasourceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.DatasourceId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "datasource_id 不能为空"
		return resp, nil
	}

	param := protoUpdateDatasourceReqToBizParam(&req)
	if err := s.uc.Update(ctx, param); err != nil {
		log.Errorf("UpdateDatasource id=%d error: %v", req.DatasourceId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return resp, nil
}

// DeleteDatasource godoc
//
//	@Summary		删除数据源（管理端，软删）
//	@Tags			Datasource Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.DatasourceIDRequest	true	"数据源ID"
//	@Success		200		{object}	querySceneApi.BaseResponse
//	@Router			/query_scene/v1/admin/datasources/delete [POST]
func (s *DatasourceService) DeleteDatasource(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.DatasourceIDRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.DatasourceId == 0 {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "无效的 datasource_id"
		return resp, nil
	}
	if err := s.uc.Delete(ctx, req.DatasourceId); err != nil {
		log.Errorf("DeleteDatasource id=%d error: %v", req.DatasourceId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	return resp, nil
}

// GetSchemaTables godoc
//
//	@Summary		获取数据源下的表列表（管理端）
//	@Tags			Datasource Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.SchemaTablesRequest	true	"数据源ID（0=默认Doris）"
//	@Success		200		{object}	querySceneApi.SchemaTablesResponse
//	@Router			/query_scene/v1/admin/datasources/schema/tables [POST]
func (s *DatasourceService) GetSchemaTables(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.SchemaTablesResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.SchemaTablesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	tables, err := s.uc.GetTables(ctx, req.DatasourceId)
	if err != nil {
		log.Errorf("GetSchemaTables datasource_id=%d error: %v", req.DatasourceId, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	resp.Tables = tables
	return resp, nil
}

// GetSchemaColumns godoc
//
//	@Summary		获取指定表的字段列表（管理端）
//	@Tags			Datasource Admin
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			body	body		querySceneApi.SchemaColumnsRequest	true	"数据源ID和表名"
//	@Success		200		{object}	querySceneApi.SchemaColumnsResponse
//	@Router			/query_scene/v1/admin/datasources/schema/columns [POST]
func (s *DatasourceService) GetSchemaColumns(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &querySceneApi.SchemaColumnsResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	var req querySceneApi.SchemaColumnsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	if req.TableName == "" {
		resp.Code = int32(gcode.CodeInvalidParameter.Code())
		resp.Message = "table_name 不能为空"
		return resp, nil
	}
	columns, err := s.uc.GetColumns(ctx, req.DatasourceId, req.TableName)
	if err != nil {
		log.Errorf("GetSchemaColumns datasource_id=%d table=%s error: %v", req.DatasourceId, req.TableName, err)
		resp.Code = int32(gcode.CodeInternalError.Code())
		resp.Message = err.Error()
		return resp, nil
	}
	for _, c := range columns {
		resp.Columns = append(resp.Columns, &querySceneApi.ColumnInfo{
			Field: c.Field,
			Type:  c.Type,
		})
	}
	return resp, nil
}

// ==================== 转换函数 ====================

func toDatasourceProto(d *biz.DatasourceItem) *querySceneApi.DatasourceItem {
	return &querySceneApi.DatasourceItem{
		Id:           d.ID,
		Name:         d.Name,
		Description:  d.Description,
		DsType:       d.DsType,
		Host:         d.Host,
		Port:         d.Port,
		Username:     d.Username,
		DatabaseName: d.DatabaseName,
		MaxIdl:       int32(d.MaxIdl),
		MaxOpen:      int32(d.MaxOpen),
		Status:       int32(d.Status),
		CreateTime:   d.CreateTime,
	}
}

func protoUpdateDatasourceReqToBizParam(req *querySceneApi.UpdateDatasourceRequest) *biz.UpdateDatasourceParam {
	param := &biz.UpdateDatasourceParam{ID: req.DatasourceId}
	if req.Name != nil {
		param.Name = req.Name
	}
	if req.Description != nil {
		param.Description = req.Description
	}
	if req.Host != nil {
		param.Host = req.Host
	}
	if req.Port != nil {
		param.Port = req.Port
	}
	if req.Username != nil {
		param.Username = req.Username
	}
	if req.Password != nil {
		param.Password = req.Password
	}
	if req.DatabaseName != nil {
		param.DatabaseName = req.DatabaseName
	}
	if req.MaxIdl != nil {
		v := int(*req.MaxIdl)
		param.MaxIdl = &v
	}
	if req.MaxOpen != nil {
		v := int(*req.MaxOpen)
		param.MaxOpen = &v
	}
	if req.Status != nil {
		v := int8(*req.Status)
		param.Status = &v
	}
	return param
}
