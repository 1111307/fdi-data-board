package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

const (
	IssueLevelCritical = "critical"
	IssueLevelHigh     = "high"
	IssueLevelMedium   = "medium"
	IssueLevelLow      = "low"
)

var (
	ErrIssueReportRequestRequired = errors.New("issue report request is required")
	ErrIssueReportTitleRequired   = errors.New("issue report title is required")
)

type IssueReportRequest struct {
	App        string            `json:"app"`
	Module     string            `json:"module"`
	Env        string            `json:"env"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Level      string            `json:"level"`
	TraceID    string            `json:"trace_id"`
	Extra      map[string]string `json:"extra"`
	ClientIP   string            `json:"client_ip"`
	UserAgent  string            `json:"user_agent"`
	OccurredAt time.Time         `json:"occurred_at"`
}

type IssueReportRecord struct {
	ReportID   string            `json:"report_id"`
	App        string            `json:"app"`
	Module     string            `json:"module"`
	Env        string            `json:"env"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Level      string            `json:"level"`
	TraceID    string            `json:"trace_id"`
	Extra      map[string]string `json:"extra"`
	ClientIP   string            `json:"client_ip"`
	UserAgent  string            `json:"user_agent"`
	OccurredAt time.Time         `json:"occurred_at"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type IssueReportQuery struct {
	App       string
	Module    string
	Env       string
	Level     string
	Keyword   string
	StartTime time.Time
	EndTime   time.Time
	Page      int
	PageSize  int
}

type IssueReportListResult struct {
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	List     []*IssueReportRecord `json:"list"`
}

type IssueReportAppSummary struct {
	App              string    `json:"app"`
	Total            int64     `json:"total"`
	CriticalCount    int64     `json:"critical_count"`
	HighCount        int64     `json:"high_count"`
	MediumCount      int64     `json:"medium_count"`
	LowCount         int64     `json:"low_count"`
	LatestOccurredAt time.Time `json:"latest_occurred_at"`
}

type IssueReportSummary struct {
	Total            int64                    `json:"total"`
	CriticalCount    int64                    `json:"critical_count"`
	HighCount        int64                    `json:"high_count"`
	MediumCount      int64                    `json:"medium_count"`
	LowCount         int64                    `json:"low_count"`
	AppCount         int64                    `json:"app_count"`
	ModuleCount      int64                    `json:"module_count"`
	LatestOccurredAt time.Time                `json:"latest_occurred_at"`
	AppStats         []*IssueReportAppSummary `json:"app_stats"`
}

type IssueReportRepo interface {
	CreateIssueReport(ctx context.Context, record *IssueReportRecord) error
	ListIssueReports(ctx context.Context, param *IssueReportQuery) ([]*IssueReportRecord, int64, error)
	SummarizeIssueReports(ctx context.Context, param *IssueReportQuery) (*IssueReportSummary, error)
}

type IssueReportUseCase struct {
	repo IssueReportRepo
	log  *log.Helper
}

func NewIssueReportUseCase(repo IssueReportRepo, logger log.Logger) *IssueReportUseCase {
	return &IssueReportUseCase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *IssueReportUseCase) Report(ctx context.Context, req *IssueReportRequest) (*IssueReportRecord, error) {
	if req == nil {
		return nil, ErrIssueReportRequestRequired
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrIssueReportTitleRequired
	}

	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	record := &IssueReportRecord{
		ReportID:   uuid.NewString(),
		App:        strings.TrimSpace(req.App),
		Module:     strings.TrimSpace(req.Module),
		Env:        strings.TrimSpace(req.Env),
		Title:      title,
		Content:    req.Content,
		Level:      normalizeIssueLevel(req.Level),
		TraceID:    strings.TrimSpace(req.TraceID),
		Extra:      req.Extra,
		ClientIP:   strings.TrimSpace(req.ClientIP),
		UserAgent:  strings.TrimSpace(req.UserAgent),
		OccurredAt: occurredAt,
	}
	if record.Extra == nil {
		record.Extra = map[string]string{}
	}

	if err := uc.repo.CreateIssueReport(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (uc *IssueReportUseCase) List(ctx context.Context, query *IssueReportQuery) (*IssueReportListResult, error) {
	normalized := normalizeIssueReportQuery(query)
	list, total, err := uc.repo.ListIssueReports(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return &IssueReportListResult{
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
		List:     list,
	}, nil
}

func (uc *IssueReportUseCase) Summary(ctx context.Context, query *IssueReportQuery) (*IssueReportSummary, error) {
	return uc.repo.SummarizeIssueReports(ctx, normalizeIssueReportQuery(query))
}

func normalizeIssueLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case IssueLevelCritical:
		return IssueLevelCritical
	case IssueLevelMedium:
		return IssueLevelMedium
	case IssueLevelLow:
		return IssueLevelLow
	default:
		return IssueLevelHigh
	}
}

func normalizeIssueReportQuery(query *IssueReportQuery) *IssueReportQuery {
	normalized := &IssueReportQuery{}
	if query != nil {
		*normalized = *query
	}
	normalized.App = strings.TrimSpace(normalized.App)
	normalized.Module = strings.TrimSpace(normalized.Module)
	normalized.Env = strings.TrimSpace(normalized.Env)
	normalized.Level = normalizeOptionalIssueLevel(normalized.Level)
	normalized.Keyword = strings.TrimSpace(normalized.Keyword)
	if normalized.Page <= 0 {
		normalized.Page = 1
	}
	if normalized.PageSize <= 0 {
		normalized.PageSize = 20
	}
	if normalized.PageSize > 200 {
		normalized.PageSize = 200
	}
	return normalized
}

func normalizeOptionalIssueLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case IssueLevelCritical, IssueLevelHigh, IssueLevelMedium, IssueLevelLow:
		return level
	default:
		return ""
	}
}
