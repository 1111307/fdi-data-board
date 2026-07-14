package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

// fakeReconcileRepo 实现 biz.ReconcileRepo，供 service 层单测使用
type fakeReconcileRepo struct {
	overviewData *biz.ReconcileOverviewData
	overviewErr  error

	trendData *biz.ReconcileTrendData
	trendErr  error

	moduleList []*biz.ReconcileModuleItem
	moduleErr  error

	decodeStatusList []*biz.ReconcileDecodeStatusItem
	decodeStatusErr  error

	projectList []*biz.ReconcileProjectItem
	projectErr  error

	md5List    []*biz.ReconcileMd5Item
	md5ListErr error

	md5DetailList []*biz.ReconcileMd5DetailItem
	md5DetailErr  error

	eventListItems []*biz.ReconcileEventItem
	eventListTotal int64
	eventListErr   error

	recordConsistencyItems []*biz.ReconcileRecordConsistencyItem
	recordConsistencyErr   error

	uuidSourceItems []*biz.ReconcileUuidSourceItem
	uuidSourceErr   error

	failureSummaryData *biz.ReconcileFailureSummaryData
	failureSummaryErr  error

	pipelineTreeData *biz.ReconcilePipelineTreeData
	pipelineTreeErr  error
}

func (f *fakeReconcileRepo) GetOverview(ctx context.Context, date string) (*biz.ReconcileOverviewData, error) {
	return f.overviewData, f.overviewErr
}

func (f *fakeReconcileRepo) GetTrend(ctx context.Context, startDt, endDt string) (*biz.ReconcileTrendData, error) {
	return f.trendData, f.trendErr
}

func (f *fakeReconcileRepo) GetModule(ctx context.Context, date, project string) ([]*biz.ReconcileModuleItem, error) {
	return f.moduleList, f.moduleErr
}

func (f *fakeReconcileRepo) GetProject(ctx context.Context, date, orderBy string, limit int) ([]*biz.ReconcileProjectItem, error) {
	return f.projectList, f.projectErr
}

func (f *fakeReconcileRepo) GetDecodeStatus(ctx context.Context, date string) ([]*biz.ReconcileDecodeStatusItem, error) {
	return f.decodeStatusList, f.decodeStatusErr
}

func (f *fakeReconcileRepo) ListDiffMd5(ctx context.Context, date, diffType string, limit int) ([]*biz.ReconcileMd5Item, error) {
	return f.md5List, f.md5ListErr
}

func (f *fakeReconcileRepo) GetMd5Detail(ctx context.Context, date, md5 string) ([]*biz.ReconcileMd5DetailItem, error) {
	return f.md5DetailList, f.md5DetailErr
}

func (f *fakeReconcileRepo) ListEventList(ctx context.Context, date, moduleName, project, eventType string, page, pageSize int) ([]*biz.ReconcileEventItem, int64, error) {
	return f.eventListItems, f.eventListTotal, f.eventListErr
}

func (f *fakeReconcileRepo) GetRecordConsistency(ctx context.Context, date string, limit int) ([]*biz.ReconcileRecordConsistencyItem, error) {
	return f.recordConsistencyItems, f.recordConsistencyErr
}

func (f *fakeReconcileRepo) GetUuidSource(ctx context.Context, date string) ([]*biz.ReconcileUuidSourceItem, error) {
	return f.uuidSourceItems, f.uuidSourceErr
}

func (f *fakeReconcileRepo) GetFailureSummary(ctx context.Context, date string) (*biz.ReconcileFailureSummaryData, error) {
	return f.failureSummaryData, f.failureSummaryErr
}

func (f *fakeReconcileRepo) GetPipelineTree(ctx context.Context, date, project, moduleName, md5 string) (*biz.ReconcilePipelineTreeData, error) {
	return f.pipelineTreeData, f.pipelineTreeErr
}

func newTestGinContext(target string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return c
}

