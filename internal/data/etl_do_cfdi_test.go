package data

import (
	"reflect"
	"strings"
	"testing"
)

func TestInsertCfdiDailySQLMapsFffDetailToRequestedTags(t *testing.T) {
	fffCase := normalizeSQLWhitespace(extractFffDetailCase(t))

	for _, fragment := range []string{
		"WHEN fff_detail = 'check_is_no_need_cooldown' THEN 'check_is_no_need_cooldown'",
		"WHEN fff_detail IN ('check_drm_quota', 'check_drm_quota_weight') THEN 'check_drm_quota'",
		"WHEN fff_detail = 'check_need_acquire_data' THEN 'check_need_acquire_data'",
		"WHEN fff_detail = 'check_not_reach_trigger_maximum' THEN 'check_not_reach_trigger_maximum'",
		"WHEN fff_detail = 'query cloud DISCARD, detail:Filter quota exceeded' THEN 'query cloud DISCARD, detail:Filter quota exceeded'",
		"WHEN fff_detail = 'query cloud DISCARD, detail:EventName is in blacklist' THEN 'query cloud DISCARD, detail:EventName is in blacklist'",
		"WHEN fff_detail LIKE 'Bag invalid:%' THEN 'bag_invalid'",
		"WHEN fff_detail LIKE 'event_name do not recognized%' THEN 'event_not_recognized'",
		"WHEN fff_detail LIKE 'tls%' THEN 'tls_error'",
	} {
		if !strings.Contains(fffCase, fragment) {
			t.Fatalf("FFF detail mapping does not contain %q in:\n%s", fragment, fffCase)
		}
	}

	for _, oldTag := range []string{"'cooldown'", "'drm_quota'", "'acquire_data'", "'trigger_maximum'", "'quota_exceeded'", "'event_in_blacklist'"} {
		if strings.Contains(fffCase, "THEN "+oldTag) {
			t.Fatalf("FFF detail mapping still emits old tag %s in:\n%s", oldTag, fffCase)
		}
	}
}

func TestInsertCfdiDailySQLMapsFdrDetailToRequestedTags(t *testing.T) {
	fdrCase := normalizeSQLWhitespace(extractFdrDetailCase(t))

	for _, fragment := range []string{
		"WHEN fdr_detail = 'because of full gc' THEN 'because of full gc'",
		"WHEN fdr_detail = 'mem pool water line' THEN 'mem pool water line'",
		"WHEN fdr_detail = 'Disk overrun' THEN 'Disk overrun'",
		"WHEN fdr_detail = 'Exceeds the maximum number of files' THEN 'Exceeds the maximum number of files'",
		"WHEN fdr_detail LIKE 'Bag invalid:%' THEN 'bag_invalid'",
		"WHEN fdr_detail LIKE 'Dump bag dir missing%' THEN 'bag_dir_missing'",
		"WHEN fdr_detail LIKE 'event_name do not recognized%' THEN 'event_not_recognized'",
		"WHEN fdr_detail = 'unauthorized' THEN 'unauthorized'",
	} {
		if !strings.Contains(fdrCase, fragment) {
			t.Fatalf("FDR detail mapping does not contain %q in:\n%s", fragment, fdrCase)
		}
	}

	for _, oldTag := range []string{"'memory'", "'disk'"} {
		if strings.Contains(fdrCase, "THEN "+oldTag) {
			t.Fatalf("FDR detail mapping still emits old tag %s in:\n%s", oldTag, fdrCase)
		}
	}
}

