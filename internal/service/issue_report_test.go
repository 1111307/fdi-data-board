package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/internal/biz"
)

type serviceFakeIssueReportRepo struct {
	saved   *biz.IssueReportRecord
	records []*biz.IssueReportRecord
	summary *biz.IssueReportSummary
}

func (r *serviceFakeIssueReportRepo) CreateIssueReport(ctx context.Context, record *biz.IssueReportRecord) error {
	copyRecord := *record
	r.saved = &copyRecord
	return nil
}

func (r *serviceFakeIssueReportRepo) ListIssueReports(ctx context.Context, param *biz.IssueReportQuery) ([]*biz.IssueReportRecord, int64, error) {
	return r.records, int64(len(r.records)), nil
}

func (r *serviceFakeIssueReportRepo) SummarizeIssueReports(ctx context.Context, param *biz.IssueReportQuery) (*biz.IssueReportSummary, error) {
	return r.summary, nil
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

func TestIssueReportServiceListReturnsIssueReports(t *testing.T) {
	gin.SetMode(gin.TestMode)
	occurredAt := time.Date(2026, 8, 13, 14, 30, 0, 0, time.UTC)
	repo := &serviceFakeIssueReportRepo{
		records: []*biz.IssueReportRecord{
			{
				ReportID:   "report-1",
				App:        "fdi_cloud",
				Module:     "pipeline",
				Env:        "prod",
				Level:      biz.IssueLevelCritical,
				Title:      "fatal panic",
				Content:    "nil pointer",
				TraceID:    "trace-1",
				Extra:      map[string]string{"job_id": "job-1"},
				OccurredAt: occurredAt,
			},
		},
	}
	uc := biz.NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))
	svc := NewIssueReportService(uc)

	resp, err := svc.List(newIssueReportQueryTestContext("/issue_report/v1/list?app=fdi_cloud&page=1&page_size=20"))
	if err != nil {
		t.Fatalf("List() error = %v, want nil", err)
	}
	got := resp.(*IssueReportListResponse)
	if got.Code != int32(gcode.CodeOK.Code()) {
		t.Fatalf("Code = %d, want %d", got.Code, gcode.CodeOK.Code())
	}
	if got.Total != 1 || len(got.List) != 1 {
		t.Fatalf("total/list = %d/%d, want 1/1", got.Total, len(got.List))
	}
	if got.List[0].App != "fdi_cloud" || got.List[0].TraceID != "trace-1" {
		t.Fatalf("List[0] = %+v, want fdi_cloud trace-1", got.List[0])
	}
}

func TestIssueReportServiceSummaryReturnsAppStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	occurredAt := time.Date(2026, 8, 13, 14, 30, 0, 0, time.UTC)
	repo := &serviceFakeIssueReportRepo{
		summary: &biz.IssueReportSummary{
			Total:            3,
			CriticalCount:    1,
			HighCount:        2,
			AppCount:         2,
			ModuleCount:      3,
			LatestOccurredAt: occurredAt,
			AppStats: []*biz.IssueReportAppSummary{
				{App: "fdi_cloud", Total: 2, CriticalCount: 1, HighCount: 1, LatestOccurredAt: occurredAt},
			},
		},
	}
	uc := biz.NewIssueReportUseCase(repo, log.NewStdLogger(io.Discard))
	svc := NewIssueReportService(uc)

	resp, err := svc.Summary(newIssueReportQueryTestContext("/issue_report/v1/summary?env=prod"))
	if err != nil {
		t.Fatalf("Summary() error = %v, want nil", err)
	}
	got := resp.(*IssueReportSummaryResponse)
	if got.Code != int32(gcode.CodeOK.Code()) {
		t.Fatalf("Code = %d, want %d", got.Code, gcode.CodeOK.Code())
	}
	if got.Data.Total != 3 || got.Data.AppCount != 2 || len(got.Data.AppStats) != 1 {
		t.Fatalf("summary = %+v, want total=3 app_count=2 one app stat", got.Data)
	}
	if got.Data.AppStats[0].App != "fdi_cloud" {
		t.Fatalf("AppStats[0].App = %q, want fdi_cloud", got.Data.AppStats[0].App)
	}
}

func TestIssueReportSwaggerResponsesDoNotExposeBizTypes(t *testing.T) {
	listField, ok := reflect.TypeOf(IssueReportListResponse{}).FieldByName("List")
	if !ok {
		t.Fatal("IssueReportListResponse missing List field")
	}
	if strings.Contains(listField.Type.String(), "biz.") {
		t.Fatalf("IssueReportListResponse.List type = %s, should use a service DTO so swag can parse it", listField.Type)
	}

	dataField, ok := reflect.TypeOf(IssueReportSummaryResponse{}).FieldByName("Data")
	if !ok {
		t.Fatal("IssueReportSummaryResponse missing Data field")
	}
	if strings.Contains(dataField.Type.String(), "biz.") {
		t.Fatalf("IssueReportSummaryResponse.Data type = %s, should use a service DTO so swag can parse it", dataField.Type)
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

func newIssueReportQueryTestContext(target string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("User-Agent", "riskreport-test")
	req.RemoteAddr = "10.0.0.1:12345"
	c.Request = req
	return c
}
