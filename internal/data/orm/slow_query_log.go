package orm

import "time"

type SlowQueryLogDo struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	QueryType     string    `gorm:"column:query_type;size:64;not null;comment:查询类型"`
	DurationMs    int64     `gorm:"column:duration_ms;not null;comment:耗时(ms)"`
	StartDt       string    `gorm:"column:start_dt;size:16;comment:查询起始日期"`
	EndDt         string    `gorm:"column:end_dt;size:16;comment:查询结束日期"`
	DateRangeDays int       `gorm:"column:date_range_days;comment:日期跨度(天)"`
	EventNames    string    `gorm:"column:event_names;size:512;comment:事件名(逗号分隔)"`
	FilterName    string    `gorm:"column:filter_name;size:256;comment:筛选器"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime;comment:记录时间"`
}

func (SlowQueryLogDo) TableName() string { return "slow_query_log" }
