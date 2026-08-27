package route

import "fdi_data_board/internal/service"

func RegisterIssueReportService(s *service.IssueReportService) []GroupUrl {
	return []GroupUrl{
		{
			GroupAddr: "/issue_report/v1/",
			Urls: []Url{
				{JsonHandlerFunc: s.Report, Path: "report", Method: POST},
				{JsonHandlerFunc: s.List, Path: "list", Method: GET},
				{JsonHandlerFunc: s.Summary, Path: "summary", Method: GET},
			},
		},
	}
}
