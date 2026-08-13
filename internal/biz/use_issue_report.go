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
}

type IssueReportRepo interface {
	CreateIssueReport(ctx context.Context, record *IssueReportRecord) error
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
