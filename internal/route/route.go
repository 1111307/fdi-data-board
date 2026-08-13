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
	alertService *service.AlertService,
	issueReportService *service.IssueReportService,
	reconcileService *service.ReconcileService,
) []GroupUrl {

	var routes []GroupUrl
	routes = append(routes, RegisterGreeterService(vehicleService, cd))
	routes = append(routes, RegisterFoDashboardService(foDashboardService, cd)...)
	routes = append(routes, RegisterDoDashboardService(doDashboardService, cd)...)
	routes = append(routes, RegisterEtlService(etlService)...)
	routes = append(routes, RegisterAlertService(alertService)...)
	routes = append(routes, RegisterIssueReportService(issueReportService)...)
	routes = append(routes, RegisterReconcileService(reconcileService, cd)...)

	return routes
}
