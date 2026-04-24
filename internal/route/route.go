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
) []GroupUrl {

	var routes []GroupUrl
	routes = append(routes, RegisterGreeterService(vehicleService, cd))
	routes = append(routes, RegisterQuerySceneService(querySceneService, cd)...)

	return routes
}
