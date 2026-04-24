package route

import (
	"github.com/gin-gonic/gin"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterGreeterService(s *service.GreeterApiService, cd *conf.Data) GroupUrl {
	return GroupUrl{
		GroupAddr: "/v1/greeter",
		Urls: []Url{
			{
				JsonHandlerFunc: s.SayHello,
				Path:            ":name",
				Method:          GET,
				Middleware:      nil,
			},
		},
		Middleware: []gin.HandlerFunc{
			UMAuthMiddleware(cd.GetKeycloak().GetUrl(), cd.GetKeycloak().GetRealm()),
			UMUserResourcesMiddleware(cd.GetRuntime().GetDomain(), cd.GetGrpcClient().GetUmEndpoint()),
		},
	}
}
