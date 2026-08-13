package biz

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeIssueReportRepo struct {
	saved   *IssueReportRecord
	records []*IssueReportRecord
}

func (r *fakeIssueReportRepo) CreateIssueReport(ctx context.Context, record *IssueReportRecord) error {
	copyRecord := *record
	r.saved = &copyRecord
	return nil
}

func (r *fakeIssueReportRepo) ListIssueReports(ctx context.Context, param *IssueReportQuery) ([]*IssueReportRecord, int64, error) {
	return r.records, int64(len(r.records)), nil
}

func (r *fakeIssueReportRepo) SummarizeIssueReports(ctx context.Context, param *IssueReportQuery) (*IssueReportSummary, error) {
	return &IssueReportSummary{
		Total:       int64(len(r.records)),
		AppCount:    2,
		ModuleCount: 2,
		AppStats: []*IssueReportAppSummary{
			{App: "fdi_cloud", Total: 2, CriticalCount: 1, HighCount: 1, LatestOccurredAt: r.records[0].OccurredAt},
			{App: "riskreport", Total: 1, HighCount: 1, LatestOccurredAt: r.records[2].OccurredAt},
		},
	}, nil
}

func TestIssueReportUseCaseReportDefaultsAndSaves(t *testing.T) {
	repo := &fakeIssueReportRepo{}
	uc := NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))

	record, err := uc.Report(context.Background(), &IssueReportRequest{
		App:     "fdi_cloud",
		Module:  "pipeline",
		Env:     "dev",
		Title:   "high risk",
		Content: "detail",
		TraceID: "trace-1",
		Extra:   map[string]string{"job_id": "job-1"},
	})
	if err != nil {
		t.Fatalf("Report() error = %v, want nil", err)
	}
	if record.ReportID == "" {
		t.Fatal("Report() ReportID is empty")
	}
	if record.Level != IssueLevelHigh {
		t.Fatalf("Level = %q, want %q", record.Level, IssueLevelHigh)
	}
	if record.OccurredAt.IsZero() {
		t.Fatal("OccurredAt is zero")
	}
	if repo.saved == nil {
		t.Fatal("repo did not save record")
	}
	if repo.saved.ReportID != record.ReportID {
		t.Fatalf("saved ReportID = %q, want %q", repo.saved.ReportID, record.ReportID)
	}
	if repo.saved.Extra["job_id"] != "job-1" {
		t.Fatalf("saved Extra = %#v, want job_id", repo.saved.Extra)
	}
}

func TestIssueReportUseCaseRejectsEmptyTitle(t *testing.T) {
	repo := &fakeIssueReportRepo{}
	uc := NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))

	_, err := uc.Report(context.Background(), &IssueReportRequest{Content: "detail"})
	if err == nil {
		t.Fatal("Report() error = nil, want validation error")
	}
	if repo.saved != nil {
		t.Fatal("repo saved invalid request")
	}
}

func TestIssueReportUseCaseListReturnsPagedIssueReports(t *testing.T) {
	occurredAt := time.Date(2026, 8, 13, 14, 30, 0, 0, time.UTC)
	repo := &fakeIssueReportRepo{
		records: []*IssueReportRecord{
			{
				ReportID:   "report-1",
				App:        "fdi_cloud",
				Module:     "pipeline",
				Env:        "prod",
				Level:      IssueLevelCritical,
				Title:      "fatal panic",
				Content:    "nil pointer",
				TraceID:    "trace-1",
				Extra:      map[string]string{"job_id": "job-1"},
				OccurredAt: occurredAt,
			},
		},
	}
	uc := NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))

	result, err := uc.List(context.Background(), &IssueReportQuery{App: "fdi_cloud", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	if result.Total != 1 || len(result.List) != 1 {
		t.Fatalf("List() total/list = %d/%d, want 1/1", result.Total, len(result.List))
	}
	if result.List[0].App != "fdi_cloud" || result.List[0].TraceID != "trace-1" {
		t.Fatalf("List()[0] = %+v, want fdi_cloud trace-1", result.List[0])
	}
}

func TestIssueReportUseCaseSummaryAggregatesByApp(t *testing.T) {
	occurredAt := time.Date(2026, 8, 13, 14, 30, 0, 0, time.UTC)
	repo := &fakeIssueReportRepo{
		records: []*IssueReportRecord{
			{App: "fdi_cloud", Module: "pipeline", Level: IssueLevelCritical, OccurredAt: occurredAt},
			{App: "fdi_cloud", Module: "exporter", Level: IssueLevelHigh, OccurredAt: occurredAt.Add(-time.Hour)},
			{App: "riskreport", Module: "reporter", Level: IssueLevelHigh, OccurredAt: occurredAt.Add(-2 * time.Hour)},
		},
	}
	uc := NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))

	summary, err := uc.Summary(context.Background(), &IssueReportQuery{Env: "prod"})
	if err != nil {
		t.Fatalf("Summary() error = %v, want nil", err)
	}
	if summary.Total != 3 || summary.AppCount != 2 || summary.ModuleCount != 2 {
		t.Fatalf("Summary counts = total:%d app:%d module:%d, want 3/2/2", summary.Total, summary.AppCount, summary.ModuleCount)
	}
	if len(summary.AppStats) != 2 {
		t.Fatalf("len(AppStats) = %d, want 2", len(summary.AppStats))
	}
	if summary.AppStats[0].App != "fdi_cloud" || summary.AppStats[0].CriticalCount != 1 || summary.AppStats[0].HighCount != 1 {
		t.Fatalf("AppStats[0] = %+v, want fdi_cloud critical=1 high=1", summary.AppStats[0])
	}
}
