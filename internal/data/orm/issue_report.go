package orm

import "time"

type IssueReportRecordDo struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	ReportID   string    `gorm:"column:report_id;size:64;not null;uniqueIndex;comment:上报记录ID"`
	App        string    `gorm:"column:app;size:64;index;comment:上报应用"`
	Module     string    `gorm:"column:module;size:128;index;comment:上报模块"`
	Env        string    `gorm:"column:env;size:32;index;comment:环境"`
	Level      string    `gorm:"column:level;size:32;not null;index;comment:风险等级"`
	Title      string    `gorm:"column:title;size:255;not null;comment:问题标题"`
	Content    string    `gorm:"column:content;type:text;comment:问题内容"`
	TraceID    string    `gorm:"column:trace_id;size:128;index;comment:链路追踪ID"`
	Extra      string    `gorm:"column:extra;type:text;comment:扩展信息JSON"`
	ClientIP   string    `gorm:"column:client_ip;size:64;comment:客户端IP"`
	UserAgent  string    `gorm:"column:user_agent;size:255;comment:User-Agent"`
	OccurredAt time.Time `gorm:"column:occurred_at;index;comment:发生时间"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (IssueReportRecordDo) TableName() string { return "issue_report_record" }
