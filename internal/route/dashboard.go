package route

import (
	"github.com/gin-gonic/gin"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterDoDashboardService(s *service.DoDashboardService, cd *conf.Data) []GroupUrl {
	middleware := []gin.HandlerFunc{
		UMAuthMiddleware(cd.GetKeycloak().GetUrl(), cd.GetKeycloak().GetRealm()),
		UMUserResourcesMiddleware(cd.GetRuntime().GetDomain(), cd.GetGrpcClient().GetUmEndpoint()),
	}
	return []GroupUrl{
		{
			GroupAddr:  "/dashboard/v1/do/",
			Middleware: middleware,
			Urls: []Url{
				{JsonHandlerFunc: s.GetOverview, Path: "overview", Method: GET},
				{JsonHandlerFunc: s.GetTrend, Path: "trend", Method: GET},
				{JsonHandlerFunc: s.GetFailReason, Path: "fail_reason", Method: GET},
				{JsonHandlerFunc: s.GetCoolTop, Path: "cool_top", Method: GET},
				{JsonHandlerFunc: s.GetSwVersion, Path: "sw_version", Method: GET},
				{JsonHandlerFunc: s.GetTriggerRank, Path: "trigger_rank", Method: GET},
				{JsonHandlerFunc: s.GetProjectCar, Path: "project_car", Method: GET},
				{JsonHandlerFunc: s.GetTopVehicles, Path: "top_vehicles", Method: GET},
				{JsonHandlerFunc: s.GetAnomalyVehicles, Path: "anomaly_vehicles", Method: GET},
				{JsonHandlerFunc: s.GetActiveTrend, Path: "active_trend", Method: GET},
				{JsonHandlerFunc: s.GetNetSpeed, Path: "net_speed", Method: GET},
				{JsonHandlerFunc: s.GetFclBw, Path: "fcl_bw", Method: GET},
				{JsonHandlerFunc: s.GetQuotaTop, Path: "quota_top", Method: GET},
				{JsonHandlerFunc: s.GetProjectEvent, Path: "project_event", Method: GET},
				{JsonHandlerFunc: s.GetMemTop, Path: "mem_top", Method: GET},
				{JsonHandlerFunc: s.GetDiskTop, Path: "disk_top", Method: GET},
				{JsonHandlerFunc: s.GetCloseTop, Path: "close_top", Method: GET},
				{JsonHandlerFunc: s.GetDoFunnel, Path: "funnel", Method: GET},
				{JsonHandlerFunc: s.GetFdrQuality, Path: "fdr_quality", Method: GET},
				{JsonHandlerFunc: s.GetFclQuality, Path: "fcl_quality", Method: GET},
				{JsonHandlerFunc: s.GetFdrFragment, Path: "fdr_fragment", Method: GET},
			},
		},
	}
}

func RegisterFoDashboardService(s *service.FoDashboardService, cd *conf.Data) []GroupUrl {
	middleware := []gin.HandlerFunc{
		UMAuthMiddleware(cd.GetKeycloak().GetUrl(), cd.GetKeycloak().GetRealm()),
		UMUserResourcesMiddleware(cd.GetRuntime().GetDomain(), cd.GetGrpcClient().GetUmEndpoint()),
	}
	return []GroupUrl{
		{
			// 公共接口，FO 和 DO 共用
			GroupAddr:  "/dashboard/v1/",
			Middleware: middleware,
			Urls: []Url{
				{JsonHandlerFunc: s.GetDimensions, Path: "dimensions", Method: GET},
				{JsonHandlerFunc: s.GetFunnel, Path: "diag/funnel", Method: GET},
			},
		},
		{
			GroupAddr:  "/dashboard/v1/fo/",
			Middleware: middleware,
			Urls: []Url{
				{JsonHandlerFunc: s.ListFffRunning, Path: "detail/running", Method: GET},
				{JsonHandlerFunc: s.ListFffTrigger, Path: "detail/trigger", Method: GET},
				{JsonHandlerFunc: s.ListFffClose, Path: "detail/close", Method: GET},
				{JsonHandlerFunc: s.ListFdrTrigger, Path: "detail/fdr", Method: GET},
				{JsonHandlerFunc: s.ListFclTrigger, Path: "detail/fcl", Method: GET},
				{JsonHandlerFunc: s.ListUuidDetail, Path: "detail/uuid", Method: GET},
				{JsonHandlerFunc: s.GetStageTrend, Path: "diag/stage_trend", Method: GET},
				{JsonHandlerFunc: s.GetCloseReason, Path: "diag/close_reason", Method: GET},
				{JsonHandlerFunc: s.GetFffRunningTrend, Path: "running/trend", Method: GET},
				{JsonHandlerFunc: s.GetRunningOverview, Path: "running/overview", Method: GET},
				{JsonHandlerFunc: s.GetFffOverview, Path: "fff/overview", Method: GET},
				{JsonHandlerFunc: s.GetFffFailReason, Path: "fff/fail_reason", Method: GET},
			},
		},
	}
}
