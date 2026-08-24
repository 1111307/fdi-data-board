package biz

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	dashboard_api "fdi_data_board/api/dashboard"
)

// ---------- fake 仓储:嵌入接口零值,只覆写 AI 快照用到的方法 ----------

type fakeFoRepo struct {
	FoDashboardRepo // 其余方法命中会 panic,测试即失败
}

func (fakeFoRepo) GetFffOverview(context.Context, *FffTriggerParam) (*FoFffOverviewData, error) {
	return &FoFffOverviewData{TriggerTotal: 14578, TriggerSuccess: 14110, TriggerFailed: 468,
		TriggerFilterCount: 277, CloseFilterCount: 118}, nil
}

func (fakeFoRepo) GetFffFailReason(context.Context, *FffTriggerParam) ([]*DoFailReasonItem, error) {
	return []*DoFailReasonItem{{Name: "FFF-cooldown", Value: 185}, {Name: "FFF-switch_off", Value: 112}}, nil
}

func (fakeFoRepo) GetRunningOverview(context.Context, *FffRunningParam) (*FoRunningOverviewData, error) {
	return &FoRunningOverviewData{RunningTotal: 9000, VehicleTotal: 1842, FilterCount: 326}, nil
}

func (fakeFoRepo) GetCloseReason(context.Context, *CloseReasonParam) ([]*CloseReasonItem, error) {
	return []*CloseReasonItem{{Name: "RunTimeGuard", Value: 218}}, nil
}

func (fakeFoRepo) GetStageTrend(context.Context, *StageTrendParam) (*StageTrendData, error) {
	return &StageTrendData{
		Dates: []string{"07-30", "07-31"},
		Fff:   []*StageTrendSeries{{Name: "success", Data: []int64{1782, 1908}}},
	}, nil
}

type fakeDoRepo struct {
	DoDashboardRepo
}

func (fakeDoRepo) GetMemTop(context.Context, *DoCommonParam) ([]*DoEventTopItem, error) {
	return []*DoEventTopItem{{EventName: "hard_brake", Count: 86}}, nil
}

func (fakeDoRepo) GetDiskTop(context.Context, *DoCommonParam) ([]*DoEventTopItem, error) {
	return []*DoEventTopItem{{EventName: "cut_in", Count: 42}}, nil
}

func (fakeDoRepo) GetQuotaTop(context.Context, *DoCommonParam) ([]*DoEventTopItem, error) {
	return []*DoEventTopItem{{EventName: "hard_brake", Count: 7}}, nil
}

func (fakeDoRepo) GetFdrQuality(context.Context, *DoCommonParam) (*DoFdrQualityData, error) {
	return &DoFdrQualityData{FdrTotal: 14110, FdrSuccess: 13900, TimeCostMsP95: 3200, TdMbP95: 512}, nil
}

func (fakeDoRepo) GetFclQuality(context.Context, *DoCommonParam) (*DoFclQualityData, error) {
	return &DoFclQualityData{UploadTotal: 13800, BagSizeP95: 3.2 * 1024 * 1024 * 1024}, nil
}

// fakeLlmRepo 按序吐出预设增量
type fakeLlmRepo struct {
	deltas []string
}

func (f *fakeLlmRepo) Enabled() bool { return true }

func (f *fakeLlmRepo) Model() string { return "kimi-k3-test" }
func (f *fakeLlmRepo) MaxTokens() int64 { return 16384 }

func (f *fakeLlmRepo) ChatStreamEx(_ context.Context, params anthropic.MessageNewParams, onEvent func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error) {
	return nil, fmt.Errorf("not implemented in summary fake")
}

func (f *fakeLlmRepo) ChatStream(_ context.Context, _, _ string, onDelta func(string)) error {
	for _, d := range f.deltas {
		onDelta(d)
	}
	return nil
}

type disabledLlmRepo struct{}

func (disabledLlmRepo) Enabled() bool { return false }
func (disabledLlmRepo) Model() string { return "" }
func (disabledLlmRepo) MaxTokens() int64 { return 16384 }
func (disabledLlmRepo) ChatStream(context.Context, string, string, func(string)) error {
	return nil
}
func (disabledLlmRepo) ChatStreamEx(context.Context, anthropic.MessageNewParams, func(anthropic.MessageStreamEventUnion)) (*anthropic.Message, error) {
	return nil, fmt.Errorf("disabled")
}

func newAiUcForTest(llm LlmRepo) *AiDashboardUseCase {
	return NewAiDashboardUseCase(NewFoDashboardUseCase(fakeFoRepo{}), NewDoDashboardUseCase(fakeDoRepo{}), llm)
}

func TestStreamSummaryLlmModeForwardsDeltasInOrder(t *testing.T) {
	uc := newAiUcForTest(&fakeLlmRepo{deltas: []string{"## 总体概览", "\n", "触发总数 14,578"}})

	var got []string
	err := uc.StreamSummary(context.Background(), &dashboard_api.AiSummaryRequest{
		StartDt: "2026-07-22", EndDt: "2026-07-28",
	}, func(delta string) error {
		got = append(got, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSummary error: %v", err)
	}
	if strings.Join(got, "") != "## 总体概览\n触发总数 14,578" {
		t.Fatalf("deltas = %q", got)
	}
}

func TestStreamSummaryLocalModeGeneratesDeterministicSummary(t *testing.T) {
	uc := newAiUcForTest(disabledLlmRepo{})

	var sb strings.Builder
	err := uc.StreamSummary(context.Background(), &dashboard_api.AiSummaryRequest{
		StartDt: "2026-07-22", EndDt: "2026-07-28", EventNames: "hard_brake",
	}, func(delta string) error {
		sb.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSummary error: %v", err)
	}

	summary := sb.String()
	for _, want := range []string{"统计模式", "查询范围:2026-07-22 ~ 2026-07-28", "hard_brake",
		"触发总数 14,578", "cooldown(185", "落盘 14,110", "上传总数 13,800"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("local summary missing %q, got:\n%s", want, summary)
		}
	}
}

func TestBuildAiUserPromptContainsSnapshot(t *testing.T) {
	uc := newAiUcForTest(&fakeLlmRepo{})
	snap := uc.buildSnapshot(context.Background(), &dashboard_api.AiSummaryRequest{StartDt: "2026-07-22", EndDt: "2026-07-28"})

	prompt, err := buildAiUserPrompt(snap)
	if err != nil {
		t.Fatalf("buildAiUserPrompt error: %v", err)
	}
	// 快照 JSON 里应包含聚合结果(带 json tag)与查询范围
	for _, want := range []string{`"trigger_total":14578`, `"query_range"`, `"2026-07-22"`, `FFF-cooldown`} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func (fakeFoRepo) ListFffTrigger(_ context.Context, _ *FffTriggerParam) ([]*FffTriggerItem, int64, error) {
	return []*FffTriggerItem{
		{Dt: "2026-08-22", Uuid: "u1", EventName: "mid_highbeam_on", AnonymousId: "a1", TriggerType: "periodic", CarType: "M03"},
		{Dt: "2026-08-23", Uuid: "u2", EventName: "mid_highbeam_on", AnonymousId: "a2", TriggerType: "event", CarType: "M05"},
	}, 235, nil
}

func (fakeFoRepo) ListUuidDetail(_ context.Context, _ *UuidDetailParam) ([]*UuidDetailItem, int64, error) {
	return []*UuidDetailItem{
		{Dt: "2026-08-23", AnonymousId: "a1", EventName: "mid_highbeam_on", Uuid: "u1"},
	}, 139, nil
}
