package biz

import (
	"context"
	"reflect"
	"testing"

	dashboard_api "fdi_data_board/api/dashboard"
)

type fakeDoDashboardRepo struct {
	closeTopParam *DoCommonParam
}

func (r *fakeDoDashboardRepo) GetOverview(context.Context, *DoOverviewParam) ([]*DoOverviewItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetTrend(context.Context, *DoTrendParam) (*DoTrendData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetFailReason(context.Context, *DoFailReasonParam) ([]*DoFailReasonItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetCoolTop(context.Context, *DoCommonParam) ([]*DoCoolTopItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetSwVersion(context.Context, *DoCommonParam) ([]*DoSwVersionItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetTriggerRank(context.Context, *DoCommonParam) ([]*DoCoolTopItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetProjectCar(context.Context, *DoCommonParam) (*DoProjectCarData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetMemTop(context.Context, *DoCommonParam) ([]*DoEventTopItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetDiskTop(context.Context, *DoCommonParam) ([]*DoEventTopItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetCloseTop(_ context.Context, param *DoCommonParam) ([]*DoCoolTopItem, error) {
	r.closeTopParam = param
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetQuotaTop(context.Context, *DoCommonParam) ([]*DoEventTopItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetProjectEvent(context.Context, *DoCommonParam) ([]*DoProjectEventItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetNetSpeed(context.Context, *DoCommonParam) (*DoNetSpeedData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetFclBw(context.Context, *DoCommonParam) (*DoFclBwData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetTopVehicles(context.Context, *DoVehicleParam) ([]*DoVehicleItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetAnomalyVehicles(context.Context, *DoAnomalyParam) ([]*DoVehicleItem, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetActiveTrend(context.Context, *DoVehicleParam) (*DoActiveTrendData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetDoFunnel(context.Context, *DoCommonParam) (*FunnelData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetFdrQuality(context.Context, *DoCommonParam) (*DoFdrQualityData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetFclQuality(context.Context, *DoCommonParam) (*DoFclQualityData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetFdrFragment(context.Context, *DoCommonParam) (*DoFdrFragmentData, error) {
	return nil, nil
}

func (r *fakeDoDashboardRepo) GetFdrBandwidthTop(context.Context, *DoCommonParam) (*DoFdrBandwidthTopData, error) {
	return nil, nil
}

func TestDoDashboardUseCaseGetCloseTopForwardsFilterName(t *testing.T) {
	repo := &fakeDoDashboardRepo{}
	uc := NewDoDashboardUseCase(repo)

	_, err := uc.GetCloseTop(context.Background(), &dashboard_api.DoCoolTopRequest{
		FilterName:  "filter_a",
		EventNames:  "event_a,event_b",
		ProjectName: "project_x",
		CarTypes:    "SUV,MPV",
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})
	if err != nil {
		t.Fatalf("GetCloseTop returned error: %v", err)
	}

	wantParam := &DoCommonParam{
		FilterName:  "filter_a",
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	}
	if !reflect.DeepEqual(repo.closeTopParam, wantParam) {
		t.Fatalf("unexpected repo param: got %#v want %#v", repo.closeTopParam, wantParam)
	}
}
