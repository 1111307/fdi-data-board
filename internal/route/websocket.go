package route

import (
	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterWebsocketService(s *service.WsService, cd *conf.Data) GroupUrl {
	return GroupUrl{
		GroupAddr: "/",
		Urls: []Url{
			{
				EmptyHandlerFunc: s.ReceiveWs,
				Path:             "ws",
				Method:           WS,
				Middleware:       nil,
			},
		},
	}
}
