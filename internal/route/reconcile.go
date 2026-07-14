package route

import (
	"github.com/gin-gonic/gin"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/service"
)

func RegisterReconcileService(s *service.ReconcileService, cd *conf.Data) []GroupUrl {
	middleware := []gin.HandlerFunc{
		UMAuthMiddleware(cd.GetKeycloak().GetUrl(), cd.GetKeycloak().GetRealm()),
		UMUserResourcesMiddleware(cd.GetRuntime().GetDomain(), cd.GetGrpcClient().GetUmEndpoint()),
	}
	return []GroupUrl{
		{
			GroupAddr:  "/dashboard/v1/reconcile/",
			Middleware: middleware,
			Urls: []Url{
				{JsonHandlerFunc: s.GetOverview, Path: "overview", Method: GET},
				{JsonHandlerFunc: s.GetTrend, Path: "trend", Method: GET},
				{JsonHandlerFunc: s.GetModule, Path: "module", Method: GET},
				{JsonHandlerFunc: s.GetProject, Path: "project", Method: GET},
				{JsonHandlerFunc: s.GetDecodeStatus, Path: "decode_status", Method: GET},
				{JsonHandlerFunc: s.ListMd5, Path: "md5", Method: GET},
				{JsonHandlerFunc: s.GetMd5Detail, Path: "md5_detail", Method: GET},
				{JsonHandlerFunc: s.ListEventList, Path: "event_list", Method: GET},
				{JsonHandlerFunc: s.GetRecordConsistency, Path: "record_consistency", Method: GET},
				{JsonHandlerFunc: s.GetUuidSource, Path: "uuid_source", Method: GET},
				{JsonHandlerFunc: s.GetFailureSummary, Path: "failure_summary", Method: GET},
				{JsonHandlerFunc: s.GetPipelineTree, Path: "pipeline_tree", Method: GET},
			},
		},
	}
}
