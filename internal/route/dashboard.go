package route

import (
	"fdi_data_board/internal/service"
)

func RegisterDoDashboardService(s *service.DoDashboardService) []GroupUrl {
	return []GroupUrl{
		{
			GroupAddr: "/dashboard/v1/do/",
			Urls: []Url{
				{JsonHandlerFunc: s.GetOverview, Path: "overview", Method: GET},
				{JsonHandlerFunc: s.GetTrend, Path: "trend", Method: GET},
				{JsonHandlerFunc: s.GetFailReason, Path: "fail_reason", Method: GET},
				{JsonHandlerFunc: s.GetCoolTop, Path: "cool_top", Method: GET},
				{JsonHandlerFunc: s.GetSwVersion, Path: "sw_version", Method: GET},
				{JsonHandlerFunc: s.GetTriggerRank, Path: "trigger_rank", Method: GET},
			},
		},
	}
}

func RegisterFoDashboardService(s *service.FoDashboardService) []GroupUrl {
	return []GroupUrl{
		{
			// 公共接口，FO 和 DO 共用
			GroupAddr: "/dashboard/v1/",
			Urls: []Url{
				{JsonHandlerFunc: s.GetDimensions, Path: "dimensions", Method: GET},
				{JsonHandlerFunc: s.GetFunnel, Path: "diag/funnel", Method: GET},
			},
		},
		{
			GroupAddr: "/dashboard/v1/fo/",
			Urls: []Url{
				{JsonHandlerFunc: s.ListFffRunning, Path: "detail/running", Method: GET},
				{JsonHandlerFunc: s.ListFffTrigger, Path: "detail/trigger", Method: GET},
				{JsonHandlerFunc: s.ListFffClose, Path: "detail/close", Method: GET},
				{JsonHandlerFunc: s.ListFdrTrigger, Path: "detail/fdr", Method: GET},
				{JsonHandlerFunc: s.ListFclTrigger, Path: "detail/fcl", Method: GET},
				{JsonHandlerFunc: s.ListUuidDetail, Path: "detail/uuid", Method: GET},
				{JsonHandlerFunc: s.GetStageTrend, Path: "diag/stage_trend", Method: GET},
				{JsonHandlerFunc: s.GetCloseReason, Path: "diag/close_reason", Method: GET},
			},
		},
	}
}
