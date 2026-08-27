package biz

import (
	"context"
	"reflect"
	"testing"

	dashboard_api "fdi_data_board/api/dashboard"
)

type fakeFoDashboardRepo struct {
	fffFailReason        []*DoFailReasonItem
	fffFailReasonParam   *FffTriggerParam
	fffFailReasonErr     error
	closeReasonParam     *CloseReasonParam
	fffRunning           []*FffRunningItem
	fffRunningTotal      int64
	fffRunningParam      *FffRunningParam
	runningTrend         *FffRunningTrendData
	runningTrendByFilter map[string]*FffRunningTrendData
	runningTrendParam    *FffRunningTrendParam
	runningTrendParams   []*FffRunningTrendParam
	runningOverview      *FoRunningOverviewData
	runningOverviewParam *FffRunningParam
}

func (r *fakeFoDashboardRepo) GetFunnel(context.Context, *FunnelParam) (*FunnelData, error) {
	return nil, nil
}

func (r *fakeFoDashboardRepo) ListFffRunning(_ context.Context, param *FffRunningParam) ([]*FffRunningItem, int64, error) {
	r.fffRunningParam = param
	return r.fffRunning, r.fffRunningTotal, nil
}

func (r *fakeFoDashboardRepo) ListFffTrigger(context.Context, *FffTriggerParam) ([]*FffTriggerItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeFoDashboardRepo) ListFffClose(context.Context, *FffCloseParam) ([]*FffCloseItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeFoDashboardRepo) ListFdrTrigger(context.Context, *FdrTriggerParam) ([]*FdrTriggerItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeFoDashboardRepo) ListFclTrigger(context.Context, *FclTriggerParam) ([]*FclTriggerItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeFoDashboardRepo) ListUuidDetail(context.Context, *UuidDetailParam) ([]*UuidDetailItem, int64, error) {
	return nil, 0, nil
}

func (r *fakeFoDashboardRepo) GetCloseReason(_ context.Context, param *CloseReasonParam) ([]*CloseReasonItem, error) {
	r.closeReasonParam = param
	return nil, nil
}

func (r *fakeFoDashboardRepo) GetStageTrend(context.Context, *StageTrendParam) (*StageTrendData, error) {
	return nil, nil
}

func (r *fakeFoDashboardRepo) GetDimensions(context.Context) (*FoDimensions, error) {
	return nil, nil
}

func (r *fakeFoDashboardRepo) GetCarTypesByProject(context.Context, string) ([]string, error) {
	return nil, nil
}

func (r *fakeFoDashboardRepo) GetFffRunningTrend(_ context.Context, param *FffRunningTrendParam) (*FffRunningTrendData, error) {
	r.runningTrendParam = param
	cp := *param
	r.runningTrendParams = append(r.runningTrendParams, &cp)
	if r.runningTrendByFilter != nil {
		if data, ok := r.runningTrendByFilter[param.FilterName]; ok {
			return data, nil
		}
		return &FffRunningTrendData{}, nil
	}
	if r.runningTrend != nil {
		return r.runningTrend, nil
	}
	return &FffRunningTrendData{}, nil
}

func (r *fakeFoDashboardRepo) GetRunningOverview(_ context.Context, param *FffRunningParam) (*FoRunningOverviewData, error) {
	r.runningOverviewParam = param
	if r.runningOverview != nil {
		return r.runningOverview, nil
	}
	return &FoRunningOverviewData{}, nil
}

func (r *fakeFoDashboardRepo) GetFffOverview(context.Context, *FffTriggerParam) (*FoFffOverviewData, error) {
	return nil, nil
}

func (r *fakeFoDashboardRepo) GetFffFailReason(_ context.Context, param *FffTriggerParam) ([]*DoFailReasonItem, error) {
	r.fffFailReasonParam = param
	return r.fffFailReason, r.fffFailReasonErr
}

type fakeRunningFilterNameResolver struct {
	eventName   string
	filterNames []string
	err         error
}

func (r *fakeRunningFilterNameResolver) ResolveRunningFilterNames(_ context.Context, eventName string) ([]string, error) {
	r.eventName = eventName
	return r.filterNames, r.err
}

