package data

import (
	"reflect"
	"strings"
	"testing"

	"fdi_data_board/internal/biz"
)

func TestBuildCloseTopSQLUsesSelectedFilterName(t *testing.T) {
	sql, args := buildCloseTopSQL(&biz.DoCommonParam{
		FilterName:  "filter_a",
		EventNames:  []string{"event_a", "event_b"},
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
		tableFffCloseDailySummary,
		"filter_name = ?",
		"summary_grain = 'filter'",
		"GROUP BY filter_name",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("sql %q does not contain %q", sql, fragment)
		}
	}
	if strings.Contains(sql, "event_name IN") {
		t.Fatalf("sql %q should not filter close_top by event_name", sql)
	}
}
