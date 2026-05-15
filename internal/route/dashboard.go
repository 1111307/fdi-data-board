package route

import (
	"fdi_data_board/internal/service"
)

func RegisterFoDashboardService(s *service.FoDashboardService) []GroupUrl {
	return []GroupUrl{
		{
			GroupAddr: "/dashboard/v1/fo/",
			Urls: []Url{
				{JsonHandlerFunc: s.GetDimensions, Path: "dimensions", Method: GET},
				{JsonHandlerFunc: s.ListFffRunning, Path: "detail/running", Method: GET},
				{JsonHandlerFunc: s.ListFffTrigger, Path: "detail/trigger", Method: GET},
				{JsonHandlerFunc: s.ListFffClose, Path: "detail/close", Method: GET},
				{JsonHandlerFunc: s.ListFdrTrigger, Path: "detail/fdr", Method: GET},
				{JsonHandlerFunc: s.ListFclTrigger, Path: "detail/fcl", Method: GET},
				{JsonHandlerFunc: s.ListUuidDetail, Path: "detail/uuid", Method: GET},
				{JsonHandlerFunc: s.GetCloseReason, Path: "diag/close_reason", Method: GET},
			},
		},
	}
}
