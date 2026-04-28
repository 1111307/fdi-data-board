package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/data/orm"
)

// SceneGroupRepo 场景集仓储接口
type SceneGroupRepo interface {
	List(ctx context.Context) ([]*orm.QuerySceneGroupDo, error)
	GetByPageKey(ctx context.Context, pageKey string) (*orm.QuerySceneGroupDo, error)
	Create(ctx context.Context, do *orm.QuerySceneGroupDo) (uint64, error)
	Update(ctx context.Context, param *UpdateGroupParam) error
	Delete(ctx context.Context, id uint64) error
	GetDimensionValues(ctx context.Context, group *orm.QuerySceneGroupDo, fieldName string) ([]string, error)
}

// ==================== DTO ====================

type SceneGroupItem struct {
	ID              uint64   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	PageKey         string   `json:"page_key"`
	DatasourceID    uint64   `json:"datasource_id"`
	SourceTable     string   `json:"source_table"`
	DimensionFields []string `json:"dimension_fields"`
	PartitionField  string   `json:"partition_field"`
	LookbackDays    int      `json:"lookback_days"`
	Status          int8     `json:"status"`
}

type CreateGroupParam struct {
	Name            string
	Description     string
	PageKey         string
	DatasourceID    uint64
	SourceTable     string
	DimensionFields []string
	PartitionField  string
	LookbackDays    int
}

type UpdateGroupParam struct {
	ID              uint64
	Name            *string
	Description     *string
	SourceTable     *string
	DatasourceID    *uint64
	Status          *int8
	PartitionField  *string
	LookbackDays    *int
	DimensionFields string // 序列化后的 JSON string，HasDimUpdate=true 时生效
	HasDimUpdate    bool
}

// ==================== UseCase ====================

type SceneGroupUseCase struct {
	repo SceneGroupRepo
}

func NewSceneGroupUseCase(repo SceneGroupRepo) *SceneGroupUseCase {
	return &SceneGroupUseCase{repo: repo}
}

func (uc *SceneGroupUseCase) List(ctx context.Context) ([]*SceneGroupItem, error) {
	list, err := uc.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*SceneGroupItem, 0, len(list))
	for _, g := range list {
		result = append(result, toGroupItem(g))
	}
	return result, nil
}

func (uc *SceneGroupUseCase) GetByPageKey(ctx context.Context, pageKey string) (*SceneGroupItem, error) {
	g, err := uc.repo.GetByPageKey(ctx, pageKey)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, nil
	}
	return toGroupItem(g), nil
}

func (uc *SceneGroupUseCase) Create(ctx context.Context, param *CreateGroupParam) (uint64, error) {
	fieldsJSON, err := marshalFields(param.DimensionFields)
	if err != nil {
		return 0, err
	}
	lookbackDays := param.LookbackDays
	if lookbackDays <= 0 && param.PartitionField != "" {
		lookbackDays = 1
	}
	now := time.Now()
	do := &orm.QuerySceneGroupDo{
		Name:            param.Name,
		Description:     param.Description,
		PageKey:         param.PageKey,
		DatasourceID:    param.DatasourceID,
		SourceTable:     param.SourceTable,
		DimensionFields: fieldsJSON,
		PartitionField:  param.PartitionField,
		LookbackDays:    lookbackDays,
		Status:          1,
		CreateTime:      now,
		UpdateTime:      now,
	}
	id, err := uc.repo.Create(ctx, do)
	if err != nil {
		log.Errorf("SceneGroupUseCase.Create error: %v", err)
	}
	return id, err
}

func (uc *SceneGroupUseCase) Update(ctx context.Context, param *UpdateGroupParam) error {
	return uc.repo.Update(ctx, param)
}

func (uc *SceneGroupUseCase) Delete(ctx context.Context, id uint64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *SceneGroupUseCase) GetDimensionValues(ctx context.Context, groupID uint64, fieldName string) ([]string, error) {
	groups, err := uc.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var group *orm.QuerySceneGroupDo
	for _, g := range groups {
		if g.ID == groupID {
			group = g
			break
		}
	}
	if group == nil {
		return nil, fmt.Errorf("场景集不存在")
	}
	return uc.repo.GetDimensionValues(ctx, group, fieldName)
}

// ==================== 内部工具 ====================

func marshalFields(fields []string) (string, error) {
	if len(fields) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(fields)
	return string(b), err
}

func unmarshalFields(s string) []string {
	if s == "" || s == "null" {
		return nil
	}
	var fields []string
	_ = json.Unmarshal([]byte(s), &fields)
	return fields
}

func nilIfEmpty(s *string) *string { return s }

func toGroupItem(g *orm.QuerySceneGroupDo) *SceneGroupItem {
	return &SceneGroupItem{
		ID:              g.ID,
		Name:            g.Name,
		Description:     g.Description,
		PageKey:         g.PageKey,
		DatasourceID:    g.DatasourceID,
		SourceTable:     g.SourceTable,
		DimensionFields: unmarshalFields(g.DimensionFields),
		PartitionField:  g.PartitionField,
		LookbackDays:    g.LookbackDays,
		Status:          g.Status,
	}
}
