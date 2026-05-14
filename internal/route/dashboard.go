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
			},
		},
	}
}
