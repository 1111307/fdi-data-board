package route

import (
	"github.com/google/wire"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

var ProviderSet = wire.NewSet(RegisterHttpService)

func RegisterHttpService(
	cd *conf.Data,
	vehicleService *service.GreeterApiService,
	querySceneService *service.QuerySceneService,
	datasourceService *service.DatasourceService,
	sceneGroupService *service.SceneGroupService,
	foDashboardService *service.FoDashboardService,
	doDashboardService *service.DoDashboardService,
) []GroupUrl {

	var routes []GroupUrl
	routes = append(routes, RegisterGreeterService(vehicleService, cd))
	routes = append(routes, RegisterQuerySceneService(querySceneService, datasourceService, sceneGroupService, cd)...)
	routes = append(routes, RegisterFoDashboardService(foDashboardService)...)
	routes = append(routes, RegisterDoDashboardService(doDashboardService)...)

	return routes
}
