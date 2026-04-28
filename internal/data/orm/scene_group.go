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
	PartitionField  string
	LookbackDays    string
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
	PartitionField:  "partition_field",
	LookbackDays:    "lookback_days",
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
	PartitionField  string     `gorm:"column:partition_field;size:64;comment:分区列名，如dt，空则不加时间过滤"`
	LookbackDays    int        `gorm:"column:lookback_days;default:1;comment:维度查询往回取多少天，0=不限"`
	Status          int8       `gorm:"column:status;default:1;comment:1=启用 0=禁用"`
	CreateTime      time.Time  `gorm:"column:create_time;not null"`
	UpdateTime      time.Time  `gorm:"column:update_time;not null"`
	DeleteTime      *time.Time `gorm:"column:delete_time"`
}

func (QuerySceneGroupDo) TableName() string {
	return "query_scene_group"
}
