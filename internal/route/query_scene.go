package route

import (
	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterQuerySceneService(s *service.QuerySceneService, ds *service.DatasourceService, sg *service.SceneGroupService, cd *conf.Data) []GroupUrl {
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
				{JsonHandlerFunc: ds.ListDatasources, Path: "datasources/list", Method: GET},
				{JsonHandlerFunc: sg.GetGroupByPageKey, Path: "scene_groups/page", Method: GET},
				{JsonHandlerFunc: sg.GetDimensionValues, Path: "scene_groups/dimensions", Method: GET},
			},
		},
		// 管理端
		{
			GroupAddr: "/query_scene/v1/admin/",
			Urls: []Url{
				{JsonHandlerFunc: s.CreateScene, Path: "scenes/create", Method: POST},
				{JsonHandlerFunc: s.UpdateScene, Path: "scenes/update", Method: POST},
				{JsonHandlerFunc: s.DeleteScene, Path: "scenes/delete", Method: POST},
				{JsonHandlerFunc: s.CreateParam, Path: "params/create", Method: POST},
				{JsonHandlerFunc: s.UpdateParam, Path: "params/update", Method: POST},
				{JsonHandlerFunc: s.SaveWidget, Path: "widgets/save", Method: POST},
				{JsonHandlerFunc: s.PreviewWidget, Path: "widgets/preview", Method: POST},
				{JsonHandlerFunc: ds.CreateDatasource, Path: "datasources/create", Method: POST},
				{JsonHandlerFunc: ds.UpdateDatasource, Path: "datasources/update", Method: POST},
				{JsonHandlerFunc: ds.DeleteDatasource, Path: "datasources/delete", Method: POST},
				{JsonHandlerFunc: ds.GetSchemaTables, Path: "datasources/schema/tables", Method: POST},
				{JsonHandlerFunc: ds.GetSchemaColumns, Path: "datasources/schema/columns", Method: POST},
				{JsonHandlerFunc: sg.ListGroups, Path: "scene_groups/list", Method: GET},
				{JsonHandlerFunc: sg.CreateGroup, Path: "scene_groups/create", Method: POST},
				{JsonHandlerFunc: sg.UpdateGroup, Path: "scene_groups/update", Method: POST},
				{JsonHandlerFunc: sg.DeleteGroup, Path: "scene_groups/delete", Method: POST},
			},
		},
	}
}
