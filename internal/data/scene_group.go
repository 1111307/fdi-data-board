package data

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
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

// invalidateDimCache 清除该场景集所有维度字段的缓存
func (r *sceneGroupRepo) invalidateDimCache(groupID uint64) {
	prefix := fmt.Sprintf("%d:", groupID)
	r.data.dimCache.Range(func(k, _ interface{}) bool {
		if key, ok := k.(string); ok && len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			r.data.dimCache.Delete(k)
		}
		return true
	})
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
	if param.PartitionField != nil {
		updateMap[orm.QuerySceneGroupColumns.PartitionField] = *param.PartitionField
	}
	if param.LookbackDays != nil {
		updateMap[orm.QuerySceneGroupColumns.LookbackDays] = *param.LookbackDays
	}
	if param.HasDimUpdate {
		updateMap[orm.QuerySceneGroupColumns.DimensionFields] = param.DimensionFields
	}
	if err := r.mysqlDB(ctx).Model(&orm.QuerySceneGroupDo{}).
		Where(orm.QuerySceneGroupColumns.ID+" = ? AND "+orm.QuerySceneGroupColumns.DeleteTime+" IS NULL", param.ID).
		Updates(updateMap).Error; err != nil {
		return err
	}
	r.invalidateDimCache(param.ID)
	return nil
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
	cacheKey := fmt.Sprintf("%d:%s", group.ID, fieldName)
	if v, ok := r.data.dimCache.Load(cacheKey); ok {
		entry := v.(dimCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.values, nil
		}
	}

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

	whereClause := fmt.Sprintf("`%s` IS NOT NULL", fieldName)
	if group.PartitionField != "" && group.LookbackDays > 0 {
		whereClause += fmt.Sprintf(" AND `%s` >= DATE_SUB(CURDATE(), INTERVAL %d DAY)",
			group.PartitionField, group.LookbackDays)
	}

	sql := fmt.Sprintf("SELECT DISTINCT `%s` FROM `%s` WHERE %s ORDER BY `%s` LIMIT 500",
		fieldName, group.SourceTable, whereClause, fieldName)

	queryStart := time.Now()
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

	r.data.dimCache.Store(cacheKey, dimCacheEntry{
		values:    values,
		expiresAt: time.Now().Add(10 * time.Minute),
	})
	log.Infof("dim cache miss: group=%d field=%s rows=%d cost=%s sql=%s",
		group.ID, fieldName, len(values), time.Since(queryStart), sql)
	return values, nil
}
