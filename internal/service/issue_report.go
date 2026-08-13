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

func issueReportInvalidResponse(message string) *IssueReportResponse {
	return &IssueReportResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeInvalidRequest.Code()),
			Message: message,
		},
	}
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