func TestReconcileService_GetOverview(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeReconcileRepo{overviewData: &biz.ReconcileOverviewData{
			Bag:        biz.ReconcileOverviewBag{Total: 100, DecodeSuccess: 90, DecodeFailed: 8, DecodePartial: 2, DecodeSuccessRate: 0.9},
			EventParse: biz.ReconcileOverviewEventParse{Expected: 1000, SendFailed: 5, SendFailedRate: 0.005},
			EventLand:  biz.ReconcileOverviewEventLand{Matched: 80, ConvertFailed: 10, LandingFailed: 5, Missing: 5, MatchRate: 0.8},
		}}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetOverview(newTestGinContext("/dashboard/v1/reconcile/overview?date=2026-07-01"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeOK.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeOK.Code())
		}
		got, ok := resp.(*dashboard_api.ReconcileOverviewResponse)
		if !ok {
			t.Fatalf("unexpected response type %T", resp)
		}
		if got.Date != "2026-07-01" || got.Bag.Total != 100 || got.EventLand.Matched != 80 {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	t.Run("repo error maps to internal error code", func(t *testing.T) {
		repo := &fakeReconcileRepo{overviewErr: errors.New("doris down")}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetOverview(newTestGinContext("/dashboard/v1/reconcile/overview"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeInternalError.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInternalError.Code())
		}
	})
}

func TestReconcileService_GetTrend_InvalidDateRange(t *testing.T) {
	repo := &fakeReconcileRepo{trendData: &biz.ReconcileTrendData{}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetTrend(newTestGinContext("/dashboard/v1/reconcile/trend?start_dt=not-a-date&end_dt=2026-07-03"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetCode() != int32(gcode.CodeInvalidParameter.Code()) {
		t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInvalidParameter.Code())
	}
}

func TestReconcileService_GetTrend_Points(t *testing.T) {
	repo := &fakeReconcileRepo{trendData: &biz.ReconcileTrendData{Points: []*biz.ReconcileTrendPoint{
		{Dt: "2026-07-01", Expected: 100, Matched: 90, ConvertFailed: 5, LandingFailed: 3, Missing: 2, MatchRate: 0.9},
	}}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetTrend(newTestGinContext("/dashboard/v1/reconcile/trend?start_dt=2026-07-01&end_dt=2026-07-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := resp.(*dashboard_api.ReconcileTrendResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if len(got.Points) != 1 || got.Points[0].Dt != "2026-07-01" || got.Points[0].ConvertFailed != 5 {
		t.Errorf("unexpected points: %+v", got.Points)
	}
}

func TestReconcileService_GetModule(t *testing.T) {
	repo := &fakeReconcileRepo{moduleList: []*biz.ReconcileModuleItem{
		{ModuleName: "fff_close", Expected: 10, Matched: 8, ConvertFailed: 1, LandingFailed: 1, Missing: 0},
	}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetModule(newTestGinContext("/dashboard/v1/reconcile/module?date=2026-07-01&project=proj-a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := resp.(*dashboard_api.ReconcileModuleResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if got.Project != "proj-a" || len(got.Modules) != 1 || got.Modules[0].ModuleName != "fff_close" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestReconcileService_GetProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeReconcileRepo{projectList: []*biz.ReconcileProjectItem{
			{Project: "proj-a", Expected: 8000, Matched: 7950, ConvertFailed: 30, LandingFailed: 20, Missing: 0, MatchRate: 0.9938},
		}}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetProject(newTestGinContext("/dashboard/v1/reconcile/project?date=2026-07-01"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := resp.(*dashboard_api.ReconcileProjectResponse)
		if !ok {
			t.Fatalf("unexpected response type %T", resp)
		}
		if len(got.Projects) != 1 || got.Projects[0].Project != "proj-a" {
			t.Errorf("unexpected projects: %+v", got.Projects)
		}
	})

	t.Run("invalid order_by returns CodeInvalidParameter", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetProject(newTestGinContext("/dashboard/v1/reconcile/project?order_by=bogus"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeInvalidParameter.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInvalidParameter.Code())
		}
	})
}

func TestReconcileService_GetDecodeStatus(t *testing.T) {
	repo := &fakeReconcileRepo{decodeStatusList: []*biz.ReconcileDecodeStatusItem{
		{Status: 1, Stage: 0, Count: 95},
	}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetDecodeStatus(newTestGinContext("/dashboard/v1/reconcile/decode_status"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := resp.(*dashboard_api.ReconcileDecodeStatusResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if len(got.Items) != 1 || got.Items[0].Count != 95 {
		t.Errorf("unexpected items: %+v", got.Items)
	}
}

func TestReconcileService_ListMd5(t *testing.T) {
	t.Run("success default type", func(t *testing.T) {
		repo := &fakeReconcileRepo{md5List: []*biz.ReconcileMd5Item{{Md5: "abc", Count: 3}}}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.ListMd5(newTestGinContext("/dashboard/v1/reconcile/md5"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := resp.(*dashboard_api.ReconcileMd5ListResponse)
		if !ok {
			t.Fatalf("unexpected response type %T", resp)
		}
		if got.Type != "missing" || len(got.Items) != 1 {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	t.Run("invalid type returns CodeInvalidParameter", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.ListMd5(newTestGinContext("/dashboard/v1/reconcile/md5?type=failed"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeInvalidParameter.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInvalidParameter.Code())
		}
	})
}

func TestReconcileService_GetMd5Detail(t *testing.T) {
	t.Run("missing md5 returns CodeInvalidParameter", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetMd5Detail(newTestGinContext("/dashboard/v1/reconcile/md5_detail?date=2026-07-01"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeInvalidParameter.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInvalidParameter.Code())
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := &fakeReconcileRepo{md5DetailList: []*biz.ReconcileMd5DetailItem{
			{Uuid: "u1", ModuleName: "fdr_status", Consumed: true},
		}}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetMd5Detail(newTestGinContext("/dashboard/v1/reconcile/md5_detail?date=2026-07-01&md5=abc123"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := resp.(*dashboard_api.ReconcileMd5DetailResponse)
		if !ok {
			t.Fatalf("unexpected response type %T", resp)
		}
		if got.Md5 != "abc123" || len(got.Items) != 1 {
			t.Errorf("unexpected response: %+v", got)
		}
	})
}

func TestReconcileService_ListEventList(t *testing.T) {
	t.Run("success default type", func(t *testing.T) {
		repo := &fakeReconcileRepo{eventListItems: []*biz.ReconcileEventItem{{Md5: "abc", Uuid: "u1"}}, eventListTotal: 1}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.ListEventList(newTestGinContext("/dashboard/v1/reconcile/event_list"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := resp.(*dashboard_api.ReconcileEventListResponse)
		if !ok {
			t.Fatalf("unexpected response type %T", resp)
		}
		if got.Type != "missing" || got.Total != 1 || len(got.Items) != 1 {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	t.Run("invalid type returns CodeInvalidParameter", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.ListEventList(newTestGinContext("/dashboard/v1/reconcile/event_list?type=bogus"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeInvalidParameter.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInvalidParameter.Code())
		}
	})
}

func TestReconcileService_GetRecordConsistency(t *testing.T) {
	repo := &fakeReconcileRepo{recordConsistencyItems: []*biz.ReconcileRecordConsistencyItem{
		{Md5: "abc", ParsedLineCount: 100, DetailCount: 98, Diff: 2},
	}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetRecordConsistency(newTestGinContext("/dashboard/v1/reconcile/record_consistency?date=2026-07-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := resp.(*dashboard_api.ReconcileRecordConsistencyResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if got.Date != "2026-07-01" || len(got.Items) != 1 || got.Items[0].Diff != 2 {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestReconcileService_GetUuidSource(t *testing.T) {
	rate := 0.9
	repo := &fakeReconcileRepo{uuidSourceItems: []*biz.ReconcileUuidSourceItem{
		{UuidSource: "real", Expected: 100, Matched: 90, Missing: 10, MatchRate: &rate},
	}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetUuidSource(newTestGinContext("/dashboard/v1/reconcile/uuid_source"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := resp.(*dashboard_api.ReconcileUuidSourceResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if len(got.Sources) != 1 || got.Sources[0].MatchRate == nil || *got.Sources[0].MatchRate != 0.9 {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestReconcileService_GetFailureSummary(t *testing.T) {
	repo := &fakeReconcileRepo{failureSummaryData: &biz.ReconcileFailureSummaryData{
		DecodeFailed:  []*biz.ReconcileDecodeFailedItem{{Stage: 1, Count: 10, SampleErrorMsg: "unzip failed"}},
		SendFailed:    []*biz.ReconcileSendFailedItem{{ModuleName: "fff_close", ErrDetail: "5:timeout", Count: 3}},
		ConvertFailed: []*biz.ReconcileConvertFailedItem{{ModuleName: "fff_close", ErrDetail: "3:nil trigger", Count: 4}},
		LandingFailed: []*biz.ReconcileLandingFailedItem{{ModuleName: "fff_close", ErrDetail: "6:write failed", Count: 5}},
	}}
	svc := NewReconcileService(biz.NewReconcileUseCase(repo))

	resp, err := svc.GetFailureSummary(newTestGinContext("/dashboard/v1/reconcile/failure_summary?date=2026-07-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := resp.(*dashboard_api.ReconcileFailureSummaryResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", resp)
	}
	if got.Date != "2026-07-01" {
		t.Errorf("got.Date = %q, want 2026-07-01", got.Date)
	}
	if len(got.DecodeFailed) != 1 || len(got.SendFailed) != 1 || len(got.ConvertFailed) != 1 || len(got.LandingFailed) != 1 {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestReconcileService_GetPipelineTree(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeReconcileRepo{pipelineTreeData: &biz.ReconcilePipelineTreeData{
			Bag: biz.ReconcilePipelineBagData{
				Total:           100,
				ParseSuccess:    95,
				DecodeSuccess:   90,
				DecodePartial:   5,
				DecodeFailed:    5,
				StatusLineCount: 100,
				SkipLineCount:   2,
				ParsedLineCount: 200,
			},
			EventParse: biz.ReconcilePipelineEventParseData{ParseSuccess: 190, ParseFailed: 10},
			EventLand: biz.ReconcilePipelineEventLandData{
				Matched:       180,
				ConvertFailed: 5,
				LandingFailed: 3,
				Missing:       2,
			},
		}}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetPipelineTree(newTestGinContext("/dashboard/v1/reconcile/pipeline_tree?date=2026-07-01&project=proj-a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := resp.(*dashboard_api.ReconcilePipelineTreeResponse)
		if !ok {
			t.Fatalf("unexpected response type %T", resp)
		}
		if got.Date != "2026-07-01" {
			t.Errorf("got.Date = %q, want 2026-07-01", got.Date)
		}
		if got.Filters.Project == nil || *got.Filters.Project != "proj-a" {
			t.Errorf("got.Filters.Project = %v, want proj-a", got.Filters.Project)
		}
		if got.Tree == nil || got.Tree.Key != "tar_received" || got.Tree.Count != 100 {
			t.Errorf("unexpected tree: %+v", got.Tree)
		}
	})

	t.Run("repo error maps to internal error code", func(t *testing.T) {
		repo := &fakeReconcileRepo{pipelineTreeErr: errors.New("doris down")}
		svc := NewReconcileService(biz.NewReconcileUseCase(repo))

		resp, err := svc.GetPipelineTree(newTestGinContext("/dashboard/v1/reconcile/pipeline_tree"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCode() != int32(gcode.CodeInternalError.Code()) {
			t.Errorf("code = %d, want %d", resp.GetCode(), gcode.CodeInternalError.Code())
		}
	})
}
