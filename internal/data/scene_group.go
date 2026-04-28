package data

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/data/orm"
)

var _ biz.SceneGroupRepo = (*sceneGroupRepo)(nil)

type sceneGroupRepo struct {
	*baseRepo
}

func NewSceneGroupRepo(data *Data) biz.SceneGroupRepo {
	return &sceneGroupRepo{baseRepo: &baseRepo{data: data}}
}

func (r *sceneGroupRepo) List(ctx context.Context) ([]*orm.QuerySceneGroupDo, error) {
	var list []*orm.QuerySceneGroupDo
	err := r.mysqlDB(ctx).Model(&orm.QuerySceneGroupDo{}).
		Where(orm.QuerySceneGroupColumns.DeleteTime + " IS NULL").
		Order(orm.QuerySceneGroupColumns.ID + " ASC").
		Find(&list).Error
	return list, err
}

func (r *sceneGroupRepo) GetByPageKey(ctx context.Context, pageKey string) (*orm.QuerySceneGroupDo, error) {
	var g orm.QuerySceneGroupDo
	err := r.mysqlDB(ctx).
		Where(orm.QuerySceneGroupColumns.PageKey+" = ? AND "+orm.QuerySceneGroupColumns.DeleteTime+" IS NULL AND "+orm.QuerySceneGroupColumns.Status+" = 1", pageKey).
		First(&g).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (r *sceneGroupRepo) Create(ctx context.Context, do *orm.QuerySceneGroupDo) (uint64, error) {
	err := r.mysqlDB(ctx).Create(do).Error
	return do.ID, err
}

func (r *sceneGroupRepo) Update(ctx context.Context, param *biz.UpdateGroupParam) error {
	now := time.Now()
	updateMap := map[string]interface{}{
		orm.QuerySceneGroupColumns.UpdateTime: now,
	}
	if param.Name != nil {
		updateMap[orm.QuerySceneGroupColumns.Name] = *param.Name
	}
	if param.Description != nil {
		updateMap[orm.QuerySceneGroupColumns.Description] = *param.Description
	}
	if param.SourceTable != nil {
		updateMap[orm.QuerySceneGroupColumns.SourceTable] = *param.SourceTable
	}
	if param.DatasourceID != nil {
		updateMap[orm.QuerySceneGroupColumns.DatasourceID] = *param.DatasourceID
	}
	if param.Status != nil {
		updateMap[orm.QuerySceneGroupColumns.Status] = *param.Status
	}
	if param.HasDimUpdate {
		updateMap[orm.QuerySceneGroupColumns.DimensionFields] = param.DimensionFields
	}
	return r.mysqlDB(ctx).Model(&orm.QuerySceneGroupDo{}).
		Where(orm.QuerySceneGroupColumns.ID+" = ? AND "+orm.QuerySceneGroupColumns.DeleteTime+" IS NULL", param.ID).
		Updates(updateMap).Error
}

func (r *sceneGroupRepo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	result := r.mysqlDB(ctx).Model(&orm.QuerySceneGroupDo{}).
		Where(orm.QuerySceneGroupColumns.ID+" = ? AND "+orm.QuerySceneGroupColumns.DeleteTime+" IS NULL", id).
		Update(orm.QuerySceneGroupColumns.DeleteTime, now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("场景集不存在或已被删除")
	}
	return nil
}

func (r *sceneGroupRepo) GetDimensionValues(ctx context.Context, group *orm.QuerySceneGroupDo, fieldName string) ([]string, error) {
	var db *gorm.DB
	if group.DatasourceID == 0 {
		db = r.dorisDB(ctx)
		if db == nil {
			return nil, fmt.Errorf("默认 Doris 连接不可用")
		}
	} else {
		var err error
		db, err = r.data.GetDatasourceDB(ctx, group.DatasourceID)
		if err != nil {
			return nil, err
		}
	}

	sql := fmt.Sprintf("SELECT DISTINCT `%s` FROM `%s` WHERE `%s` IS NOT NULL ORDER BY `%s` LIMIT 500",
		fieldName, group.SourceTable, fieldName, fieldName)

	rows, err := db.WithContext(ctx).Raw(sql).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询维度值失败: %w", err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		values = append(values, v)
	}
	return values, nil
}
