package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/data/orm"
)

var errDatasourceNotFound = errors.New("数据源不存在或已被删除")

var _ biz.DatasourceRepo = (*datasourceRepo)(nil)

type datasourceRepo struct {
	*baseRepo
}

func NewDatasourceRepo(data *Data) biz.DatasourceRepo {
	return &datasourceRepo{baseRepo: &baseRepo{data: data}}
}

func (r *datasourceRepo) List(ctx context.Context, onlyEnabled bool) ([]*orm.QueryDatasourceDo, error) {
	var list []*orm.QueryDatasourceDo
	db := r.mysqlDB(ctx).Model(&orm.QueryDatasourceDo{}).
		Where(orm.QueryDatasourceColumns.DeleteTime + " IS NULL")
	if onlyEnabled {
		db = db.Where(orm.QueryDatasourceColumns.Status+" = ?", 1)
	}
	err := db.Order(orm.QueryDatasourceColumns.ID + " ASC").Find(&list).Error
	return list, err
}

func (r *datasourceRepo) Get(ctx context.Context, id uint64) (*orm.QueryDatasourceDo, error) {
	var ds orm.QueryDatasourceDo
	err := r.mysqlDB(ctx).
		Where(orm.QueryDatasourceColumns.ID+" = ? AND "+orm.QueryDatasourceColumns.DeleteTime+" IS NULL", id).
		First(&ds).Error
	return &ds, err
}

func (r *datasourceRepo) Create(ctx context.Context, do *orm.QueryDatasourceDo) (uint64, error) {
	now := time.Now()
	do.CreateTime = now
	do.UpdateTime = now
	err := r.mysqlDB(ctx).Create(do).Error
	return do.ID, err
}

func (r *datasourceRepo) Update(ctx context.Context, param *biz.UpdateDatasourceParam) error {
	now := time.Now()
	updateMap := map[string]interface{}{
		orm.QueryDatasourceColumns.UpdateTime: now,
	}
	if param.Name != nil {
		updateMap[orm.QueryDatasourceColumns.Name] = *param.Name
	}
	if param.Description != nil {
		updateMap[orm.QueryDatasourceColumns.Description] = *param.Description
	}
	if param.Host != nil {
		updateMap[orm.QueryDatasourceColumns.Host] = *param.Host
	}
	if param.Port != nil {
		updateMap[orm.QueryDatasourceColumns.Port] = *param.Port
	}
	if param.Username != nil {
		updateMap[orm.QueryDatasourceColumns.Username] = *param.Username
	}
	if param.Password != nil {
		updateMap[orm.QueryDatasourceColumns.Password] = *param.Password
	}
	if param.DatabaseName != nil {
		updateMap[orm.QueryDatasourceColumns.DatabaseName] = *param.DatabaseName
	}
	if param.MaxIdl != nil {
		updateMap[orm.QueryDatasourceColumns.MaxIdl] = *param.MaxIdl
	}
	if param.MaxOpen != nil {
		updateMap[orm.QueryDatasourceColumns.MaxOpen] = *param.MaxOpen
	}
	if param.Status != nil {
		updateMap[orm.QueryDatasourceColumns.Status] = *param.Status
	}

	if err := r.mysqlDB(ctx).Model(&orm.QueryDatasourceDo{}).
		Where(orm.QueryDatasourceColumns.ID+" = ? AND "+orm.QueryDatasourceColumns.DeleteTime+" IS NULL", param.ID).
		Updates(updateMap).Error; err != nil {
		return err
	}
	// 清除连接缓存，下次执行时重新建立
	r.data.InvalidateDatasourceCache(param.ID)
	return nil
}

func (r *datasourceRepo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	result := r.mysqlDB(ctx).Model(&orm.QueryDatasourceDo{}).
		Where(orm.QueryDatasourceColumns.ID+" = ? AND "+orm.QueryDatasourceColumns.DeleteTime+" IS NULL", id).
		Update(orm.QueryDatasourceColumns.DeleteTime, now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errDatasourceNotFound
	}
	r.data.InvalidateDatasourceCache(id)
	return nil
}

// getSchemaDB 根据 datasource_id 获取连接，0 走默认 Doris
func (r *datasourceRepo) getSchemaDB(ctx context.Context, datasourceID uint64) (*gorm.DB, error) {
	if datasourceID == 0 {
		db := r.dorisDB(ctx)
		if db == nil {
			return nil, fmt.Errorf("默认 Doris 连接不可用")
		}
		return db, nil
	}
	return r.data.GetDatasourceDB(ctx, datasourceID)
}

func (r *datasourceRepo) GetTables(ctx context.Context, datasourceID uint64) ([]string, error) {
	db, err := r.getSchemaDB(ctx, datasourceID)
	if err != nil {
		return nil, err
	}

	rows, err := db.WithContext(ctx).Raw("SHOW TABLES").Rows()
	if err != nil {
		return nil, fmt.Errorf("获取表列表失败: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func (r *datasourceRepo) GetColumns(ctx context.Context, datasourceID uint64, tableName string) ([]*biz.ColumnInfo, error) {
	db, err := r.getSchemaDB(ctx, datasourceID)
	if err != nil {
		return nil, err
	}

	type descRow struct {
		Field   string  `gorm:"column:Field"`
		Type    string  `gorm:"column:Type"`
		Null    string  `gorm:"column:Null"`
		Key     string  `gorm:"column:Key"`
		Default *string `gorm:"column:Default"`
		Extra   string  `gorm:"column:Extra"`
	}

	var result []descRow
	if err := db.WithContext(ctx).Raw(fmt.Sprintf("DESCRIBE `%s`", tableName)).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("获取字段列表失败: %w", err)
	}

	columns := make([]*biz.ColumnInfo, 0, len(result))
	for _, row := range result {
		columns = append(columns, &biz.ColumnInfo{
			Field: row.Field,
			Type:  row.Type,
		})
	}
	return columns, nil
}
