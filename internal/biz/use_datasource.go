package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/data/orm"
)

// DatasourceRepo 数据源仓储接口
type DatasourceRepo interface {
	List(ctx context.Context, onlyEnabled bool) ([]*orm.QueryDatasourceDo, error)
	Get(ctx context.Context, id uint64) (*orm.QueryDatasourceDo, error)
	Create(ctx context.Context, do *orm.QueryDatasourceDo) (uint64, error)
	Update(ctx context.Context, param *UpdateDatasourceParam) error
	Delete(ctx context.Context, id uint64) error
	GetTables(ctx context.Context, datasourceID uint64) ([]string, error)
	GetColumns(ctx context.Context, datasourceID uint64, tableName string) ([]*ColumnInfo, error)
}

type ColumnInfo struct {
	Field string `json:"field"`
	Type  string `json:"type"`
}

// ==================== DTO ====================

type DatasourceItem struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	DsType       string `json:"ds_type"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	Username     string `json:"username"`
	DatabaseName string `json:"database_name"`
	MaxIdl       int    `json:"max_idl"`
	MaxOpen      int    `json:"max_open"`
	Status       int8   `json:"status"`
	CreateTime   int64  `json:"create_time"`
}

type CreateDatasourceParam struct {
	Name         string
	Description  string
	DsType       string
	Host         string
	Port         string
	Username     string
	Password     string
	DatabaseName string
	MaxIdl       int
	MaxOpen      int
}

type UpdateDatasourceParam struct {
	ID           uint64
	Name         *string
	Description  *string
	Host         *string
	Port         *string
	Username     *string
	Password     *string
	DatabaseName *string
	MaxIdl       *int
	MaxOpen      *int
	Status       *int8
}

// ==================== UseCase ====================

type DatasourceUseCase struct {
	repo DatasourceRepo
}

func NewDatasourceUseCase(repo DatasourceRepo) *DatasourceUseCase {
	return &DatasourceUseCase{repo: repo}
}

func (uc *DatasourceUseCase) List(ctx context.Context, onlyEnabled bool) ([]*DatasourceItem, error) {
	list, err := uc.repo.List(ctx, onlyEnabled)
	if err != nil {
		return nil, err
	}
	result := make([]*DatasourceItem, 0, len(list))
	for _, d := range list {
		result = append(result, toDatasourceItem(d))
	}
	return result, nil
}

func (uc *DatasourceUseCase) Create(ctx context.Context, param *CreateDatasourceParam) (uint64, error) {
	if param.MaxIdl <= 0 {
		param.MaxIdl = 10
	}
	if param.MaxOpen <= 0 {
		param.MaxOpen = 50
	}
	do := &orm.QueryDatasourceDo{
		Name:         param.Name,
		Description:  param.Description,
		DsType:       param.DsType,
		Host:         param.Host,
		Port:         param.Port,
		Username:     param.Username,
		Password:     param.Password,
		DatabaseName: param.DatabaseName,
		MaxIdl:       param.MaxIdl,
		MaxOpen:      param.MaxOpen,
		Status:       1,
	}
	id, err := uc.repo.Create(ctx, do)
	if err != nil {
		log.Errorf("DatasourceUseCase.Create error: %v", err)
	}
	return id, err
}

func (uc *DatasourceUseCase) Update(ctx context.Context, param *UpdateDatasourceParam) error {
	err := uc.repo.Update(ctx, param)
	if err != nil {
		log.Errorf("DatasourceUseCase.Update id=%d error: %v", param.ID, err)
	}
	return err
}

func (uc *DatasourceUseCase) Delete(ctx context.Context, id uint64) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		log.Errorf("DatasourceUseCase.Delete id=%d error: %v", id, err)
	}
	return err
}

func (uc *DatasourceUseCase) GetTables(ctx context.Context, datasourceID uint64) ([]string, error) {
	return uc.repo.GetTables(ctx, datasourceID)
}

func (uc *DatasourceUseCase) GetColumns(ctx context.Context, datasourceID uint64, tableName string) ([]*ColumnInfo, error) {
	return uc.repo.GetColumns(ctx, datasourceID, tableName)
}

func toDatasourceItem(d *orm.QueryDatasourceDo) *DatasourceItem {
	return &DatasourceItem{
		ID:           d.ID,
		Name:         d.Name,
		Description:  d.Description,
		DsType:       d.DsType,
		Host:         d.Host,
		Port:         d.Port,
		Username:     d.Username,
		DatabaseName: d.DatabaseName,
		MaxIdl:       d.MaxIdl,
		MaxOpen:      d.MaxOpen,
		Status:       d.Status,
		CreateTime:   d.CreateTime.Unix(),
	}
}
