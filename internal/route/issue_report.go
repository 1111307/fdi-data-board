package route

import "fdi_data_board/internal/service"

func RegisterIssueReportService(s *service.IssueReportService) []GroupUrl {
	return []GroupUrl{
		{
			GroupAddr: "/issue_report/v1/",
			Urls: []Url{
				{JsonHandlerFunc: s.Report, Path: "report", Method: POST},
			},
		},
	}
}
