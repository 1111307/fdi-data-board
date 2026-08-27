package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

type IssueReportService struct {
	uc *biz.IssueReportUseCase
}

func NewIssueReportService(uc *biz.IssueReportUseCase) *IssueReportService {
	return &IssueReportService{uc: uc}
}

type IssueReportPayload struct {
	App        string            `json:"app"`
	Module     string            `json:"module"`
	Env        string            `json:"env"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Level      string            `json:"level"`
	TraceID    string            `json:"trace_id"`
	Extra      map[string]string `json:"extra"`
	OccurredAt string            `json:"occurred_at"`
}

type IssueReportResponse struct {
	dashboard_api.BaseResponse
	ReportID string `json:"report_id,omitempty"`
}

type IssueReportQueryRequest struct {
	App       string `form:"app"`
	Module    string `form:"module"`
	Env       string `form:"env"`
	Level     string `form:"level"`
	Keyword   string `form:"keyword"`
	StartTime int64  `form:"start_time"`
	EndTime   int64  `form:"end_time"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type IssueReportRecordItem struct {
	ReportID   string            `json:"report_id"`
	App        string            `json:"app"`
	Module     string            `json:"module"`
	Env        string            `json:"env"`
	Level      string            `json:"level"`
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	TraceID    string            `json:"trace_id"`
	Extra      map[string]string `json:"extra"`
	ClientIP   string            `json:"client_ip"`
	UserAgent  string            `json:"user_agent"`
	OccurredAt time.Time         `json:"occurred_at"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type IssueReportAppSummaryItem struct {
	App              string    `json:"app"`
	Total            int64     `json:"total"`
	CriticalCount    int64     `json:"critical_count"`
	HighCount        int64     `json:"high_count"`
	MediumCount      int64     `json:"medium_count"`
	LowCount         int64     `json:"low_count"`
	LatestOccurredAt time.Time `json:"latest_occurred_at"`
}

type IssueReportSummaryData struct {
	Total            int64                        `json:"total"`
	CriticalCount    int64                        `json:"critical_count"`
	HighCount        int64                        `json:"high_count"`
	MediumCount      int64                        `json:"medium_count"`
	LowCount         int64                        `json:"low_count"`
	AppCount         int64                        `json:"app_count"`
	ModuleCount      int64                        `json:"module_count"`
	LatestOccurredAt time.Time                    `json:"latest_occurred_at"`
	AppStats         []*IssueReportAppSummaryItem `json:"app_stats"`
}

type IssueReportListResponse struct {
	dashboard_api.BaseResponse
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
	List     []*IssueReportRecordItem `json:"list"`
}

type IssueReportSummaryResponse struct {
	dashboard_api.BaseResponse
	Data *IssueReportSummaryData `json:"data"`
}

// Report godoc
//
//	@Summary	接收 FDI Cloud 高危问题上报并记录
//	@Tags		IssueReport
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		service.IssueReportPayload	true	"高危问题内容"
//	@Success	200		{object}	service.IssueReportResponse
//	@Router		/issue_report/v1/report [POST]
func (s *IssueReportService) Report(c *gin.Context) (api.HttpResponse, error) {
	var req IssueReportPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("IssueReport bind json error: %v", err)
		return issueReportInvalidResponse("invalid request body: " + err.Error()), nil
	}

	occurredAt, err := parseIssueOccurredAt(req.OccurredAt)
	if err != nil {
		return issueReportInvalidResponse(err.Error()), nil
	}

	record, err := s.uc.Report(c.Request.Context(), &biz.IssueReportRequest{
		App:        req.App,
		Module:     req.Module,
		Env:        req.Env,
		Title:      req.Title,
		Content:    req.Content,
		Level:      req.Level,
		TraceID:    req.TraceID,
		Extra:      req.Extra,
		ClientIP:   c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		OccurredAt: occurredAt,
	})
	if err != nil {
		if errors.Is(err, biz.ErrIssueReportRequestRequired) || errors.Is(err, biz.ErrIssueReportTitleRequired) {
			return issueReportInvalidResponse(err.Error()), nil
		}
		log.Errorf("IssueReport save error: %v", err)
		return &IssueReportResponse{
			BaseResponse: dashboard_api.BaseResponse{
				Code:    int32(gcode.CodeInternalError.Code()),
				Message: "failed to save issue report: " + err.Error(),
			},
		}, nil
	}

	return &IssueReportResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeOK.Code()),
			Message: "ok",
		},
		ReportID: record.ReportID,
	}, nil
}

// List godoc
//
//	@Summary	查询 FDI Cloud 高危问题上报明细
//	@Tags		IssueReport
//	@Produce	json
//	@Param		app			query		string	false	"应用"
//	@Param		module		query		string	false	"模块"
//	@Param		env			query		string	false	"环境"
//	@Param		level		query		string	false	"风险等级"
//	@Param		keyword		query		string	false	"关键词"
//	@Param		start_time	query		int		false	"开始时间 Unix 秒或毫秒"
//	@Param		end_time	query		int		false	"结束时间 Unix 秒或毫秒"
//	@Param		page		query		int		false	"页码"
//	@Param		page_size	query		int		false	"每页数量"
//	@Success	200			{object}	service.IssueReportListResponse
//	@Router		/issue_report/v1/list [GET]
func (s *IssueReportService) List(c *gin.Context) (api.HttpResponse, error) {
	var req IssueReportQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		return issueReportListErrorResponse(int32(gcode.CodeInvalidParameter.Code()), err.Error()), nil
	}

	result, err := s.uc.List(c.Request.Context(), issueReportQueryFromRequest(&req))
	if err != nil {
		log.Errorf("IssueReport list error: %v", err)
		return issueReportListErrorResponse(int32(gcode.CodeInternalError.Code()), "failed to list issue reports"), nil
	}

	return &IssueReportListResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeOK.Code()),
			Message: "ok",
		},
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
		List:     issueReportRecordItems(result.List),
	}, nil
}

// Summary godoc
//
//	@Summary	查询 FDI Cloud 高危问题上报聚合统计
//	@Tags		IssueReport
//	@Produce	json
//	@Param		app			query		string	false	"应用"
//	@Param		module		query		string	false	"模块"
//	@Param		env			query		string	false	"环境"
//	@Param		level		query		string	false	"风险等级"
//	@Param		keyword		query		string	false	"关键词"
//	@Param		start_time	query		int		false	"开始时间 Unix 秒或毫秒"
//	@Param		end_time	query		int		false	"结束时间 Unix 秒或毫秒"
//	@Success	200			{object}	service.IssueReportSummaryResponse
//	@Router		/issue_report/v1/summary [GET]
func (s *IssueReportService) Summary(c *gin.Context) (api.HttpResponse, error) {
	var req IssueReportQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		return issueReportSummaryErrorResponse(int32(gcode.CodeInvalidParameter.Code()), err.Error()), nil
	}

	result, err := s.uc.Summary(c.Request.Context(), issueReportQueryFromRequest(&req))
	if err != nil {
		log.Errorf("IssueReport summary error: %v", err)
		return issueReportSummaryErrorResponse(int32(gcode.CodeInternalError.Code()), "failed to summarize issue reports"), nil
	}

	return &IssueReportSummaryResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeOK.Code()),
			Message: "ok",
		},
		Data: issueReportSummaryData(result),
	}, nil
}

func issueReportInvalidResponse(message string) *IssueReportResponse {
	return &IssueReportResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeInvalidRequest.Code()),
			Message: message,
		},
	}
}

func issueReportListErrorResponse(code int32, message string) *IssueReportListResponse {
	return &IssueReportListResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: code, Message: message},
		List:         []*IssueReportRecordItem{},
	}
}

func issueReportSummaryErrorResponse(code int32, message string) *IssueReportSummaryResponse {
	return &IssueReportSummaryResponse{
		BaseResponse: dashboard_api.BaseResponse{Code: code, Message: message},
		Data:         &IssueReportSummaryData{AppStats: []*IssueReportAppSummaryItem{}},
	}
}

func issueReportRecordItems(records []*biz.IssueReportRecord) []*IssueReportRecordItem {
	items := make([]*IssueReportRecordItem, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		items = append(items, &IssueReportRecordItem{
			ReportID:   record.ReportID,
			App:        record.App,
			Module:     record.Module,
			Env:        record.Env,
			Level:      record.Level,
			Title:      record.Title,
			Content:    record.Content,
			TraceID:    record.TraceID,
			Extra:      record.Extra,
			ClientIP:   record.ClientIP,
			UserAgent:  record.UserAgent,
			OccurredAt: record.OccurredAt,
			CreatedAt:  record.CreatedAt,
			UpdatedAt:  record.UpdatedAt,
		})
	}
	return items
}

func issueReportSummaryData(summary *biz.IssueReportSummary) *IssueReportSummaryData {
	if summary == nil {
		return &IssueReportSummaryData{AppStats: []*IssueReportAppSummaryItem{}}
	}
	appStats := make([]*IssueReportAppSummaryItem, 0, len(summary.AppStats))
	for _, item := range summary.AppStats {
		if item == nil {
			continue
		}
		appStats = append(appStats, &IssueReportAppSummaryItem{
			App:              item.App,
			Total:            item.Total,
			CriticalCount:    item.CriticalCount,
			HighCount:        item.HighCount,
			MediumCount:      item.MediumCount,
			LowCount:         item.LowCount,
			LatestOccurredAt: item.LatestOccurredAt,
		})
	}
	return &IssueReportSummaryData{
		Total:            summary.Total,
		CriticalCount:    summary.CriticalCount,
		HighCount:        summary.HighCount,
		MediumCount:      summary.MediumCount,
		LowCount:         summary.LowCount,
		AppCount:         summary.AppCount,
		ModuleCount:      summary.ModuleCount,
		LatestOccurredAt: summary.LatestOccurredAt,
		AppStats:         appStats,
	}
}

func issueReportQueryFromRequest(req *IssueReportQueryRequest) *biz.IssueReportQuery {
	if req == nil {
		return &biz.IssueReportQuery{}
	}
	return &biz.IssueReportQuery{
		App:       req.App,
		Module:    req.Module,
		Env:       req.Env,
		Level:     req.Level,
		Keyword:   req.Keyword,
		StartTime: unixQueryTime(req.StartTime),
		EndTime:   unixQueryTime(req.EndTime),
		Page:      req.Page,
		PageSize:  req.PageSize,
	}
}

func unixQueryTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value > 10_000_000_000 {
		return time.UnixMilli(value)
	}
	return time.Unix(value, 0)
}

func parseIssueOccurredAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid occurred_at: %s", raw)
}
