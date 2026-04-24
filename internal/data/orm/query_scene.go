package orm

import "time"

type querySceneColumns struct {
	ID           string
	Name         string
	Description  string
	Category     string
	Status       string
	SortOrder    string
	CreatedBy    string
	CreateTime   string
	UpdateTime   string
	DeleteTime   string
	DatasourceID string
}

var QuerySceneColumns = querySceneColumns{
	ID:           "id",
	Name:         "name",
	Description:  "description",
	Category:     "category",
	Status:       "status",
	SortOrder:    "sort_order",
	CreatedBy:    "created_by",
	CreateTime:   "create_time",
	UpdateTime:   "update_time",
	DeleteTime:   "delete_time",
	DatasourceID: "datasource_id",
}

type QuerySceneDo struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name         string     `gorm:"column:name;size:100;not null;comment:场景名称"`
	Description  string     `gorm:"column:description;size:500;comment:场景描述"`
	Category     string     `gorm:"column:category;size:50;index:idx_category;comment:分类标签"`
	Status       int8       `gorm:"column:status;default:1;index:idx_status;comment:1=启用 0=禁用"`
	SortOrder    int        `gorm:"column:sort_order;default:0;comment:排序权重"`
	CreatedBy    string     `gorm:"column:created_by;size:100;comment:创建人"`
	DatasourceID uint64     `gorm:"column:datasource_id;default:0;comment:关联数据源ID，0=默认Doris"`
	CreateTime   time.Time  `gorm:"column:create_time;not null;comment:创建时间"`
	UpdateTime   time.Time  `gorm:"column:update_time;not null;comment:更新时间"`
	DeleteTime   *time.Time `gorm:"column:delete_time;comment:软删除时间"`
}

func (QuerySceneDo) TableName() string {
	return "query_scene"
}

type querySceneParamColumns struct {
	ID         string
	SceneID    string
	KeyName    string
	Label      string
	ParamType  string
	Required   string
	DefaultVal string
	Options    string
	DependsOn  string
	SortOrder  string
	CreateTime string
	UpdateTime string
}

var QuerySceneParamColumns = querySceneParamColumns{
	ID:         "id",
	SceneID:    "scene_id",
	KeyName:    "key_name",
	Label:      "label",
	ParamType:  "param_type",
	Required:   "required",
	DefaultVal: "default_val",
	Options:    "options",
	DependsOn:  "depends_on",
	SortOrder:  "sort_order",
	CreateTime: "create_time",
	UpdateTime: "update_time",
}

type QuerySceneParamDo struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	SceneID    uint64    `gorm:"column:scene_id;not null;index:idx_scene_id;comment:所属场景ID"`
	KeyName    string    `gorm:"column:key_name;size:50;not null;comment:参数key"`
	Label      string    `gorm:"column:label;size:100;not null;comment:用户可见标签"`
	ParamType  string    `gorm:"column:param_type;size:20;not null;comment:text/number/select/date/date_range"`
	Required   int8      `gorm:"column:required;default:1;comment:是否必填"`
	DefaultVal string    `gorm:"column:default_val;size:500;comment:默认值"`
	Options    string    `gorm:"column:options;type:json;comment:select类型选项 [{label,value}]"`
	DependsOn  string    `gorm:"column:depends_on;size:50;comment:级联依赖的父参数key"`
	SortOrder  int       `gorm:"column:sort_order;default:0;comment:排序"`
	CreateTime time.Time `gorm:"column:create_time;not null;comment:创建时间"`
	UpdateTime time.Time `gorm:"column:update_time;not null;comment:更新时间"`
}

func (QuerySceneParamDo) TableName() string {
	return "query_scene_param"
}

type querySceneWidgetColumns struct {
	ID           string
	SceneID      string
	Title        string
	SQLTemplate  string
	DisplayType  string
	ResultConfig string
	MaxRows      string
	TimeoutSec   string
	SortOrder    string
	CreateTime   string
	UpdateTime   string
}

var QuerySceneWidgetColumns = querySceneWidgetColumns{
	ID:           "id",
	SceneID:      "scene_id",
	Title:        "title",
	SQLTemplate:  "sql_template",
	DisplayType:  "display_type",
	ResultConfig: "result_config",
	MaxRows:      "max_rows",
	TimeoutSec:   "timeout_sec",
	SortOrder:    "sort_order",
	CreateTime:   "create_time",
	UpdateTime:   "update_time",
}

type QuerySceneWidgetDo struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	SceneID      uint64    `gorm:"column:scene_id;not null;index:idx_scene_id;comment:所属场景ID"`
	Title        string    `gorm:"column:title;size:100;not null;comment:组件标题"`
	SQLTemplate  string    `gorm:"column:sql_template;type:text;not null;comment:带{{变量}}的SQL模板"`
	DisplayType  string    `gorm:"column:display_type;size:20;not null;comment:table/line_chart/bar_chart/pie_chart/number"`
	ResultConfig string    `gorm:"column:result_config;type:json;comment:展示配置"`
	MaxRows      int       `gorm:"column:max_rows;default:1000;comment:最大返回行数"`
	TimeoutSec   int       `gorm:"column:timeout_sec;default:30;comment:查询超时秒数"`
	SortOrder    int       `gorm:"column:sort_order;default:0;comment:排序"`
	CreateTime   time.Time `gorm:"column:create_time;not null;comment:创建时间"`
	UpdateTime   time.Time `gorm:"column:update_time;not null;comment:更新时间"`
}

func (QuerySceneWidgetDo) TableName() string {
	return "query_scene_widget"
}