func TestInsertCfdiDailySQLMapsFclDetailToRequestedTags(t *testing.T) {
	fclCase := normalizeSQLWhitespace(extractFclDetailCase(t))

	for _, fragment := range []string{
		"WHEN fcl_detail = 'query cloud DISCARD, detail:Filter quota exceeded' THEN 'query cloud DISCARD, detail:Filter quota exceeded'",
		"WHEN fcl_detail = 'reach upload limit' THEN 'reach upload limit'",
		"WHEN fcl_detail = 'query cloud DISCARD, detail:EventName is in blacklist' THEN 'query cloud DISCARD, detail:EventName is in blacklist'",
		"WHEN fcl_detail LIKE 'geofence forbidden%' THEN 'geofence_error'",
		"WHEN fcl_detail = 'unexpected geofence cause' THEN 'unexpected geofence cause'",
		"WHEN fcl_detail LIKE 'tls%' THEN 'tls_error'",
		"WHEN fcl_detail = 'bag not exist' THEN 'bag not exist'",
		"WHEN fcl_detail = 'meta file lost' THEN 'meta file lost'",
		"WHEN fcl_detail = 'meta file empty' THEN 'meta file empty'",
		"WHEN fcl_detail = 'unexpected bag_upload_query cause' THEN 'unexpected bag_upload_query cause'",
		"WHEN fcl_detail = 's3 upload force quit' THEN 's3 upload force quit'",
		"WHEN fcl_detail = 'create socket failed' THEN 'create socket failed'",
		"WHEN fcl_detail = 'http request failed' THEN 'http request failed'",
		"WHEN fcl_detail = 'transfer dns failed' THEN 'transfer dns failed'",
	} {
		if !strings.Contains(fclCase, fragment) {
			t.Fatalf("FCL detail mapping does not contain %q in:\n%s", fragment, fclCase)
		}
	}

	for _, oldTag := range []string{"'quota_exceeded'", "'reach_upload_limit'", "'event_in_blacklist'", "'bag_missing'", "'upload_error'", "'network_error'"} {
		if strings.Contains(fclCase, "THEN "+oldTag) {
			t.Fatalf("FCL detail mapping still emits old tag %s in:\n%s", oldTag, fclCase)
		}
	}
}

func TestFffStageTrendNamesUseRequestedDetailTags(t *testing.T) {
	want := []string{
		"success",
		"check_is_no_need_cooldown",
		"check_drm_quota",
		"check_need_acquire_data",
		"check_not_reach_trigger_maximum",
		"bag_invalid",
		"event_not_recognized",
		"tls_error",
		"query cloud DISCARD, detail:Filter quota exceeded",
		"query cloud DISCARD, detail:EventName is in blacklist",
		"other",
	}

	if got := fffStageTrendNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected FFF stage trend names:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestFdrStageTrendNamesUseRequestedDetailTags(t *testing.T) {
	want := []string{
		"success",
		"because of full gc",
		"mem pool water line",
		"Disk overrun",
		"Exceeds the maximum number of files",
		"bag_invalid",
		"bag_dir_missing",
		"event_not_recognized",
		"unauthorized",
		"other",
	}

	if got := fdrStageTrendNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected FDR stage trend names:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestFclStageTrendNamesUseRequestedDetailTags(t *testing.T) {
	want := []string{
		"success",
		"query cloud DISCARD, detail:Filter quota exceeded",
		"reach upload limit",
		"query cloud DISCARD, detail:EventName is in blacklist",
		"geofence_error",
		"unexpected geofence cause",
		"tls_error",
		"bag not exist",
		"meta file lost",
		"meta file empty",
		"unexpected bag_upload_query cause",
		"s3 upload force quit",
		"create socket failed",
		"http request failed",
		"transfer dns failed",
		"other",
	}

	if got := fclStageTrendNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected FCL stage trend names:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestFdrStageFailedConditionRequiresDiscard(t *testing.T) {
	want := "fff_status != 'discard' AND fdr_status = 'discard' AND fcl_status = ''"
	if got := fdrStageFailedCondition(); got != want {
		t.Fatalf("unexpected FDR failed condition: got %q want %q", got, want)
	}
}

func extractFffDetailCase(t *testing.T) string {
	t.Helper()

	start := strings.Index(insertCfdiDailySQL, "WHEN fff_detail = 'check_is_no_need_cooldown'")
	if start < 0 {
		t.Fatal("FFF detail CASE start not found")
	}
	end := strings.Index(insertCfdiDailySQL[start:], "END AS fff_detail_tag")
	if end < 0 {
		t.Fatal("FFF detail CASE end not found")
	}
	return insertCfdiDailySQL[start : start+end]
}

func extractFclDetailCase(t *testing.T) string {
	t.Helper()

	start := strings.Index(insertCfdiDailySQL, "WHEN fcl_detail")
	if start < 0 {
		t.Fatal("FCL detail CASE start not found")
	}
	end := strings.Index(insertCfdiDailySQL[start:], "END AS fcl_detail_tag")
	if end < 0 {
		t.Fatal("FCL detail CASE end not found")
	}
	return insertCfdiDailySQL[start : start+end]
}

func normalizeSQLWhitespace(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func extractFdrDetailCase(t *testing.T) string {
	t.Helper()

	start := strings.Index(insertCfdiDailySQL, "WHEN fdr_detail")
	if start < 0 {
		t.Fatal("FDR detail CASE start not found")
	}
	end := strings.Index(insertCfdiDailySQL[start:], "END AS fdr_detail_tag")
	if end < 0 {
		t.Fatal("FDR detail CASE end not found")
	}
	return insertCfdiDailySQL[start : start+end]
}
