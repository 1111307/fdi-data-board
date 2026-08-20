package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

// AiDashboardService 看板 AI 总结服务(SSE 流式)
type AiDashboardService struct {
	uc *biz.AiDashboardUseCase
}

func NewAiDashboardService(uc *biz.AiDashboardUseCase) *AiDashboardService {
	return &AiDashboardService{uc: uc}
}

// StreamSummary godoc
//
//	@Summary	AI 看板数据总结(SSE 流式)
//	@Tags		AiDashboard
//	@Produce	text/event-stream
//	@Security	OAuth2Password
//	@Param		event_names		query		string	false	"事件名，多选逗号分隔"
//	@Param		project_name	query		string	false	"项目名称"
//	@Param		car_types		query		string	false	"车型，多选逗号分隔"
//	@Param		start_dt		query		string	false	"开始日期 YYYY-MM-DD"
//	@Param		end_dt			query		string	false	"结束日期 YYYY-MM-DD"
//	@Success	200				{string}	string	"text/event-stream, delta/done/error 事件"
//	@Router		/dashboard/v1/ai/summary [GET]
func (s *AiDashboardService) StreamSummary(c *gin.Context) {
	var req dashboard_api.AiSummaryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeSseError(c.Writer, 400, err.Error())
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flushWriter(c.Writer)

	err := s.uc.StreamSummary(c.Request.Context(), &req, func(delta string) error {
		return writeSseDelta(c.Writer, delta)
	})
	if err != nil {
		log.Errorf("StreamSummary error: %v, req: %+v", err, req)
		writeSseError(c.Writer, 500, "AI 总结生成失败: "+err.Error())
		return
	}
	writeSseDone(c.Writer)
}

// ---------- SSE 帧写入 ----------

func writeSseDelta(w http.ResponseWriter, text string) error {
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: delta\ndata: %s\n\n", payload)
	if err != nil {
		return err
	}
	flushWriter(w)
	return nil
}

func writeSseDone(w http.ResponseWriter) {
	payload, _ := json.Marshal(map[string]interface{}{"code": 0})
	_, _ = fmt.Fprintf(w, "event: done\ndata: %s\n\n", payload)
	flushWriter(w)
}

func writeSseError(w http.ResponseWriter, code int, message string) {
	payload, _ := json.Marshal(map[string]interface{}{"code": code, "message": message})
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload)
	flushWriter(w)
}

func flushWriter(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
