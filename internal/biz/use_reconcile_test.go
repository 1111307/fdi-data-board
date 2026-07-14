package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	dashboard_api "fdi_data_board/api/dashboard"
)

// fakeReconcileRepo 实现 ReconcileRepo，供 biz 层单测使用，记录每次调用的入参
type fakeReconcileRepo struct {
	overviewDate string
	overviewData *ReconcileOverviewData
	overviewErr  error

	trendStart string
	trendEnd   string
	trendData  *ReconcileTrendData
	trendErr   error

	moduleDate    string
	moduleProject string
	moduleList    []*ReconcileModuleItem
	moduleErr     error

	projectDate    string
	projectOrderBy string
	projectLimit   int
	projectList    []*ReconcileProjectItem
	projectErr     error

	decodeStatusDate string
	decodeStatusList []*ReconcileDecodeStatusItem
	decodeStatusErr  error

	md5ListDate string
	md5ListType string
	md5ListLim  int
	md5List     []*ReconcileMd5Item
	md5ListErr  error

	md5DetailDate string
	md5DetailMd5  string
	md5DetailList []*ReconcileMd5DetailItem
	md5DetailErr  error

	eventListDate       string
	eventListModuleName string
	eventListProject    string
	eventListType       string
	eventListPage       int
	eventListPageSize   int
	eventListItems      []*ReconcileEventItem
	eventListTotal      int64
	eventListErr        error

	recordConsistencyDate  string
	recordConsistencyLimit int
	recordConsistencyItems []*ReconcileRecordConsistencyItem
	recordConsistencyErr   error

	uuidSourceDate  string
	uuidSourceItems []*ReconcileUuidSourceItem
	uuidSourceErr   error

	failureSummaryDate string
	failureSummaryData *ReconcileFailureSummaryData
	failureSummaryErr  error

	pipelineTreeDate       string
	pipelineTreeProject    string
	pipelineTreeModuleName string
	pipelineTreeMd5        string
	pipelineTreeData       *ReconcilePipelineTreeData
	pipelineTreeErr        error
}

func (f *fakeReconcileRepo) GetOverview(ctx context.Context, date string) (*ReconcileOverviewData, error) {
	f.overviewDate = date
	return f.overviewData, f.overviewErr
}

func (f *fakeReconcileRepo) GetTrend(ctx context.Context, startDt, endDt string) (*ReconcileTrendData, error) {
	f.trendStart = startDt
	f.trendEnd = endDt
	return f.trendData, f.trendErr
}

func (f *fakeReconcileRepo) GetModule(ctx context.Context, date, project string) ([]*ReconcileModuleItem, error) {
	f.moduleDate = date
	f.moduleProject = project
	return f.moduleList, f.moduleErr
}

func (f *fakeReconcileRepo) GetProject(ctx context.Context, date, orderBy string, limit int) ([]*ReconcileProjectItem, error) {
	f.projectDate = date
	f.projectOrderBy = orderBy
	f.projectLimit = limit
	return f.projectList, f.projectErr
}

func (f *fakeReconcileRepo) GetDecodeStatus(ctx context.Context, date string) ([]*ReconcileDecodeStatusItem, error) {
	f.decodeStatusDate = date
	return f.decodeStatusList, f.decodeStatusErr
}

func (f *fakeReconcileRepo) ListDiffMd5(ctx context.Context, date, diffType string, limit int) ([]*ReconcileMd5Item, error) {
	f.md5ListDate = date
	f.md5ListType = diffType
	f.md5ListLim = limit
	return f.md5List, f.md5ListErr
}

func (f *fakeReconcileRepo) GetMd5Detail(ctx context.Context, date, md5 string) ([]*ReconcileMd5DetailItem, error) {
	f.md5DetailDate = date
	f.md5DetailMd5 = md5
	return f.md5DetailList, f.md5DetailErr
}

func (f *fakeReconcileRepo) ListEventList(ctx context.Context, date, moduleName, project, eventType string, page, pageSize int) ([]*ReconcileEventItem, int64, error) {
	f.eventListDate = date
	f.eventListModuleName = moduleName
	f.eventListProject = project
	f.eventListType = eventType
	f.eventListPage = page
	f.eventListPageSize = pageSize
	return f.eventListItems, f.eventListTotal, f.eventListErr
}

