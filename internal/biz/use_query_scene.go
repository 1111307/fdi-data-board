package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/data/orm"
)

// QuerySceneRepo 场景仓储接口
type QuerySceneRepo interface {
	ListScenes(ctx context.Context, category string, status int8) ([]*orm.QuerySceneDo, error)
	GetScene(ctx context.Context, sceneID uint64) (*orm.QuerySceneDo, error)
	GetSceneParams(ctx context.Context, sceneID uint64) ([]*orm.QuerySceneParamDo, error)
	GetSceneWidgets(ctx context.Context, sceneID uint64) ([]*orm.QuerySceneWidgetDo, error)
	SaveScene(ctx context.Context, scene *orm.QuerySceneDo, params []*orm.QuerySceneParamDo, widgets []*orm.QuerySceneWidgetDo) (uint64, error)
	DeleteScene(ctx context.Context, sceneID uint64) error
	ExecuteWidget(ctx context.Context, widget *orm.QuerySceneWidgetDo, userParams map[string]string, datasourceID uint64) (*QueryResult, error)

	CreateParam(ctx context.Context, param *orm.QuerySceneParamDo) (uint64, error)
	UpdateParam(ctx context.Context, param *UpdateParamParam) error
	CreateWidget(ctx context.Context, widget *orm.QuerySceneWidgetDo) error
	UpdateWidget(ctx context.Context, widget *orm.QuerySceneWidgetDo) error
	UpdateScene(ctx context.Context, param *UpdateSceneParam) error
}

// QueryResult 查询结果
type QueryResult struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
	Total   int             `json:"total"`
}

// QuerySceneUseCase 场景业务用例
type QuerySceneUseCase struct {
	repo QuerySceneRepo
}

func NewQuerySceneUseCase(repo QuerySceneRepo) *QuerySceneUseCase {
	return &QuerySceneUseCase{repo: repo}
}

// ==================== DTO ====================

type QuerySceneItem struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	Status         int8   `json:"status"`
	SortOrder      int    `json:"sort_order"`
	CreatedBy      string `json:"created_by"`
	CreatedAt      int64  `json:"create_time"`
	DatasourceID   uint64 `json:"datasource_id"`
	DatasourceName string `json:"datasource_name"`
}

type QuerySceneParamItem struct {
	ID         uint64 `json:"id"`
	KeyName    string `json:"key_name"`
	Label      string `json:"label"`
	ParamType  string `json:"param_type"`
	Required   int8   `json:"required"`
	DefaultVal string `json:"default_val"`
	Options    string `json:"options"`
	DependsOn  string `json:"depends_on"`
	SortOrder  int    `json:"sort_order"`
}

type QuerySceneWidgetItem struct {
	ID           uint64 `json:"id"`
	Title        string `json:"title"`
	SQLTemplate  string `json:"sql_template"`
	DisplayType  string `json:"display_type"`
	ResultConfig string `json:"result_config"`
	MaxRows      int    `json:"max_rows"`
	TimeoutSec   int    `json:"timeout_sec"`
	SortOrder    int    `json:"sort_order"`
}

type QuerySceneDetail struct {
	Scene   *QuerySceneItem        `json:"scene"`
	Params  []*QuerySceneParamItem `json:"params"`
	Widgets []*QuerySceneWidgetItem `json:"widgets"`
}

type SaveSceneParam struct {
	ID           uint64
	Name         string
	Description  string
	Category     string
	Status       int8
	SortOrder    int
	CreatedBy    string
	DatasourceID uint64
	Params       []*QuerySceneParamItem
	Widgets      []*QuerySceneWidgetItem
}

// UpdateSceneParam 用于部分更新，指针为 nil 表示不更新该字段
type UpdateSceneParam struct {
	ID           uint64
	Name         *string
	Description  *string
	Category     *string
	Status       *int8
	SortOrder    *int
	DatasourceID *uint64
	// nil 表示不操作；non-nil（含空切片）表示全量替换
	Params           []*QuerySceneParamItem
	Widgets          []*QuerySceneWidgetItem
	HasParamsUpdate  bool
	HasWidgetsUpdate bool
}

// UpdateParamParam 部分更新参数，指针为 nil 表示不更新该字段
type UpdateParamParam struct {
	ID         uint64
	KeyName    *string
	Label      *string
	ParamType  *string
	Required   *int8
	DefaultVal *string
	Options    *string
	DependsOn  *string
	SortOrder  *int
}

type WidgetQueryResult struct {
	WidgetID     uint64       `json:"widget_id"`
	Title        string       `json:"title"`
	DisplayType  string       `json:"display_type"`
	ResultConfig string       `json:"result_config"`
	Data         *QueryResult `json:"data"`
	Error        string       `json:"error,omitempty"`
}

// ==================== UseCase 方法 ====================

func (uc *QuerySceneUseCase) ListScenes(ctx context.Context, category string, status int8) ([]*QuerySceneItem, error) {
	list, err := uc.repo.ListScenes(ctx, category, status)
	if err != nil {
		return nil, err
	}
	result := make([]*QuerySceneItem, 0, len(list))
	for _, s := range list {
		result = append(result, toSceneItem(s))
	}
	return result, nil
}

