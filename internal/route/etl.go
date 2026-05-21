package route

import (
	"fdi_data_board/internal/service"
)

func RegisterEtlService(s *service.EtlService) []GroupUrl {
	return []GroupUrl{
		{
			GroupAddr: "/internal/etl/",
			Urls: []Url{
				{JsonHandlerFunc: s.RunETL, Path: "run", Method: POST},
				{JsonHandlerFunc: s.GetStatus, Path: "status", Method: GET},
			},
		},
	}
}
