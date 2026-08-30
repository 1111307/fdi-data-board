package data

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildAggEventConditionUsesAllRowWhenNoEvents(t *testing.T) {
	cond, args := buildAggEventCondition(nil)

	if cond != "event_name = ?" {
		t.Fatalf("cond = %q, want event_name = ?", cond)
	}
	if !reflect.DeepEqual(args, []interface{}{"__ALL__"}) {
		t.Fatalf("args = %#v, want __ALL__", args)
	}
}

func TestBuildAggEventConditionUsesInWhenEventsProvided(t *testing.T) {
	cond, args := buildAggEventCondition([]string{"eventA", "eventB"})

	if cond != "event_name IN (?,?)" {
		t.Fatalf("cond = %q, want event_name IN (?,?)", cond)
	}
	if !reflect.DeepEqual(args, []interface{}{"eventA", "eventB"}) {
		t.Fatalf("args = %#v, want two events", args)
	}
}

func TestBuildAggCommonWhereIncludesDimensionFilters(t *testing.T) {
	where, args := buildAggCommonWhere("overview", "filter-a", []string{"eventA"}, "project-a", []string{"carA", "carB"}, "2026-08-01", "2026-08-07")

	for _, want := range []string{
		"dt BETWEEN ? AND ?",
		"summary_grain = ?",
		"event_name = ?",
		"filter_name = ?",
		"project_name = ?",
		"car_type IN (?,?)",
	} {
		if !strings.Contains(where, want) {
			t.Fatalf("where = %q, missing %q", where, want)
		}
	}

	wantArgs := []interface{}{"2026-08-01", "2026-08-07", "overview", "eventA", "filter-a", "project-a", "carA", "carB"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}
