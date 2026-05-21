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
	foDashboardService *service.FoDashboardService,
	doDashboardService *service.DoDashboardService,
	etlService *service.EtlService,
) []GroupUrl {

	var routes []GroupUrl
	routes = append(routes, RegisterGreeterService(vehicleService, cd))
	routes = append(routes, RegisterFoDashboardService(foDashboardService)...)
	routes = append(routes, RegisterDoDashboardService(doDashboardService)...)
	routes = append(routes, RegisterEtlService(etlService)...)

	return routes
}
