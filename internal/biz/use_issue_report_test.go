package biz

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeIssueReportRepo struct {
	saved *IssueReportRecord
}

func (r *fakeIssueReportRepo) CreateIssueReport(ctx context.Context, record *IssueReportRecord) error {
	copyRecord := *record
	r.saved = &copyRecord
	return nil
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
