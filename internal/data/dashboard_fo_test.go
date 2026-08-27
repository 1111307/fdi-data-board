package data

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"fdi_data_board/internal/biz"
)

func TestBuildFffTriggerReasonWhereUsesAllEventWhenEventFilterIsEmpty(t *testing.T) {
	where, args := buildFffTriggerReasonWhere(&biz.FffTriggerParam{
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-01",
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
	})

	wantArgs := []interface{}{"2026-06-01", "2026-06-01", aggAllValue, aggAllValue, "project_x", "SUV", "MPV"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	for _, fragment := range []string{
		"dt BETWEEN ? AND ?",
		"event_name = ?",
		"filter_name = ?",
		"project_name = ?",
		"car_type IN (?,?)",
	} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("where %q does not contain %q", where, fragment)
		}
	}
}

func TestBuildFffTriggerReasonWhereUsesSelectedEvents(t *testing.T) {
	where, args := buildFffTriggerReasonWhere(&biz.FffTriggerParam{
		StartDt:    "2026-06-01",
		EndDt:      "2026-06-01",
		EventNames: []string{"event_a", "event_b"},
	})

	wantArgs := []interface{}{"2026-06-01", "2026-06-01", "event_a", "event_b", aggAllValue}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	if !strings.Contains(where, "event_name IN (?,?)") {
		t.Fatalf("where %q does not use selected event filter", where)
	}
	if strings.Contains(where, "event_name = ?") {
		t.Fatalf("where %q also contains all-event equality", where)
	}
}

func TestBuildFffFailReasonSQLUsesSummaryWhenFilterNameIsPresent(t *testing.T) {
	sql, args := buildFffFailReasonSQL(&biz.FffTriggerParam{
		FilterName: "filter_x",
		StartDt:    "2026-06-01",
		EndDt:      "2026-06-01",
	})

	if !strings.Contains(sql, tableFffTriggerDailySummary) {
		t.Fatalf("sql %q does not query summary table", sql)
	}
	if strings.Contains(sql, "dwd_cfdi_basic_fff_trigger ") || strings.Contains(sql, "dwd_cfdi_basic_fff_trigger`") {
		t.Fatalf("sql %q should not query detail table", sql)
	}
	wantArgs := []interface{}{"2026-06-01", "2026-06-01", aggAllValue, aggAllValue}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
}

