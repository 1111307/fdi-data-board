package route

import (
	"fdi_data_board/internal/service"
)

func RegisterAlertService(s *service.AlertService) []GroupUrl {
	return []GroupUrl{
		{
			GroupAddr: "/alert/",
			Urls: []Url{
				{JsonHandlerFunc: s.GrafanaWebhook, Path: "grafana/webhook", Method: POST},
			},
		},
	}
}
