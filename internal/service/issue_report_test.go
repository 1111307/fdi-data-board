package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/internal/biz"
)

type serviceFakeIssueReportRepo struct {
	saved *biz.IssueReportRecord
}

func (r *serviceFakeIssueReportRepo) CreateIssueReport(ctx context.Context, record *biz.IssueReportRecord) error {
	copyRecord := *record
	r.saved = &copyRecord
	return nil
}

func TestIssueReportServiceReportBindsAndReturnsReportID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &serviceFakeIssueReportRepo{}
	uc := biz.NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))
	svc := NewIssueReportService(uc)
	occurredAt := "2026-08-12T16:30:00Z"

	c := newIssueReportTestContext(`{
		"app":"fdi_cloud",
		"module":"pipeline",
		"env":"dev",
		"title":"high risk",
		"content":"detail",
		"trace_id":"trace-1",
		"extra":{"job_id":"job-1"},
		"occurred_at":"` + occurredAt + `"
	}`)

	resp, err := svc.Report(c)
	if err != nil {
		t.Fatalf("Report() error = %v, want nil", err)
	}
	got := resp.(*IssueReportResponse)
	if got.Code != int32(gcode.CodeOK.Code()) {
		t.Fatalf("Code = %d, want %d", got.Code, gcode.CodeOK.Code())
	}
	if got.ReportID == "" {
		t.Fatal("ReportID is empty")
	}
	if repo.saved == nil {
		t.Fatal("repo did not save record")
	}
	if repo.saved.UserAgent != "riskreport-test" {
		t.Fatalf("UserAgent = %q, want riskreport-test", repo.saved.UserAgent)
	}
	parsedAt, err := time.Parse(time.RFC3339Nano, occurredAt)
	if err != nil {
		t.Fatal(err)
	}
	if !repo.saved.OccurredAt.Equal(parsedAt) {
		t.Fatalf("OccurredAt = %s, want %s", repo.saved.OccurredAt, parsedAt)
	}
	if repo.saved.Extra["job_id"] != "job-1" {
		t.Fatalf("Extra = %#v, want job_id", repo.saved.Extra)
	}
}

func TestIssueReportServiceReportRejectsEmptyTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &serviceFakeIssueReportRepo{}
	uc := biz.NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))
	svc := NewIssueReportService(uc)

	resp, err := svc.Report(newIssueReportTestContext(`{"content":"detail"}`))
	if err != nil {
		t.Fatalf("Report() error = %v, want nil", err)
	}
	got := resp.(*IssueReportResponse)
	if got.Code != int32(gcode.CodeInvalidRequest.Code()) {
		t.Fatalf("Code = %d, want %d", got.Code, gcode.CodeInvalidRequest.Code())
	}
	if repo.saved != nil {
		t.Fatal("repo saved invalid request")
	}
}

func newIssueReportTestContext(body string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/issue_report/v1/report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "riskreport-test")
	req.RemoteAddr = "10.0.0.1:12345"
	c.Request = req
	return c
}
