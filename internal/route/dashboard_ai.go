package route

import (
	"github.com/gin-gonic/gin"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterAiDashboardService(s *service.AiDashboardService, cd *conf.Data) []GroupUrl {
	middleware := []gin.HandlerFunc{
		UMAuthMiddleware(cd.GetKeycloak().GetUrl(), cd.GetKeycloak().GetRealm()),
		UMUserResourcesMiddleware(cd.GetRuntime().GetDomain(), cd.GetGrpcClient().GetUmEndpoint()),
	}
	return []GroupUrl{
		{
			GroupAddr:  "/dashboard/v1/ai/",
			Middleware: middleware,
			Urls: []Url{
				// SSE 流式接口,需要直接写 gin.Context,走 EmptyHandlerFunc
				{EmptyHandlerFunc: s.StreamSummary, Path: "summary", Method: GET},
				{EmptyHandlerFunc: s.StreamChat, Path: "chat", Method: POST},
			},
		},
	}
}
