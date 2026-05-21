package orm

import "time"

type EtlJobLogDo struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Dt        string    `gorm:"column:dt;size:16;not null;uniqueIndex:uk_dt_table;comment:统计日期"`
	Target    string    `gorm:"column:table_name;size:64;not null;uniqueIndex:uk_dt_table;comment:目标表"`
	Status    string    `gorm:"column:status;size:16;not null;comment:pending/running/success/failed"`
	RunType   string    `gorm:"column:run_type;size:16;default:'';comment:触发类型"`
	Cnt       int64     `gorm:"column:cnt;default:0;comment:写入行数"`
	CostMs    int64     `gorm:"column:cost_ms;default:0;comment:耗时毫秒"`
	ErrorMsg  string    `gorm:"column:error_msg;size:512;comment:错误信息"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (EtlJobLogDo) TableName() string { return "etl_job_log" }
