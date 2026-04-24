package route

import (
	"github.com/gin-gonic/gin"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterQuerySceneService(s *service.QuerySceneService, cd *conf.Data) []GroupUrl {
	authMiddleware := []gin.HandlerFunc{
		UMAuthMiddleware(cd.GetKeycloak().GetUrl(), cd.GetKeycloak().GetRealm()),
		UMUserResourcesMiddleware(cd.GetRuntime().GetDomain(), cd.GetGrpcClient().GetUmEndpoint()),
	}

	return []GroupUrl{
		{
			GroupAddr: "/query_scene/v1/",
			Urls: []Url{
				{
					JsonHandlerFunc: s.ListScenes,
					Path:            "scenes",
					Method:          GET,
				},
				{
					JsonHandlerFunc: s.GetSceneDetail,
					Path:            "scenes/:id",
					Method:          GET,
				},
				{
					JsonHandlerFunc: s.ExecuteScene,
					Path:            "scenes/:id/execute",
					Method:          POST,
				},
			},
			Middleware: authMiddleware,
		},
		{
			GroupAddr: "/query_scene/v1/admin/",
			Urls: []Url{
				{
					JsonHandlerFunc: s.CreateScene,
					Path:            "scenes",
					Method:          POST,
				},
				{
					JsonHandlerFunc: s.UpdateScene,
					Path:            "scenes/:id",
					Method:          PUT,
				},
				{
					JsonHandlerFunc: s.DeleteScene,
					Path:            "scenes/:id",
					Method:          DELETE,
				},
				{
					JsonHandlerFunc: s.PreviewWidget,
					Path:            "scenes/:id/preview",
					Method:          POST,
				},
			},
			Middleware: authMiddleware,
		},
	}
}