func (uc *QuerySceneUseCase) GetSceneDetail(ctx context.Context, sceneID uint64) (*QuerySceneDetail, error) {
	scene, err := uc.repo.GetScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	params, err := uc.repo.GetSceneParams(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	widgets, err := uc.repo.GetSceneWidgets(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	paramItems := make([]*QuerySceneParamItem, 0, len(params))
	for _, p := range params {
		paramItems = append(paramItems, toParamItem(p))
	}
	widgetItems := make([]*QuerySceneWidgetItem, 0, len(widgets))
	for _, w := range widgets {
		widgetItems = append(widgetItems, toWidgetItem(w))
	}

	return &QuerySceneDetail{
		Scene:   toSceneItem(scene),
		Params:  paramItems,
		Widgets: widgetItems,
	}, nil
}

// SaveScene 创建或更新场景，返回 scene_id（创建时为新 ID，更新时为传入 ID）
func (uc *QuerySceneUseCase) SaveScene(ctx context.Context, param *SaveSceneParam) (uint64, error) {
	now := time.Now()

	sceneDo := &orm.QuerySceneDo{
		ID:           param.ID,
		Name:         param.Name,
		Description:  param.Description,
		Category:     param.Category,
		Status:       param.Status,
		SortOrder:    param.SortOrder,
		CreatedBy:    param.CreatedBy,
		DatasourceID: param.DatasourceID,
		CreateTime:   now,
		UpdateTime:   now,
	}

	paramDos := make([]*orm.QuerySceneParamDo, 0, len(param.Params))
	for _, p := range param.Params {
		options := p.Options
		if options == "" {
			options = "null"
		}
		paramDos = append(paramDos, &orm.QuerySceneParamDo{
			KeyName:    p.KeyName,
			Label:      p.Label,
			ParamType:  p.ParamType,
			Required:   &p.Required,
			DefaultVal: p.DefaultVal,
			Options:    options,
			DependsOn:  p.DependsOn,
			SortOrder:  p.SortOrder,
		})
	}

	widgetDos := make([]*orm.QuerySceneWidgetDo, 0, len(param.Widgets))
	for _, w := range param.Widgets {
		resultConfig := w.ResultConfig
		if resultConfig == "" {
			resultConfig = "null"
		}
		widgetDos = append(widgetDos, &orm.QuerySceneWidgetDo{
			Title:        w.Title,
			SQLTemplate:  w.SQLTemplate,
			DisplayType:  w.DisplayType,
			ResultConfig: resultConfig,
			MaxRows:      w.MaxRows,
			TimeoutSec:   w.TimeoutSec,
			SortOrder:    w.SortOrder,
		})
	}

	sceneID, err := uc.repo.SaveScene(ctx, sceneDo, paramDos, widgetDos)
	return sceneID, err
}

func (uc *QuerySceneUseCase) DeleteScene(ctx context.Context, sceneID uint64) error {
	return uc.repo.DeleteScene(ctx, sceneID)
}

// UpdateScene 部分更新场景，nil 字段不更新
func (uc *QuerySceneUseCase) UpdateScene(ctx context.Context, param *UpdateSceneParam) error {
	return uc.repo.UpdateScene(ctx, param)
}

func (uc *QuerySceneUseCase) ExecuteScene(ctx context.Context, sceneID uint64, userParams map[string]string) ([]*WidgetQueryResult, error) {
	scene, err := uc.repo.GetScene(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	widgets, err := uc.repo.GetSceneWidgets(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	results := make([]*WidgetQueryResult, 0, len(widgets))
	for _, w := range widgets {
		r := &WidgetQueryResult{
			WidgetID:     w.ID,
			Title:        w.Title,
			DisplayType:  w.DisplayType,
			ResultConfig: w.ResultConfig,
		}
		data, err := uc.repo.ExecuteWidget(ctx, w, userParams, scene.DatasourceID)
		if err != nil {
			log.Errorf("ExecuteWidget widget_id=%d error: %v", w.ID, err)
			r.Error = err.Error()
		} else {
			r.Data = data
		}
		results = append(results, r)
	}
	return results, nil
}

func (uc *QuerySceneUseCase) PreviewWidget(ctx context.Context, sceneID uint64, widgetParam *QuerySceneWidgetItem, userParams map[string]string) (*QueryResult, error) {
	if widgetParam.TimeoutSec <= 0 {
		widgetParam.TimeoutSec = 30
	}
	if widgetParam.MaxRows <= 0 {
		widgetParam.MaxRows = 1000
	}

	widgets, err := uc.repo.GetSceneWidgets(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	var targetWidget *orm.QuerySceneWidgetDo
	if widgetParam.ID > 0 {
		for _, w := range widgets {
			if w.ID == widgetParam.ID {
				targetWidget = w
				break
			}
		}
	}

	// 没找到时（新建中预览），用传入参数临时构建
	if targetWidget == nil {
		targetWidget = &orm.QuerySceneWidgetDo{
			SceneID:      sceneID,
			Title:        widgetParam.Title,
			SQLTemplate:  widgetParam.SQLTemplate,
			DisplayType:  widgetParam.DisplayType,
			ResultConfig: widgetParam.ResultConfig,
			MaxRows:      widgetParam.MaxRows,
			TimeoutSec:   widgetParam.TimeoutSec,
		}
	}

	if targetWidget.SQLTemplate == "" {
		return nil, fmt.Errorf("SQL 模板不能为空")
	}

	// PreviewWidget 暂不绑定数据源（默认走默认 Doris）
	return uc.repo.ExecuteWidget(ctx, targetWidget, userParams, 0)
}

// ==================== 参数 CRU ====================

func (uc *QuerySceneUseCase) ListParams(ctx context.Context, sceneID uint64) ([]*QuerySceneParamItem, error) {
	list, err := uc.repo.GetSceneParams(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	result := make([]*QuerySceneParamItem, 0, len(list))
	for _, p := range list {
		result = append(result, toParamItem(p))
	}
	return result, nil
}

func (uc *QuerySceneUseCase) CreateParam(ctx context.Context, sceneID uint64, item *QuerySceneParamItem) (uint64, error) {
	now := time.Now()
	options := item.Options
	if options == "" {
		options = "null"
	}
	return uc.repo.CreateParam(ctx, &orm.QuerySceneParamDo{
		SceneID:    sceneID,
		KeyName:    item.KeyName,
		Label:      item.Label,
		ParamType:  item.ParamType,
		Required:   &item.Required,
		DefaultVal: item.DefaultVal,
		Options:    options,
		DependsOn:  item.DependsOn,
		SortOrder:  item.SortOrder,
		CreateTime: now,
		UpdateTime: now,
	})
}

func (uc *QuerySceneUseCase) UpdateParam(ctx context.Context, param *UpdateParamParam) error {
	return uc.repo.UpdateParam(ctx, param)
}

// ==================== 组件 CRU ====================

func (uc *QuerySceneUseCase) ListWidgets(ctx context.Context, sceneID uint64) ([]*QuerySceneWidgetItem, error) {
	list, err := uc.repo.GetSceneWidgets(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	result := make([]*QuerySceneWidgetItem, 0, len(list))
	for _, w := range list {
		result = append(result, toWidgetItem(w))
	}
	return result, nil
}

func (uc *QuerySceneUseCase) CreateWidget(ctx context.Context, sceneID uint64, item *QuerySceneWidgetItem) error {
	now := time.Now()
	resultConfig := item.ResultConfig
	if resultConfig == "" {
		resultConfig = "null"
	}
	return uc.repo.CreateWidget(ctx, &orm.QuerySceneWidgetDo{
		SceneID:      sceneID,
		Title:        item.Title,
		SQLTemplate:  item.SQLTemplate,
		DisplayType:  item.DisplayType,
		ResultConfig: resultConfig,
		MaxRows:      item.MaxRows,
		TimeoutSec:   item.TimeoutSec,
		SortOrder:    item.SortOrder,
		CreateTime:   now,
		UpdateTime:   now,
	})
}

func (uc *QuerySceneUseCase) UpdateWidget(ctx context.Context, widgetID uint64, item *QuerySceneWidgetItem) error {
	resultConfig := item.ResultConfig
	if resultConfig == "" {
		resultConfig = "null"
	}
	return uc.repo.UpdateWidget(ctx, &orm.QuerySceneWidgetDo{
		ID:           widgetID,
		Title:        item.Title,
		SQLTemplate:  item.SQLTemplate,
		DisplayType:  item.DisplayType,
		ResultConfig: resultConfig,
		MaxRows:      item.MaxRows,
		TimeoutSec:   item.TimeoutSec,
		SortOrder:    item.SortOrder,
		UpdateTime:   time.Now(),
	})
}

// ==================== 内部转换函数 ====================

func toSceneItem(s *orm.QuerySceneDo) *QuerySceneItem {
	return &QuerySceneItem{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		Category:     s.Category,
		Status:       s.Status,
		SortOrder:    s.SortOrder,
		CreatedBy:    s.CreatedBy,
		CreatedAt:    s.CreateTime.Unix(),
		DatasourceID: s.DatasourceID,
	}
}

func toParamItem(p *orm.QuerySceneParamDo) *QuerySceneParamItem {
	required := int8(0)
	if p.Required != nil {
		required = *p.Required
	}
	return &QuerySceneParamItem{
		ID:         p.ID,
		KeyName:    p.KeyName,
		Label:      p.Label,
		ParamType:  p.ParamType,
		Required:   required,
		DefaultVal: p.DefaultVal,
		Options:    p.Options,
		DependsOn:  p.DependsOn,
		SortOrder:  p.SortOrder,
	}
}

func toWidgetItem(w *orm.QuerySceneWidgetDo) *QuerySceneWidgetItem {
	return &QuerySceneWidgetItem{
		ID:           w.ID,
		Title:        w.Title,
		SQLTemplate:  w.SQLTemplate,
		DisplayType:  w.DisplayType,
		ResultConfig: w.ResultConfig,
		MaxRows:      w.MaxRows,
		TimeoutSec:   w.TimeoutSec,
		SortOrder:    w.SortOrder,
	}
}
