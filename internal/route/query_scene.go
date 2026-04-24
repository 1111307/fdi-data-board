package route

import (
	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterQuerySceneService(s *service.QuerySceneService, cd *conf.Data) []GroupUrl {
	return []GroupUrl{
		// 用户端
		{
			GroupAddr: "/query_scene/v1/",
			Urls: []Url{
				{JsonHandlerFunc: s.ListScenes, Path: "scenes/list", Method: GET},
				{JsonHandlerFunc: s.GetSceneDetail, Path: "scenes/detail", Method: GET},
				{JsonHandlerFunc: s.ExecuteScene, Path: "scenes/execute", Method: POST},
				{JsonHandlerFunc: s.ListParams, Path: "params/list", Method: GET},
				{JsonHandlerFunc: s.ListWidgets, Path: "widgets/list", Method: GET},
			},
		},
		// 管理端
		{
			GroupAddr: "/query_scene/v1/admin/",
			Urls: []Url{
				{JsonHandlerFunc: s.CreateScene, Path: "scenes/create", Method: POST},
				{JsonHandlerFunc: s.UpdateScene, Path: "scenes/update", Method: POST},
				{JsonHandlerFunc: s.DeleteScene, Path: "scenes/delete", Method: POST},
				{JsonHandlerFunc: s.SaveParam, Path: "params/save", Method: POST},
				{JsonHandlerFunc: s.SaveWidget, Path: "widgets/save", Method: POST},
				{JsonHandlerFunc: s.PreviewWidget, Path: "widgets/preview", Method: POST},
			},
		},
	}
}