func (f *fakeReconcileRepo) GetRecordConsistency(ctx context.Context, date string, limit int) ([]*ReconcileRecordConsistencyItem, error) {
	f.recordConsistencyDate = date
	f.recordConsistencyLimit = limit
	return f.recordConsistencyItems, f.recordConsistencyErr
}

func (f *fakeReconcileRepo) GetUuidSource(ctx context.Context, date string) ([]*ReconcileUuidSourceItem, error) {
	f.uuidSourceDate = date
	return f.uuidSourceItems, f.uuidSourceErr
}

func (f *fakeReconcileRepo) GetFailureSummary(ctx context.Context, date string) (*ReconcileFailureSummaryData, error) {
	f.failureSummaryDate = date
	return f.failureSummaryData, f.failureSummaryErr
}

func (f *fakeReconcileRepo) GetPipelineTree(ctx context.Context, date, project, moduleName, md5 string) (*ReconcilePipelineTreeData, error) {
	f.pipelineTreeDate = date
	f.pipelineTreeProject = project
	f.pipelineTreeModuleName = moduleName
	f.pipelineTreeMd5 = md5
	return f.pipelineTreeData, f.pipelineTreeErr
}

func TestReconcileUseCase_GetOverview(t *testing.T) {
	t.Run("defaults date to today and maps fields", func(t *testing.T) {
		repo := &fakeReconcileRepo{overviewData: &ReconcileOverviewData{
			Bag: ReconcileOverviewBag{
				Total:             100,
				DecodeSuccess:     90,
				DecodeFailed:      8,
				DecodePartial:     2,
				DecodeSuccessRate: 0.9,
			},
			EventParse: ReconcileOverviewEventParse{
				Expected:       1000,
				SendFailed:     5,
				SendFailedRate: 0.005,
			},
			EventLand: ReconcileOverviewEventLand{
				Matched:       950,
				ConvertFailed: 20,
				LandingFailed: 10,
				Missing:       20,
				MatchRate:     0.95,
			},
			ExtraConsume: 3,
		}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetOverview(context.Background(), &dashboard_api.ReconcileOverviewRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantDate := time.Now().Format("2006-01-02")
		if repo.overviewDate != wantDate {
			t.Errorf("repo called with date %q, want %q", repo.overviewDate, wantDate)
		}
		if resp.Date != wantDate {
			t.Errorf("resp.Date = %q, want %q", resp.Date, wantDate)
		}
		if resp.Bag.Total != 100 || resp.Bag.DecodePartial != 2 || resp.Bag.DecodeSuccessRate != 0.9 {
			t.Errorf("unexpected bag: %+v", resp.Bag)
		}
		if resp.EventParse.Expected != 1000 || resp.EventParse.SendFailed != 5 {
			t.Errorf("unexpected event_parse: %+v", resp.EventParse)
		}
		if resp.EventLand.Matched != 950 || resp.EventLand.ConvertFailed != 20 || resp.EventLand.LandingFailed != 10 || resp.EventLand.MatchRate != 0.95 {
			t.Errorf("unexpected event_land: %+v", resp.EventLand)
		}
		if resp.ExtraConsume != 3 {
			t.Errorf("resp.ExtraConsume = %d, want 3", resp.ExtraConsume)
		}
	})

	t.Run("keeps explicit date and propagates repo error", func(t *testing.T) {
		repo := &fakeReconcileRepo{overviewErr: errors.New("doris down")}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetOverview(context.Background(), &dashboard_api.ReconcileOverviewRequest{Date: "2026-07-01"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if repo.overviewDate != "2026-07-01" {
			t.Errorf("repo called with date %q, want 2026-07-01", repo.overviewDate)
		}
	})
}

func TestReconcileUseCase_GetTrend(t *testing.T) {
	t.Run("defaults to last 7 days when unset", func(t *testing.T) {
		repo := &fakeReconcileRepo{trendData: &ReconcileTrendData{}}
		uc := NewReconcileUseCase(repo)

		if _, err := uc.GetTrend(context.Background(), &dashboard_api.ReconcileTrendRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		start, err := time.Parse("2006-01-02", repo.trendStart)
		if err != nil {
			t.Fatalf("trendStart %q not a valid date: %v", repo.trendStart, err)
		}
		end, err := time.Parse("2006-01-02", repo.trendEnd)
		if err != nil {
			t.Fatalf("trendEnd %q not a valid date: %v", repo.trendEnd, err)
		}
		if days := int(end.Sub(start).Hours() / 24); days != 6 {
			t.Errorf("default range spans %d days, want 6", days)
		}
	})

	t.Run("passes through explicit range and maps points", func(t *testing.T) {
		repo := &fakeReconcileRepo{trendData: &ReconcileTrendData{Points: []*ReconcileTrendPoint{
			{Dt: "2026-07-01", Expected: 100, Matched: 90, ConvertFailed: 5, LandingFailed: 3, Missing: 2, MatchRate: 0.9},
		}}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetTrend(context.Background(), &dashboard_api.ReconcileTrendRequest{
			StartDt: "2026-07-01",
			EndDt:   "2026-07-03",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.trendStart != "2026-07-01" || repo.trendEnd != "2026-07-03" {
			t.Errorf("got range [%s, %s], want [2026-07-01, 2026-07-03]", repo.trendStart, repo.trendEnd)
		}
		if resp.Start != "2026-07-01" || resp.End != "2026-07-03" {
			t.Errorf("unexpected resp range: start=%q end=%q", resp.Start, resp.End)
		}
		if len(resp.Points) != 1 || resp.Points[0].Dt != "2026-07-01" || resp.Points[0].ConvertFailed != 5 || resp.Points[0].LandingFailed != 3 {
			t.Errorf("unexpected points: %+v", resp.Points)
		}
	})

	t.Run("invalid date returns ErrInvalidDateRange", func(t *testing.T) {
		repo := &fakeReconcileRepo{trendData: &ReconcileTrendData{}}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetTrend(context.Background(), &dashboard_api.ReconcileTrendRequest{
			StartDt: "not-a-date",
			EndDt:   "2026-07-03",
		})
		if !errors.Is(err, ErrInvalidDateRange) {
			t.Errorf("err = %v, want ErrInvalidDateRange", err)
		}
	})
}

func TestReconcileUseCase_GetModule(t *testing.T) {
	t.Run("passes date and empty project through", func(t *testing.T) {
		repo := &fakeReconcileRepo{moduleList: []*ReconcileModuleItem{
			{ModuleName: "fff_close", Expected: 10, Matched: 8, ConvertFailed: 1, LandingFailed: 1, Missing: 0, SendFailed: 0, MatchRate: 0.8},
		}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetModule(context.Background(), &dashboard_api.ReconcileModuleRequest{Date: "2026-07-01"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.moduleDate != "2026-07-01" || repo.moduleProject != "" {
			t.Errorf("repo called with date %q project %q", repo.moduleDate, repo.moduleProject)
		}
		if resp.Project != "" {
			t.Errorf("resp.Project = %q, want empty", resp.Project)
		}
		if len(resp.Modules) != 1 || resp.Modules[0].ModuleName != "fff_close" || resp.Modules[0].ConvertFailed != 1 || resp.Modules[0].LandingFailed != 1 {
			t.Errorf("unexpected modules: %+v", resp.Modules)
		}
	})

	t.Run("passes project through for cross-drilldown", func(t *testing.T) {
		repo := &fakeReconcileRepo{moduleList: []*ReconcileModuleItem{}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetModule(context.Background(), &dashboard_api.ReconcileModuleRequest{Date: "2026-07-01", Project: "proj-a"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.moduleProject != "proj-a" {
			t.Errorf("repo called with project %q, want proj-a", repo.moduleProject)
		}
		if resp.Project != "proj-a" {
			t.Errorf("resp.Project = %q, want proj-a", resp.Project)
		}
	})
}

func TestReconcileUseCase_GetProject(t *testing.T) {
	t.Run("defaults order_by to missing and limit to default", func(t *testing.T) {
		repo := &fakeReconcileRepo{projectList: []*ReconcileProjectItem{
			{Project: "proj-a", Expected: 8000, Matched: 7950, ConvertFailed: 30, LandingFailed: 20, Missing: 50, MatchRate: 0.9938},
		}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetProject(context.Background(), &dashboard_api.ReconcileProjectRequest{Date: "2026-07-01"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.projectDate != "2026-07-01" || repo.projectOrderBy != "missing" || repo.projectLimit != defaultMd5ListLimit {
			t.Errorf("unexpected repo call: date=%q orderBy=%q limit=%d", repo.projectDate, repo.projectOrderBy, repo.projectLimit)
		}
		if len(resp.Projects) != 1 || resp.Projects[0].Project != "proj-a" || resp.Projects[0].ConvertFailed != 30 || resp.Projects[0].LandingFailed != 20 {
			t.Errorf("unexpected projects: %+v", resp.Projects)
		}
	})

	t.Run("accepts explicit order_by and limit", func(t *testing.T) {
		repo := &fakeReconcileRepo{projectList: []*ReconcileProjectItem{}}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetProject(context.Background(), &dashboard_api.ReconcileProjectRequest{OrderBy: "expected", Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.projectOrderBy != "expected" || repo.projectLimit != 5 {
			t.Errorf("unexpected repo call: orderBy=%q limit=%d", repo.projectOrderBy, repo.projectLimit)
		}
	})

	t.Run("rejects unsupported order_by", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetProject(context.Background(), &dashboard_api.ReconcileProjectRequest{OrderBy: "bogus"})
		if !errors.Is(err, ErrInvalidOrderBy) {
			t.Errorf("err = %v, want ErrInvalidOrderBy", err)
		}
	})
}

func TestReconcileUseCase_GetDecodeStatus(t *testing.T) {
	repo := &fakeReconcileRepo{decodeStatusList: []*ReconcileDecodeStatusItem{
		{Status: 1, Stage: 0, Count: 95},
		{Status: 2, Stage: 5, Count: 5},
	}}
	uc := NewReconcileUseCase(repo)

	resp, err := uc.GetDecodeStatus(context.Background(), &dashboard_api.ReconcileDecodeStatusRequest{Date: "2026-07-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(resp.Items))
	}
	if resp.Items[1].Status != 2 || resp.Items[1].Stage != 5 || resp.Items[1].Count != 5 {
		t.Errorf("unexpected item: %+v", resp.Items[1])
	}
}

func TestReconcileUseCase_ListMd5(t *testing.T) {
	t.Run("defaults type to missing and limit to default", func(t *testing.T) {
		repo := &fakeReconcileRepo{md5List: []*ReconcileMd5Item{{Md5: "abc", Count: 3}}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.ListMd5(context.Background(), &dashboard_api.ReconcileMd5ListRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.md5ListType != "missing" {
			t.Errorf("diffType = %q, want missing", repo.md5ListType)
		}
		if repo.md5ListLim != defaultMd5ListLimit {
			t.Errorf("limit = %d, want %d", repo.md5ListLim, defaultMd5ListLimit)
		}
		if resp.Type != "missing" || len(resp.Items) != 1 {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("accepts convert_failed/landing_failed/decode_failed types and explicit limit", func(t *testing.T) {
		for _, typ := range []string{"convert_failed", "landing_failed", "decode_failed"} {
			repo := &fakeReconcileRepo{md5List: []*ReconcileMd5Item{{Md5: "def", Status: 2}}}
			uc := NewReconcileUseCase(repo)

			_, err := uc.ListMd5(context.Background(), &dashboard_api.ReconcileMd5ListRequest{
				Type:  typ,
				Limit: 5,
				Date:  "2026-07-01",
			})
			if err != nil {
				t.Fatalf("unexpected error for type %q: %v", typ, err)
			}
			if repo.md5ListType != typ || repo.md5ListLim != 5 || repo.md5ListDate != "2026-07-01" {
				t.Errorf("unexpected repo call: type=%q limit=%d date=%q", repo.md5ListType, repo.md5ListLim, repo.md5ListDate)
			}
		}
	})

	t.Run("rejects unsupported type", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		uc := NewReconcileUseCase(repo)

		_, err := uc.ListMd5(context.Background(), &dashboard_api.ReconcileMd5ListRequest{Type: "failed"})
		if !errors.Is(err, ErrInvalidMd5Type) {
			t.Errorf("err = %v, want ErrInvalidMd5Type", err)
		}
	})
}

func TestReconcileUseCase_GetMd5Detail(t *testing.T) {
	t.Run("requires md5", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetMd5Detail(context.Background(), &dashboard_api.ReconcileMd5DetailRequest{})
		if !errors.Is(err, ErrMd5Required) {
			t.Errorf("err = %v, want ErrMd5Required", err)
		}
	})

	t.Run("returns mapped detail list", func(t *testing.T) {
		repo := &fakeReconcileRepo{md5DetailList: []*ReconcileMd5DetailItem{
			{Uuid: "u1", ModuleName: "fdr_status", SendStatus: 1, Consumed: true, ConsumeStatus: 1},
		}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetMd5Detail(context.Background(), &dashboard_api.ReconcileMd5DetailRequest{Md5: "abc123", Date: "2026-07-01"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.md5DetailMd5 != "abc123" || repo.md5DetailDate != "2026-07-01" {
			t.Errorf("unexpected repo call: md5=%q date=%q", repo.md5DetailMd5, repo.md5DetailDate)
		}
		if resp.Md5 != "abc123" || len(resp.Items) != 1 || !resp.Items[0].Consumed {
			t.Errorf("unexpected response: %+v", resp)
		}
	})
}

func TestReconcileUseCase_ListEventList(t *testing.T) {
	t.Run("defaults type to missing, page to 1 and page_size to default", func(t *testing.T) {
		repo := &fakeReconcileRepo{eventListItems: []*ReconcileEventItem{{Md5: "abc", Uuid: "u1"}}, eventListTotal: 1}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.ListEventList(context.Background(), &dashboard_api.ReconcileEventListRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.eventListType != "missing" {
			t.Errorf("eventType = %q, want missing", repo.eventListType)
		}
		if repo.eventListPage != 1 || repo.eventListPageSize != defaultEventListPageSize {
			t.Errorf("page=%d pageSize=%d, want 1/%d", repo.eventListPage, repo.eventListPageSize, defaultEventListPageSize)
		}
		if resp.Type != "missing" || resp.Total != 1 || len(resp.Items) != 1 {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("caps page_size at max and passes through module_name/project/type/page", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		uc := NewReconcileUseCase(repo)

		_, err := uc.ListEventList(context.Background(), &dashboard_api.ReconcileEventListRequest{
			ModuleName: "fff_close",
			Project:    "proj_a",
			Type:       "mismatched",
			Page:       2,
			PageSize:   9999,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.eventListModuleName != "fff_close" || repo.eventListProject != "proj_a" || repo.eventListType != "mismatched" || repo.eventListPage != 2 {
			t.Errorf("unexpected repo call: module=%q project=%q type=%q page=%d", repo.eventListModuleName, repo.eventListProject, repo.eventListType, repo.eventListPage)
		}
		if repo.eventListPageSize != maxEventListPageSize {
			t.Errorf("pageSize = %d, want capped at %d", repo.eventListPageSize, maxEventListPageSize)
		}
	})

	t.Run("accepts send_failed/convert_failed/landing_failed types", func(t *testing.T) {
		for _, typ := range []string{"send_failed", "convert_failed", "landing_failed"} {
			repo := &fakeReconcileRepo{}
			uc := NewReconcileUseCase(repo)

			_, err := uc.ListEventList(context.Background(), &dashboard_api.ReconcileEventListRequest{Type: typ})
			if err != nil {
				t.Fatalf("unexpected error for type %q: %v", typ, err)
			}
			if repo.eventListType != typ {
				t.Errorf("eventType = %q, want %q", repo.eventListType, typ)
			}
		}
	})

	t.Run("rejects unsupported type", func(t *testing.T) {
		repo := &fakeReconcileRepo{}
		uc := NewReconcileUseCase(repo)

		_, err := uc.ListEventList(context.Background(), &dashboard_api.ReconcileEventListRequest{Type: "bogus"})
		if !errors.Is(err, ErrInvalidEventType) {
			t.Errorf("err = %v, want ErrInvalidEventType", err)
		}
	})
}

func TestReconcileUseCase_GetRecordConsistency(t *testing.T) {
	repo := &fakeReconcileRepo{recordConsistencyItems: []*ReconcileRecordConsistencyItem{
		{Md5: "abc", ParsedLineCount: 100, DetailCount: 98, Diff: 2},
	}}
	uc := NewReconcileUseCase(repo)

	resp, err := uc.GetRecordConsistency(context.Background(), &dashboard_api.ReconcileRecordConsistencyRequest{Date: "2026-07-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.recordConsistencyDate != "2026-07-01" || repo.recordConsistencyLimit != defaultMd5ListLimit {
		t.Errorf("unexpected repo call: date=%q limit=%d", repo.recordConsistencyDate, repo.recordConsistencyLimit)
	}
	if len(resp.Items) != 1 || resp.Items[0].Diff != 2 {
		t.Errorf("unexpected items: %+v", resp.Items)
	}
}

func TestReconcileUseCase_GetUuidSource(t *testing.T) {
	rate := 0.9
	repo := &fakeReconcileRepo{uuidSourceItems: []*ReconcileUuidSourceItem{
		{UuidSource: "real", Expected: 100, Matched: 90, Missing: 10, MatchRate: &rate},
		{UuidSource: "gen_fallback", Expected: 0, Matched: 0, Missing: 0, MatchRate: nil},
	}}
	uc := NewReconcileUseCase(repo)

	resp, err := uc.GetUuidSource(context.Background(), &dashboard_api.ReconcileUuidSourceRequest{Date: "2026-07-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.uuidSourceDate != "2026-07-01" {
		t.Errorf("repo called with date %q, want 2026-07-01", repo.uuidSourceDate)
	}
	if len(resp.Sources) != 2 {
		t.Fatalf("got %d sources, want 2", len(resp.Sources))
	}
	if resp.Sources[0].MatchRate == nil || *resp.Sources[0].MatchRate != 0.9 {
		t.Errorf("unexpected match_rate for real: %+v", resp.Sources[0])
	}
	if resp.Sources[1].MatchRate != nil {
		t.Errorf("expected nil match_rate for gen_fallback with expected=0, got %v", *resp.Sources[1].MatchRate)
	}
}

func TestReconcileUseCase_GetFailureSummary(t *testing.T) {
	t.Run("defaults date to today and maps all four sections", func(t *testing.T) {
		repo := &fakeReconcileRepo{failureSummaryData: &ReconcileFailureSummaryData{
			DecodeFailed: []*ReconcileDecodeFailedItem{
				{Stage: 1, Count: 10, SampleErrorMsg: "unzip failed"},
			},
			SendFailed: []*ReconcileSendFailedItem{
				{ModuleName: "fff_close", ErrDetail: "5:timeout", Count: 3},
			},
			ConvertFailed: []*ReconcileConvertFailedItem{
				{ModuleName: "fff_close", ErrDetail: "3:nil trigger", Count: 4},
			},
			LandingFailed: []*ReconcileLandingFailedItem{
				{ModuleName: "fff_close", ErrDetail: "6:write failed", Count: 5},
			},
		}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetFailureSummary(context.Background(), &dashboard_api.ReconcileFailureSummaryRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantDate := time.Now().Format("2006-01-02")
		if repo.failureSummaryDate != wantDate || resp.Date != wantDate {
			t.Errorf("date = repo:%q resp:%q, want %q", repo.failureSummaryDate, resp.Date, wantDate)
		}
		if len(resp.DecodeFailed) != 1 || resp.DecodeFailed[0].Stage != 1 || resp.DecodeFailed[0].StageDesc == "" {
			t.Errorf("unexpected decode_failed: %+v", resp.DecodeFailed)
		}
		if len(resp.SendFailed) != 1 || resp.SendFailed[0].ModuleName != "fff_close" || resp.SendFailed[0].Count != 3 {
			t.Errorf("unexpected send_failed: %+v", resp.SendFailed)
		}
		if len(resp.ConvertFailed) != 1 || resp.ConvertFailed[0].Count != 4 {
			t.Errorf("unexpected convert_failed: %+v", resp.ConvertFailed)
		}
		if len(resp.LandingFailed) != 1 || resp.LandingFailed[0].Count != 5 {
			t.Errorf("unexpected landing_failed: %+v", resp.LandingFailed)
		}
	})

	t.Run("keeps explicit date and propagates repo error", func(t *testing.T) {
		repo := &fakeReconcileRepo{failureSummaryErr: errors.New("doris down")}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetFailureSummary(context.Background(), &dashboard_api.ReconcileFailureSummaryRequest{Date: "2026-07-01"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if repo.failureSummaryDate != "2026-07-01" {
			t.Errorf("repo called with date %q, want 2026-07-01", repo.failureSummaryDate)
		}
	})
}

func TestReconcileUseCase_GetPipelineTree(t *testing.T) {
	t.Run("builds tree with correct shape, sums and rates", func(t *testing.T) {
		repo := &fakeReconcileRepo{pipelineTreeData: &ReconcilePipelineTreeData{
			Bag: ReconcilePipelineBagData{
				Total:           5000,
				ParseSuccess:    4985,
				DecodeSuccess:   4900,
				DecodePartial:   85,
				DecodeFailed:    15,
				StatusLineCount: 5000,
				SkipLineCount:   10,
				ParsedLineCount: 12340,
			},
			BagFailureReasons: []*ReconcileDecodeFailedItem{
				{Stage: 1, Count: 15, SampleErrorMsg: "unzip failed"},
			},
			EventParse: ReconcilePipelineEventParseData{
				ParseSuccess: 12329,
				ParseFailed:  3,
				SendSuccess:  12322,
				SendFailed:   7,
			},
			EventParseFailureReasons: []*ReconcileParseFailedItem{
				{ModuleName: "fff_close", ErrDetail: "4:no extractable events in trigger module json", Count: 3},
			},
			EventSendFailureReasons: []*ReconcileSendFailedItem{
				{ModuleName: "fff_close", ErrDetail: "5:broker unavailable", Count: 7},
			},
			EventLand: ReconcilePipelineEventLandData{
				Matched:       12300,
				ConvertFailed: 12,
				LandingFailed: 8,
				Missing:       2,
			},
			ConvertFailedReasons: []*ReconcileConvertFailedItem{
				{ModuleName: "fff_close", ErrDetail: "3:nil trigger", Count: 12},
			},
			LandingFailedReasons: []*ReconcileLandingFailedItem{
				{ModuleName: "fff_close", ErrDetail: "6:write failed", Count: 8},
			},
		}}
		uc := NewReconcileUseCase(repo)

		resp, err := uc.GetPipelineTree(context.Background(), &dashboard_api.ReconcilePipelineTreeRequest{
			Project:    "proj-a",
			ModuleName: "fff_close",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantDate := time.Now().Format("2006-01-02")
		if repo.pipelineTreeDate != wantDate || resp.Date != wantDate {
			t.Errorf("date = repo:%q resp:%q, want %q", repo.pipelineTreeDate, resp.Date, wantDate)
		}
		if repo.pipelineTreeProject != "proj-a" || repo.pipelineTreeModuleName != "fff_close" {
			t.Errorf("unexpected repo call: project=%q module_name=%q", repo.pipelineTreeProject, repo.pipelineTreeModuleName)
		}
		if resp.Filters.Project == nil || *resp.Filters.Project != "proj-a" {
			t.Errorf("resp.Filters.Project = %v, want proj-a", resp.Filters.Project)
		}
		if resp.Filters.ModuleName == nil || *resp.Filters.ModuleName != "fff_close" {
			t.Errorf("resp.Filters.ModuleName = %v, want fff_close", resp.Filters.ModuleName)
		}
		if resp.Filters.Md5 != nil {
			t.Errorf("resp.Filters.Md5 = %v, want nil", resp.Filters.Md5)
		}

		root := resp.Tree
		if root == nil {
			t.Fatal("resp.Tree is nil")
		}
		if root.Key != "tar_received" || root.Count != 5000 {
			t.Fatalf("unexpected root: %+v", root)
		}
		if len(root.Children) != 2 {
			t.Fatalf("root has %d children, want 2", len(root.Children))
		}
		tarParseSuccess, tarParseFailed := root.Children[0], root.Children[1]
		if tarParseSuccess.Count+tarParseFailed.Count != root.Count {
			t.Errorf("tar-level children sum %d != root.Count %d", tarParseSuccess.Count+tarParseFailed.Count, root.Count)
		}
		if tarParseFailed.Count != 15 || len(tarParseFailed.FailureReasons) != 1 || tarParseFailed.FailureReasons[0].Code != "1" {
			t.Errorf("unexpected tar_parse_failed: %+v", tarParseFailed)
		}
		if tarParseSuccess.Meta["skip_line_count"] != 10 || tarParseSuccess.Meta["parsed_line_count"] != 12340 {
			t.Errorf("unexpected tar_parse_success meta: %+v", tarParseSuccess.Meta)
		}

		if len(tarParseSuccess.Children) != 2 {
			t.Fatalf("tar_parse_success has %d children, want 2", len(tarParseSuccess.Children))
		}
		eventParseSuccess, eventParseFailed := tarParseSuccess.Children[0], tarParseSuccess.Children[1]
		if eventParseSuccess.Count != 12329 || eventParseFailed.Count != 3 {
			t.Errorf("unexpected event parse split: success=%d failed=%d", eventParseSuccess.Count, eventParseFailed.Count)
		}
		if eventParseSuccess.Rate == nil || *eventParseSuccess.Rate != 0.9998 {
			t.Errorf("eventParseSuccess.Rate = %v, want 0.9998", eventParseSuccess.Rate)
		}
		if len(eventParseFailed.FailureReasons) != 1 || eventParseFailed.FailureReasons[0].Code != "4" || eventParseFailed.FailureReasons[0].ModuleName != "fff_close" {
			t.Errorf("unexpected event_parse_failed reasons: %+v", eventParseFailed.FailureReasons)
		}

		if len(eventParseSuccess.Children) != 2 {
			t.Fatalf("event_parse_success has %d children, want 2", len(eventParseSuccess.Children))
		}
		eventSendSuccess, eventSendFailed := eventParseSuccess.Children[0], eventParseSuccess.Children[1]
		if eventSendSuccess.Count != 12322 || eventSendFailed.Count != 7 {
			t.Errorf("unexpected event send split: success=%d failed=%d", eventSendSuccess.Count, eventSendFailed.Count)
		}
		if len(eventSendFailed.FailureReasons) != 1 || eventSendFailed.FailureReasons[0].Code != "5" || eventSendFailed.FailureReasons[0].ModuleName != "fff_close" {
			t.Errorf("unexpected event_send_failed reasons: %+v", eventSendFailed.FailureReasons)
		}

		if len(eventSendSuccess.Children) != 4 {
			t.Fatalf("event_send_success has %d children, want 4", len(eventSendSuccess.Children))
		}
		var landSum int64
		for _, c := range eventSendSuccess.Children {
			landSum += c.Count
		}
		if landSum != eventSendSuccess.Count {
			t.Errorf("event_land children sum %d != event_send_success.Count %d", landSum, eventSendSuccess.Count)
		}
		convertFailedNode := eventSendSuccess.Children[1]
		if convertFailedNode.Key != "event_land_convert_failed" || len(convertFailedNode.FailureReasons) != 1 || convertFailedNode.FailureReasons[0].Code != "3" {
			t.Errorf("unexpected event_land_convert_failed: %+v", convertFailedNode)
		}
		landingFailedNode := eventSendSuccess.Children[2]
		if landingFailedNode.Key != "event_land_landing_failed" || len(landingFailedNode.FailureReasons) != 1 || landingFailedNode.FailureReasons[0].Code != "6" {
			t.Errorf("unexpected event_land_landing_failed: %+v", landingFailedNode)
		}
	})

	t.Run("passes md5 through and propagates repo error", func(t *testing.T) {
		repo := &fakeReconcileRepo{pipelineTreeErr: errors.New("doris down")}
		uc := NewReconcileUseCase(repo)

		_, err := uc.GetPipelineTree(context.Background(), &dashboard_api.ReconcilePipelineTreeRequest{
			Date: "2026-07-01",
			Md5:  "abc123",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if repo.pipelineTreeDate != "2026-07-01" || repo.pipelineTreeMd5 != "abc123" {
			t.Errorf("unexpected repo call: date=%q md5=%q", repo.pipelineTreeDate, repo.pipelineTreeMd5)
		}
	})
}