func TestFoDashboardUseCaseGetFffFailReason(t *testing.T) {
	repo := &fakeFoDashboardRepo{
		fffFailReason: []*DoFailReasonItem{
			{Name: "FFF-cooldown", Value: 12},
			{Name: "FFF-drm_quota", Value: 5},
		},
	}
	uc := NewFoDashboardUseCase(repo, nil)

	resp, err := uc.GetFffFailReason(context.Background(), &dashboard_api.FffTriggerRequest{
		EventNames:  "event_a,event_b",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-01",
	})
	if err != nil {
		t.Fatalf("GetFffFailReason returned error: %v", err)
	}
	if resp.Code != 0 || resp.Message != "OK" {
		t.Fatalf("unexpected response status: code=%d message=%q", resp.Code, resp.Message)
	}
	wantList := []*dashboard_api.DoFailReasonItem{
		{Name: "FFF-cooldown", Value: 12},
		{Name: "FFF-drm_quota", Value: 5},
	}
	if !reflect.DeepEqual(resp.List, wantList) {
		t.Fatalf("unexpected fail reasons: got %#v want %#v", resp.List, wantList)
	}
	wantParam := &FffTriggerParam{
		EventNames:  []string{"event_a", "event_b"},
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-01",
	}
	if !reflect.DeepEqual(repo.fffFailReasonParam, wantParam) {
		t.Fatalf("unexpected repo param: got %#v want %#v", repo.fffFailReasonParam, wantParam)
	}
}

func TestFoDashboardUseCaseGetRunningOverviewForwardsEventNames(t *testing.T) {
	repo := &fakeFoDashboardRepo{runningOverview: &FoRunningOverviewData{VehicleTotal: 12}}
	uc := NewFoDashboardUseCase(repo, nil)

	resp, err := uc.GetRunningOverview(context.Background(), &dashboard_api.FffRunningRequest{
		EventNames:  "event_a,event_b",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})
	if err != nil {
		t.Fatalf("GetRunningOverview returned error: %v", err)
	}
	if resp.VehicleTotal != 12 {
		t.Fatalf("unexpected vehicle total: got %d want 12", resp.VehicleTotal)
	}

	wantParam := &FffRunningParam{
		EventNames:  []string{"event_a", "event_b"},
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	}
	if !reflect.DeepEqual(repo.runningOverviewParam, wantParam) {
		t.Fatalf("unexpected repo param: got %#v want %#v", repo.runningOverviewParam, wantParam)
	}
}

func TestFoDashboardUseCaseGetRunningOverviewResolvesEventToFilterNames(t *testing.T) {
	repo := &fakeFoDashboardRepo{runningOverview: &FoRunningOverviewData{FilterCount: 2}}
	resolver := &fakeRunningFilterNameResolver{
		filterNames: []string{"hotupdate_filter_operator_a", "cpp_filter_b"},
	}
	uc := NewFoDashboardUseCase(repo, resolver)

	resp, err := uc.GetRunningOverview(context.Background(), &dashboard_api.FffRunningRequest{
		EventNames:  "event_x",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})
	if err != nil {
		t.Fatalf("GetRunningOverview returned error: %v", err)
	}
	if resp.FilterCount != 2 {
		t.Fatalf("unexpected filter count: got %d want 2", resp.FilterCount)
	}
	if resolver.eventName != "event_x" {
		t.Fatalf("resolver eventName = %q, want %q", resolver.eventName, "event_x")
	}

	wantParam := &FffRunningParam{
		EventNames:  []string{"hotupdate_filter_operator_a", "cpp_filter_b"},
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	}
	if !reflect.DeepEqual(repo.runningOverviewParam, wantParam) {
		t.Fatalf("unexpected repo param: got %#v want %#v", repo.runningOverviewParam, wantParam)
	}
}

