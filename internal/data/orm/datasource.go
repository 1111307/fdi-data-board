package orm

import "time"

type queryDatasourceColumns struct {
	ID           string
	Name         string
	Description  string
	DsType       string
	Host         string
	Port         string
	Username     string
	Password     string
	DatabaseName string
	MaxIdl       string
	MaxOpen      string
	Status       string
	CreateTime   string
	UpdateTime   string
	DeleteTime   string
}

var QueryDatasourceColumns = queryDatasourceColumns{
	ID:           "id",
	Name:         "name",
	Description:  "description",
	DsType:       "ds_type",
	Host:         "host",
	Port:         "port",
	Username:     "username",
	Password:     "password",
	DatabaseName: "database_name",
	MaxIdl:       "max_idl",
	MaxOpen:      "max_open",
	Status:       "status",
	CreateTime:   "create_time",
	UpdateTime:   "update_time",
	DeleteTime:   "delete_time",
}

type QueryDatasourceDo struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name         string     `gorm:"column:name;size:100;not null;comment:数据源名称"`
	Description  string     `gorm:"column:description;size:500;comment:描述"`
	DsType       string     `gorm:"column:ds_type;size:20;not null;comment:doris/mysql"`
	Host         string     `gorm:"column:host;size:200;not null;comment:主机地址"`
	Port         string     `gorm:"column:port;size:10;not null;comment:端口"`
	Username     string     `gorm:"column:username;size:100;not null;comment:用户名"`
	Password     string     `gorm:"column:password;size:200;not null;comment:密码（明文）"`
	DatabaseName string     `gorm:"column:database_name;size:100;not null;comment:库名"`
	MaxIdl       int        `gorm:"column:max_idl;default:10;comment:最小连接数"`
	MaxOpen      int        `gorm:"column:max_open;default:50;comment:最大连接数"`
	Status       int8       `gorm:"column:status;default:1;comment:1=启用 0=禁用"`
	CreateTime   time.Time  `gorm:"column:create_time;not null"`
	UpdateTime   time.Time  `gorm:"column:update_time;not null"`
	DeleteTime   *time.Time `gorm:"column:delete_time;comment:软删除时间"`
}

func (QueryDatasourceDo) TableName() string {
	return "query_datasource"
}
