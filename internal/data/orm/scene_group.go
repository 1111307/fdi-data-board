package orm

import "time"

type querySceneGroupColumns struct {
	ID              string
	Name            string
	Description     string
	PageKey         string
	DatasourceID    string
	SourceTable     string
	DimensionFields string
	Status          string
	CreateTime      string
	UpdateTime      string
	DeleteTime      string
}

var QuerySceneGroupColumns = querySceneGroupColumns{
	ID:              "id",
	Name:            "name",
	Description:     "description",
	PageKey:         "page_key",
	DatasourceID:    "datasource_id",
	SourceTable:     "table_name",
	DimensionFields: "dimension_fields",
	Status:          "status",
	CreateTime:      "create_time",
	UpdateTime:      "update_time",
	DeleteTime:      "delete_time",
}

type QuerySceneGroupDo struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name            string     `gorm:"column:name;size:100;not null;comment:场景集名称"`
	Description     string     `gorm:"column:description;size:500;comment:描述"`
	PageKey         string     `gorm:"column:page_key;size:20;not null;comment:对应页面key(a/b/c等)"`
	DatasourceID    uint64     `gorm:"column:datasource_id;default:0;comment:维度数据源ID"`
	SourceTable     string     `gorm:"column:table_name;size:200;not null;comment:维度来源表"`
	DimensionFields string     `gorm:"column:dimension_fields;type:json;comment:左侧维度字段列表JSON"`
	Status          int8       `gorm:"column:status;default:1;comment:1=启用 0=禁用"`
	CreateTime      time.Time  `gorm:"column:create_time;not null"`
	UpdateTime      time.Time  `gorm:"column:update_time;not null"`
	DeleteTime      *time.Time `gorm:"column:delete_time"`
}

func (QuerySceneGroupDo) TableName() string {
	return "query_scene_group"
}
