package data

import (
	"context"
	"encoding/json"

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