func TestFoDashboardUseCaseListFffRunningResolvesEventToFilterNames(t *testing.T) {
	repo := &fakeFoDashboardRepo{
		fffRunningTotal: 2,
		fffRunning: []*FffRunningItem{
			{FilterName: "hotupdate_filter_operator_a"},
			{FilterName: "cpp_filter_b"},
		},
	}
	resolver := &fakeRunningFilterNameResolver{
		filterNames: []string{"hotupdate_filter_operator_a", "cpp_filter_b"},
	}
	uc := NewFoDashboardUseCase(repo, resolver)

	resp, err := uc.ListFffRunning(context.Background(), &dashboard_api.FffRunningRequest{
		EventNames:  "event_x",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
		Page:        2,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("ListFffRunning returned error: %v", err)
	}
	if resp.Total != 2 || resp.Page != 2 || resp.PageSize != 20 {
		t.Fatalf("unexpected pagination: total=%d page=%d page_size=%d", resp.Total, resp.Page, resp.PageSize)
	}
	if resolver.eventName != "event_x" {
		t.Fatalf("resolver eventName = %q, want %q", resolver.eventName, "event_x")
	}

	wantParam := &FffRunningParam{
		EventNames:  []string{"hotupdate_filter_operator_a", "cpp_filter_b"},
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
		Page:        2,
		PageSize:    20,
	}
	if !reflect.DeepEqual(repo.fffRunningParam, wantParam) {
		t.Fatalf("unexpected repo param: got %#v want %#v", repo.fffRunningParam, wantParam)
	}
}

func TestFoDashboardUseCaseGetCloseReasonForwardsFilterName(t *testing.T) {
	repo := &fakeFoDashboardRepo{}
	uc := NewFoDashboardUseCase(repo, nil)

	_, err := uc.GetCloseReason(context.Background(), &dashboard_api.CloseReasonRequest{
		FilterName:  "filter_a",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})
	if err != nil {
		t.Fatalf("GetCloseReason returned error: %v", err)
	}

	wantParam := &CloseReasonParam{
		FilterName:  "filter_a",
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	}
	if !reflect.DeepEqual(repo.closeReasonParam, wantParam) {
		t.Fatalf("unexpected repo param: got %#v want %#v", repo.closeReasonParam, wantParam)
	}
}

func TestFoDashboardUseCaseGetFffRunningTrendResolvesEventToFilterNames(t *testing.T) {
	repo := &fakeFoDashboardRepo{
		runningTrendByFilter: map[string]*FffRunningTrendData{
			"hotupdate_filter_operator_a": {
				Dates:  []string{"2026-06-01", "2026-06-02"},
				Counts: []int64{7, 3},
			},
			"cpp_filter_b": {
				Dates:  []string{"2026-06-01", "2026-06-03"},
				Counts: []int64{5, 2},
			},
		},
	}
	resolver := &fakeRunningFilterNameResolver{
		filterNames: []string{"hotupdate_filter_operator_a", "cpp_filter_b"},
	}
	uc := NewFoDashboardUseCase(repo, resolver)

	resp, err := uc.GetFffRunningTrend(context.Background(), &dashboard_api.FffRunningTrendRequest{
		FilterName:  "event_x",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})
	if err != nil {
		t.Fatalf("GetFffRunningTrend returned error: %v", err)
	}

	if resolver.eventName != "event_x" {
		t.Fatalf("resolver eventName = %q, want %q", resolver.eventName, "event_x")
	}
	wantParams := []*FffRunningTrendParam{
		{
			FilterName:  "hotupdate_filter_operator_a",
			ProjectName: "project_x",
			CarTypes:    []string{"SUV", "MPV"},
			StartDt:     "2026-06-01",
			EndDt:       "2026-06-03",
		},
		{
			FilterName:  "cpp_filter_b",
			ProjectName: "project_x",
			CarTypes:    []string{"SUV", "MPV"},
			StartDt:     "2026-06-01",
			EndDt:       "2026-06-03",
		},
	}
	if !reflect.DeepEqual(repo.runningTrendParams, wantParams) {
		t.Fatalf("unexpected repo params: got %#v want %#v", repo.runningTrendParams, wantParams)
	}
	if !reflect.DeepEqual(resp.Dates, []string{"2026-06-01", "2026-06-02", "2026-06-03"}) {
		t.Fatalf("unexpected dates: got %#v", resp.Dates)
	}
	if !reflect.DeepEqual(resp.Counts, []int64{12, 3, 2}) {
		t.Fatalf("unexpected counts: got %#v", resp.Counts)
	}
}
