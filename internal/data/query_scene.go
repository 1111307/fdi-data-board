package data

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/data/orm"
)

var _ biz.QuerySceneRepo = (*querySceneRepo)(nil)

type querySceneRepo struct {
	*baseRepo
}

func NewQuerySceneRepo(data *Data) biz.QuerySceneRepo {
	return &querySceneRepo{
		baseRepo: &baseRepo{data: data},
	}
}

// ==================== 场景 CRUD ====================

func (r *querySceneRepo) ListScenes(ctx context.Context, category string, status int8, isHome int8, groupID uint64) ([]*orm.QuerySceneDo, error) {
	var list []*orm.QuerySceneDo
	db := r.mysqlDB(ctx).Model(&orm.QuerySceneDo{}).
		Where(orm.QuerySceneColumns.DeleteTime + " IS NULL")

	if category != "" {
		db = db.Where(orm.QuerySceneColumns.Category+" = ?", category)
	}
	if status >= 0 {
		db = db.Where(orm.QuerySceneColumns.Status+" = ?", status)
	}
	if isHome >= 0 {
		db = db.Where(orm.QuerySceneColumns.IsHome+" = ?", isHome)
	}
	if groupID > 0 {
		db = db.Where(orm.QuerySceneColumns.GroupID+" = ?", groupID)
	}

	if err := db.Order(orm.QuerySceneColumns.ID + " ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *querySceneRepo) GetScene(ctx context.Context, sceneID uint64) (*orm.QuerySceneDo, error) {
	var scene orm.QuerySceneDo
	err := r.mysqlDB(ctx).
		Where(orm.QuerySceneColumns.ID+" = ? AND "+orm.QuerySceneColumns.DeleteTime+" IS NULL", sceneID).
		First(&scene).Error
	if err != nil {
		return nil, err
	}
	return &scene, nil
}

func (r *querySceneRepo) GetSceneParams(ctx context.Context, sceneID uint64) ([]*orm.QuerySceneParamDo, error) {
	var list []*orm.QuerySceneParamDo
	err := r.mysqlDB(ctx).
		Where(orm.QuerySceneParamColumns.SceneID+" = ?", sceneID).
		Order(orm.QuerySceneParamColumns.SortOrder + " ASC").
		Find(&list).Error
	return list, err
}

func (r *querySceneRepo) GetSceneWidgets(ctx context.Context, sceneID uint64) ([]*orm.QuerySceneWidgetDo, error) {
	var list []*orm.QuerySceneWidgetDo
	err := r.mysqlDB(ctx).
		Where(orm.QuerySceneWidgetColumns.SceneID+" = ?", sceneID).
		Order(orm.QuerySceneWidgetColumns.SortOrder + " ASC").
		Find(&list).Error
	return list, err
}

func (r *querySceneRepo) SaveScene(ctx context.Context, scene *orm.QuerySceneDo, params []*orm.QuerySceneParamDo, widgets []*orm.QuerySceneWidgetDo) (uint64, error) {
	err := r.InTx(ctx, func(ctx context.Context) error {
		now := time.Now()

		if scene.ID == 0 {
			scene.CreateTime = now
			scene.UpdateTime = now
			if err := r.mysqlDB(ctx).Create(scene).Error; err != nil {
				return err
			}
		} else {
			scene.UpdateTime = now
			if err := r.mysqlDB(ctx).Save(scene).Error; err != nil {
				return err
			}
			if err := r.mysqlDB(ctx).
				Where(orm.QuerySceneParamColumns.SceneID+" = ?", scene.ID).
				Delete(&orm.QuerySceneParamDo{}).Error; err != nil {
				return err
			}
			if err := r.mysqlDB(ctx).
				Where(orm.QuerySceneWidgetColumns.SceneID+" = ?", scene.ID).
				Delete(&orm.QuerySceneWidgetDo{}).Error; err != nil {
				return err
			}
		}

		for _, p := range params {
			p.SceneID = scene.ID
			p.CreateTime = now
			p.UpdateTime = now
		}
		if len(params) > 0 {
			if err := r.mysqlDB(ctx).Create(&params).Error; err != nil {
				return err
			}
		}

		for _, w := range widgets {
			w.SceneID = scene.ID
			w.CreateTime = now
			w.UpdateTime = now
		}
		if len(widgets) > 0 {
			if err := r.mysqlDB(ctx).Create(&widgets).Error; err != nil {
				return err
			}
		}

		return nil
	})
	return scene.ID, err
}

func (r *querySceneRepo) CreateParam(ctx context.Context, param *orm.QuerySceneParamDo) (uint64, error) {
	err := r.mysqlDB(ctx).Create(param).Error
	return param.ID, err
}

func (r *querySceneRepo) UpdateParam(ctx context.Context, param *biz.UpdateParamParam) error {
	now := time.Now()
	updateMap := map[string]interface{}{
		orm.QuerySceneParamColumns.UpdateTime: now,
	}
	if param.KeyName != nil {
		updateMap[orm.QuerySceneParamColumns.KeyName] = *param.KeyName
	}
	if param.Label != nil {
		updateMap[orm.QuerySceneParamColumns.Label] = *param.Label
	}
	if param.ParamType != nil {
		updateMap[orm.QuerySceneParamColumns.ParamType] = *param.ParamType
	}
	if param.Required != nil {
		updateMap[orm.QuerySceneParamColumns.Required] = *param.Required
	}
	if param.DefaultVal != nil {
		updateMap[orm.QuerySceneParamColumns.DefaultVal] = *param.DefaultVal
	}
	if param.Options != nil {
		updateMap[orm.QuerySceneParamColumns.Options] = *param.Options
	}
	if param.DependsOn != nil {
		updateMap[orm.QuerySceneParamColumns.DependsOn] = *param.DependsOn
	}
	if param.SortOrder != nil {
		updateMap[orm.QuerySceneParamColumns.SortOrder] = *param.SortOrder
	}
	return r.mysqlDB(ctx).Model(&orm.QuerySceneParamDo{}).
		Where(orm.QuerySceneParamColumns.ID+" = ?", param.ID).
		Updates(updateMap).Error
}

func (r *querySceneRepo) CreateWidget(ctx context.Context, widget *orm.QuerySceneWidgetDo) error {
	return r.mysqlDB(ctx).Create(widget).Error
}

func (r *querySceneRepo) UpdateWidget(ctx context.Context, widget *orm.QuerySceneWidgetDo) error {
	return r.mysqlDB(ctx).Model(widget).
		Where(orm.QuerySceneWidgetColumns.ID+" = ?", widget.ID).
		Updates(map[string]interface{}{
			orm.QuerySceneWidgetColumns.Title:        widget.Title,
			orm.QuerySceneWidgetColumns.SQLTemplate:  widget.SQLTemplate,
			orm.QuerySceneWidgetColumns.DisplayType:  widget.DisplayType,
			orm.QuerySceneWidgetColumns.ResultConfig: widget.ResultConfig,
			orm.QuerySceneWidgetColumns.MaxRows:      widget.MaxRows,
			orm.QuerySceneWidgetColumns.TimeoutSec:   widget.TimeoutSec,
			orm.QuerySceneWidgetColumns.SortOrder:    widget.SortOrder,
			orm.QuerySceneWidgetColumns.UpdateTime:   widget.UpdateTime,
		}).Error
}

func (r *querySceneRepo) UpdateScene(ctx context.Context, param *biz.UpdateSceneParam) error {
	return r.InTx(ctx, func(ctx context.Context) error {
		now := time.Now()

		updateMap := map[string]interface{}{
			orm.QuerySceneColumns.UpdateTime: now,
		}
		if param.Name != nil {
			updateMap[orm.QuerySceneColumns.Name] = *param.Name
		}
		if param.Description != nil {
			updateMap[orm.QuerySceneColumns.Description] = *param.Description
		}
		if param.Category != nil {
			updateMap[orm.QuerySceneColumns.Category] = *param.Category
		}
		if param.Status != nil {
			updateMap[orm.QuerySceneColumns.Status] = *param.Status
		}
		if param.SortOrder != nil {
			updateMap[orm.QuerySceneColumns.SortOrder] = *param.SortOrder
		}
		if param.DatasourceID != nil {
			updateMap[orm.QuerySceneColumns.DatasourceID] = *param.DatasourceID
		}
		if param.IsHome != nil {
			updateMap[orm.QuerySceneColumns.IsHome] = *param.IsHome
		}
		if param.GroupID != nil {
			updateMap[orm.QuerySceneColumns.GroupID] = *param.GroupID
		}

		if err := r.mysqlDB(ctx).Model(&orm.QuerySceneDo{}).
			Where(orm.QuerySceneColumns.ID+" = ? AND "+orm.QuerySceneColumns.DeleteTime+" IS NULL", param.ID).
			Updates(updateMap).Error; err != nil {
			return err
		}

		if param.HasParamsUpdate {
			if err := r.mysqlDB(ctx).
				Where(orm.QuerySceneParamColumns.SceneID+" = ?", param.ID).
				Delete(&orm.QuerySceneParamDo{}).Error; err != nil {
				return err
			}
			paramDos := make([]*orm.QuerySceneParamDo, 0, len(param.Params))
			for _, p := range param.Params {
				options := p.Options
				if options == "" {
					options = "null"
				}
				paramDos = append(paramDos, &orm.QuerySceneParamDo{
					SceneID:    param.ID,
					KeyName:    p.KeyName,
					Label:      p.Label,
					ParamType:  p.ParamType,
					Required:   &p.Required,
					DefaultVal: p.DefaultVal,
					Options:    options,
					DependsOn:  p.DependsOn,
					SortOrder:  p.SortOrder,
					CreateTime: now,
					UpdateTime: now,
				})
			}
			if len(paramDos) > 0 {
				if err := r.mysqlDB(ctx).Create(&paramDos).Error; err != nil {
					return err
				}
			}
		}

		if param.HasWidgetsUpdate {
			if err := r.mysqlDB(ctx).
				Where(orm.QuerySceneWidgetColumns.SceneID+" = ?", param.ID).
				Delete(&orm.QuerySceneWidgetDo{}).Error; err != nil {
				return err
			}
			widgetDos := make([]*orm.QuerySceneWidgetDo, 0, len(param.Widgets))
			for _, w := range param.Widgets {
				resultConfig := w.ResultConfig
				if resultConfig == "" {
					resultConfig = "null"
				}
				widgetDos = append(widgetDos, &orm.QuerySceneWidgetDo{
					SceneID:      param.ID,
					Title:        w.Title,
					SQLTemplate:  w.SQLTemplate,
					DisplayType:  w.DisplayType,
					ResultConfig: resultConfig,
					MaxRows:      w.MaxRows,
					TimeoutSec:   w.TimeoutSec,
					SortOrder:    w.SortOrder,
					CreateTime:   now,
					UpdateTime:   now,
				})
			}
			if len(widgetDos) > 0 {
				if err := r.mysqlDB(ctx).Create(&widgetDos).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (r *querySceneRepo) DeleteScene(ctx context.Context, sceneID uint64) error {
	return r.InTx(ctx, func(ctx context.Context) error {
		now := time.Now()

		result := r.mysqlDB(ctx).Model(&orm.QuerySceneDo{}).
			Where(orm.QuerySceneColumns.ID+" = ? AND "+orm.QuerySceneColumns.DeleteTime+" IS NULL", sceneID).
			Update(orm.QuerySceneColumns.DeleteTime, now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("场景不存在或已被删除")
		}

		if err := r.mysqlDB(ctx).
			Where(orm.QuerySceneParamColumns.SceneID+" = ?", sceneID).
			Delete(&orm.QuerySceneParamDo{}).Error; err != nil {
			return err
		}

		if err := r.mysqlDB(ctx).
			Where(orm.QuerySceneWidgetColumns.SceneID+" = ?", sceneID).
			Delete(&orm.QuerySceneWidgetDo{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// ==================== 查询执行 ====================

func (r *querySceneRepo) ExecuteWidget(ctx context.Context, widget *orm.QuerySceneWidgetDo, userParams map[string]string, datasourceID uint64) (*biz.QueryResult, error) {
	sql, args, err := bindParams(widget.SQLTemplate, userParams)
	if err != nil {
		return nil, fmt.Errorf("参数绑定失败: %w", err)
	}

	// 去掉末尾分号，避免子查询语法错误
	sql = strings.TrimRight(strings.TrimSpace(sql), ";")

	// 只有 SELECT 语句才做行数限制，其他语句（SHOW/DESC 等）直接执行
	// 已有 LIMIT 则不追加，避免子查询包裹导致优化器无法下推
	if strings.HasPrefix(strings.ToUpper(sql), "SELECT") {
		hasLimit := regexp.MustCompile(`(?i)\bLIMIT\b\s+\d+`).MatchString(sql)
		if !hasLimit {
			sql = fmt.Sprintf("%s LIMIT %d", sql, widget.MaxRows)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(widget.TimeoutSec)*time.Second)
	defer cancel()

	var db *gorm.DB
	if datasourceID == 0 {
		db = r.dorisDB(timeoutCtx)
		if db == nil {
			return nil, fmt.Errorf("默认 Doris 连接不可用")
		}
	} else {
		db, err = r.data.GetDatasourceDB(timeoutCtx, datasourceID)
		if err != nil {
			return nil, err
		}
	}

	rows, err := db.Raw(sql, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %w", err)
	}

	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Errorf("scan row error: %v", err)
			continue
		}
		row := make([]interface{}, len(columns))
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	return &biz.QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Total:   len(resultRows),
	}, nil
}

// bindParams 参数绑定：
//   - {{!key}} 直接替换为参数值（用于表名、列名等标识符）
//   - {{key}}  替换为 ? 占位符（用于普通值，走预编译防注入）
func bindParams(sqlTemplate string, userParams map[string]string) (string, []interface{}, error) {
	// 第一步：处理 {{!key}} → 直接字符串替换（标识符）
	reIdent := regexp.MustCompile(`\{\{!(\w+)\}\}`)
	sql := reIdent.ReplaceAllStringFunc(sqlTemplate, func(match string) string {
		key := reIdent.FindStringSubmatch(match)[1]
		val, ok := userParams[key]
		if !ok {
			return match
		}
		return val
	})

	// 第二步：处理 {{key}} → 纯整数直接内联，其他值走 ? 占位符防注入
	reVal := regexp.MustCompile(`\{\{(\w+)\}\}`)
	reInt := regexp.MustCompile(`^-?\d+$`)
	var args []interface{}
	sql = reVal.ReplaceAllStringFunc(sql, func(match string) string {
		key := reVal.FindStringSubmatch(match)[1]
		val, ok := userParams[key]
		if !ok {
			return match
		}
		if reInt.MatchString(val) {
			return val
		}
		args = append(args, val)
		return "?"
	})

	if strings.Contains(sql, "{{") {
		return "", nil, fmt.Errorf("存在未提供的参数")
	}

	return sql, args, nil
}
