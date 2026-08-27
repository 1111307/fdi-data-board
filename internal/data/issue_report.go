package data

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/data/orm"
)

var _ biz.IssueReportRepo = (*issueReportRepo)(nil)

type issueReportRepo struct {
	*baseRepo
}

func NewIssueReportRepo(data *Data) biz.IssueReportRepo {
	return &issueReportRepo{baseRepo: &baseRepo{data: data}}
}

func (r *issueReportRepo) CreateIssueReport(ctx context.Context, record *biz.IssueReportRecord) error {
	extra := "{}"
	if len(record.Extra) > 0 {
		payload, err := json.Marshal(record.Extra)
		if err != nil {
			return err
		}
		extra = string(payload)
	}

	return r.mysqlDB(ctx).Create(&orm.IssueReportRecordDo{
		ReportID:   record.ReportID,
		App:        record.App,
		Module:     record.Module,
		Env:        record.Env,
		Level:      record.Level,
		Title:      record.Title,
		Content:    record.Content,
		TraceID:    record.TraceID,
		Extra:      extra,
		ClientIP:   record.ClientIP,
		UserAgent:  record.UserAgent,
		OccurredAt: record.OccurredAt,
	}).Error
}

func (r *issueReportRepo) ListIssueReports(ctx context.Context, param *biz.IssueReportQuery) ([]*biz.IssueReportRecord, int64, error) {
	db := applyIssueReportQuery(r.mysqlDB(ctx).Model(&orm.IssueReportRecordDo{}), param)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := param.Page
	if page <= 0 {
		page = 1
	}
	pageSize := param.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var rows []*orm.IssueReportRecordDo
	if err := db.
		Order("occurred_at DESC").
		Order("id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	list := make([]*biz.IssueReportRecord, 0, len(rows))
	for _, row := range rows {
		list = append(list, issueReportDoToDomain(row))
	}
	return list, total, nil
}

func (r *issueReportRepo) SummarizeIssueReports(ctx context.Context, param *biz.IssueReportQuery) (*biz.IssueReportSummary, error) {
	db := applyIssueReportQuery(r.mysqlDB(ctx).Model(&orm.IssueReportRecordDo{}), param)

	summary := &biz.IssueReportSummary{}
	type countRow struct {
		Total            int64      `gorm:"column:total"`
		CriticalCount    int64      `gorm:"column:critical_count"`
		HighCount        int64      `gorm:"column:high_count"`
		MediumCount      int64      `gorm:"column:medium_count"`
		LowCount         int64      `gorm:"column:low_count"`
		AppCount         int64      `gorm:"column:app_count"`
		ModuleCount      int64      `gorm:"column:module_count"`
		LatestOccurredAt *time.Time `gorm:"column:latest_occurred_at"`
	}
	var counts countRow
	if err := db.Select(`
		COUNT(*) AS total,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS critical_count,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS high_count,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS medium_count,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS low_count,
		COUNT(DISTINCT app) AS app_count,
		COUNT(DISTINCT module) AS module_count,
		MAX(occurred_at) AS latest_occurred_at
	`, biz.IssueLevelCritical, biz.IssueLevelHigh, biz.IssueLevelMedium, biz.IssueLevelLow).Scan(&counts).Error; err != nil {
		return nil, err
	}
	summary.Total = counts.Total
	summary.CriticalCount = counts.CriticalCount
	summary.HighCount = counts.HighCount
	summary.MediumCount = counts.MediumCount
	summary.LowCount = counts.LowCount
	summary.AppCount = counts.AppCount
	summary.ModuleCount = counts.ModuleCount
	if counts.LatestOccurredAt != nil {
		summary.LatestOccurredAt = *counts.LatestOccurredAt
	}

	type appRow struct {
		App              string     `gorm:"column:app"`
		Total            int64      `gorm:"column:total"`
		CriticalCount    int64      `gorm:"column:critical_count"`
		HighCount        int64      `gorm:"column:high_count"`
		MediumCount      int64      `gorm:"column:medium_count"`
		LowCount         int64      `gorm:"column:low_count"`
		LatestOccurredAt *time.Time `gorm:"column:latest_occurred_at"`
	}
	var rows []*appRow
	if err := db.Select(`
		app,
		COUNT(*) AS total,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS critical_count,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS high_count,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS medium_count,
		SUM(CASE WHEN level = ? THEN 1 ELSE 0 END) AS low_count,
		MAX(occurred_at) AS latest_occurred_at
	`, biz.IssueLevelCritical, biz.IssueLevelHigh, biz.IssueLevelMedium, biz.IssueLevelLow).
		Group("app").
		Order("total DESC").
		Order("latest_occurred_at DESC").
		Limit(50).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	summary.AppStats = make([]*biz.IssueReportAppSummary, 0, len(rows))
	for _, row := range rows {
		latestOccurredAt := time.Time{}
		if row.LatestOccurredAt != nil {
			latestOccurredAt = *row.LatestOccurredAt
		}
		summary.AppStats = append(summary.AppStats, &biz.IssueReportAppSummary{
			App:              row.App,
			Total:            row.Total,
			CriticalCount:    row.CriticalCount,
			HighCount:        row.HighCount,
			MediumCount:      row.MediumCount,
			LowCount:         row.LowCount,
			LatestOccurredAt: latestOccurredAt,
		})
	}
	return summary, nil
}

func applyIssueReportQuery(db *gorm.DB, param *biz.IssueReportQuery) *gorm.DB {
	queryDB := db
	if param == nil {
		return queryDB
	}
	if param.App != "" {
		queryDB = queryDB.Where("app = ?", param.App)
	}
	if param.Module != "" {
		queryDB = queryDB.Where("module = ?", param.Module)
	}
	if param.Env != "" {
		queryDB = queryDB.Where("env = ?", param.Env)
	}
	if param.Level != "" {
		queryDB = queryDB.Where("level = ?", param.Level)
	}
	if !param.StartTime.IsZero() {
		queryDB = queryDB.Where("occurred_at >= ?", param.StartTime)
	}
	if !param.EndTime.IsZero() {
		queryDB = queryDB.Where("occurred_at <= ?", param.EndTime)
	}
	if param.Keyword != "" {
		keyword := "%" + strings.ReplaceAll(param.Keyword, "%", "\\%") + "%"
		queryDB = queryDB.Where(
			"report_id LIKE ? OR app LIKE ? OR module LIKE ? OR title LIKE ? OR content LIKE ? OR trace_id LIKE ?",
			keyword, keyword, keyword, keyword, keyword, keyword,
		)
	}
	return queryDB
}

func issueReportDoToDomain(row *orm.IssueReportRecordDo) *biz.IssueReportRecord {
	if row == nil {
		return nil
	}
	extra := map[string]string{}
	if strings.TrimSpace(row.Extra) != "" {
		_ = json.Unmarshal([]byte(row.Extra), &extra)
	}
	return &biz.IssueReportRecord{
		ReportID:   row.ReportID,
		App:        row.App,
		Module:     row.Module,
		Env:        row.Env,
		Level:      row.Level,
		Title:      row.Title,
		Content:    row.Content,
		TraceID:    row.TraceID,
		Extra:      extra,
		ClientIP:   row.ClientIP,
		UserAgent:  row.UserAgent,
		OccurredAt: row.OccurredAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