func TestBuildCloseReasonWhereUsesSelectedFilterName(t *testing.T) {
	where, args := buildCloseReasonWhere(&biz.CloseReasonParam{
		FilterName:  "filter_a",
		ProjectName: "project_x",
		CarTypes:    []string{"SUV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})

	wantArgs := []interface{}{"2026-06-01", "2026-06-03", "filter_a", "project_x", "SUV"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	for _, fragment := range []string{
		"dt BETWEEN ? AND ?",
		"filter_name = ?",
		"project_name = ?",
		"car_type = ?",
	} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("where %q does not contain %q", where, fragment)
		}
	}
	if strings.Contains(where, "event_name IN") {
		t.Fatalf("where %q should not filter close reason by event_name", where)
	}
}

func TestBuildFffOverviewSQLUsesSelectedEventWithoutAllEventConflict(t *testing.T) {
	sql, args := buildFffOverviewSQL(&biz.FffTriggerParam{
		StartDt:    "2026-06-01",
		EndDt:      "2026-06-01",
		EventNames: []string{"hotupdate_filter_navi_action"},
	})

	// trigger_filter_count / close_filter_count 两个子查询各自带一组 (日期,日期,事件) 参数,
	// 主查询再带 (日期,日期,grain,事件)
	wantArgs := []interface{}{
		"2026-06-01", "2026-06-01", "hotupdate_filter_navi_action",
		"2026-06-01", "2026-06-01", "hotupdate_filter_navi_action",
		"2026-06-01", "2026-06-01", "overview", "hotupdate_filter_navi_action",
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	if !strings.Contains(sql, tableFffTriggerDailySummary) {
		t.Fatalf("sql %q does not query summary table", sql)
	}
	for _, fragment := range []string{"summary_grain = ?", "event_name = ?"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("sql %q does not contain %q", sql, fragment)
		}
	}
	if strings.Contains(sql, "event_name = '"+aggAllValue+"'") {
		t.Fatalf("sql %q also forces all-event rows", sql)
	}
}

func TestBuildAggCommonWhereUsesSelectedEventForStageTrend(t *testing.T) {
	where, args := buildAggCommonWhere(
		"stage_reason",
		"",
		[]string{"hotupdate_filter_navi_action"},
		"",
		nil,
		"2026-06-01",
		"2026-06-01",
	)

	wantArgs := []interface{}{"2026-06-01", "2026-06-01", "stage_reason", "hotupdate_filter_navi_action"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	if !strings.Contains(where, "summary_grain = ?") || !strings.Contains(where, "event_name = ?") {
		t.Fatalf("where %q does not contain summary/event filters", where)
	}
	if strings.Contains(where, aggAllValue) {
		t.Fatalf("where %q should not hard-code all-event rows", where)
	}
}

func TestBuildTrendDatesUsesFullRequestRange(t *testing.T) {
	dates := buildTrendDates("2026-06-01", "2026-06-03")

	want := []string{"2026-06-01", "2026-06-02", "2026-06-03"}
	if !reflect.DeepEqual(dates, want) {
		t.Fatalf("unexpected dates: got %#v want %#v", dates, want)
	}
}

func TestAssembleFffStageValsBackfillsMissingFailureIntoOther(t *testing.T) {
	dates := []string{"2026-06-01"}
	success := map[string]int64{"2026-06-01": 90}
	failed := map[string]int64{"2026-06-01": 10}
	rows := []*stageTrendRow{
		{Name: "cooldown", Dt: mustDate(t, "2026-06-01"), Count: 3},
	}
	names := []string{"success", "cooldown", "other"}

	vals := assembleFffStageVals(dates, success, failed, rows, names)

	want := map[string][]int64{"2026-06-01": []int64{90, 3, 7}}
	if !reflect.DeepEqual(vals, want) {
		t.Fatalf("unexpected vals: got %#v want %#v", vals, want)
	}
}

func mustDate(t *testing.T, raw string) time.Time {
	t.Helper()
	dt, err := time.Parse("2006-01-02", raw)
	if err != nil {
		t.Fatal(err)
	}
	return dt
}

func TestBuildRunningOverviewSQLUsesAdsForSwitchOnVehicles(t *testing.T) {
	sql, args := buildRunningOverviewSQL(&biz.FffRunningParam{
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-01",
		ProjectName: "project_x",
		CarTypes:    []string{"SUV"},
	})

	wantArgs := []interface{}{"2026-06-01", "2026-06-01", aggAllValue, "project_x", "SUV", "2026-06-01", "2026-06-01", "project_x", "SUV"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	for _, fragment := range []string{
		"SUM(running_switch_on_vehicle_count)",
		tableVehicleDailySummaryAgg,
		tableFffRunningDailySummary,
		"event_name = ?",
		"summary_grain = 'filter'",
		"car_type = ?",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("sql %q does not contain %q", sql, fragment)
		}
	}
	if strings.Contains(sql, "FROM dwd_cfdi_basic_fff_running ") || strings.Contains(sql, "FROM fdi.dwd_cfdi_basic_fff_running ") {
		t.Fatalf("sql %q should not query running detail table", sql)
	}
}

func TestBuildRunningOverviewSQLUsesEventsAsAdsEventsAndRunningFilterNames(t *testing.T) {
	sql, args := buildRunningOverviewSQL(&biz.FffRunningParam{
		EventNames: []string{"event_a", "event_b"},
		StartDt:    "2026-06-01",
		EndDt:      "2026-06-03",
	})

	wantArgs := []interface{}{
		"2026-06-01", "2026-06-03", "event_a", "event_b",
		"2026-06-01", "2026-06-03", "event_a", "event_b",
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	if strings.Contains(sql, tableFffTriggerDailySummary) {
		t.Fatalf("sql %q should not use trigger summary for running event filters", sql)
	}
	for _, fragment := range []string{
		"event_name IN (?,?)",
		"filter_name IN (?,?)",
		"summary_grain = 'filter'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("sql %q does not contain %q", sql, fragment)
		}
	}
}

func TestBuildFffRunningTrendWhereForwardsResolvedFilterName(t *testing.T) {
	where, args := buildFffRunningTrendWhere(&biz.FffRunningTrendParam{
		FilterName:  "hotupdate_filter_operator_a",
		ProjectName: "project_x",
		CarTypes:    []string{"SUV", "MPV"},
		StartDt:     "2026-06-01",
		EndDt:       "2026-06-03",
	})

	wantArgs := []interface{}{"hotupdate_filter_operator_a", "2026-06-01", "2026-06-03", "project_x", "SUV", "MPV"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: got %#v want %#v", args, wantArgs)
	}
	for _, fragment := range []string{
		"event_name = ?",
		"project_name = ?",
		"car_type IN (?,?)",
	} {
		if !strings.Contains(where, fragment) {
			t.Fatalf("where %q does not contain %q", where, fragment)
		}
	}
}
