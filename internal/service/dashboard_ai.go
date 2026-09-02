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
//	@Success	200				{string}	string	"text/event-stream, thinking/delta/done/error 事件"
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

	err := s.uc.StreamSummary(c.Request.Context(), &req, func(kind, text string) error {
		if kind == "thinking" {
			return writeSseEvent(c.Writer, "thinking", text)
		}
		return writeSseDelta(c.Writer, text)
	})
	if err != nil {
		log.Errorf("StreamSummary error: %v, req: %+v", err, req)
		writeSseError(c.Writer, 500, "AI 总结生成失败: "+err.Error())
		return
	}
	writeSseDone(c.Writer)
}

// StreamChat godoc
//
//	@Summary	AI 问答(SSE 流式,带工具调用与人工确认)
//	@Tags		AiDashboard
//	@Accept		json
//	@Produce	text/event-stream
//	@Security	OAuth2Password
//	@Param		body	body		dashboard_api.AiChatRequest	true	"提问与会话历史"
//	@Success	200		{string}	string						"text/event-stream: delta/tool_call/tool_result/clarify/done/error"
//	@Router		/dashboard/v1/ai/chat [POST]
func (s *AiDashboardService) StreamChat(c *gin.Context) {
	var req dashboard_api.AiChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeSseError(c.Writer, 400, err.Error())
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flushWriter(c.Writer)

	history := make([]biz.AiChatHistoryMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msg := biz.AiChatHistoryMessage{Role: m.Role, Text: m.Text}
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, biz.AiChatToolCall{ID: tc.ID, Name: tc.Name, Args: tc.Args})
		}
		for _, tr := range m.ToolResults {
			msg.ToolResults = append(msg.ToolResults, biz.AiChatToolResult{ID: tr.ID, Content: tr.StringContent(), IsError: tr.IsError})
		}
		history = append(history, msg)
	}

	err := s.uc.StreamChat(c.Request.Context(), req.Question, history, func(ev biz.ChatEvent) error {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Type, payload)
		if err != nil {
			return err
		}
		flushWriter(c.Writer)
		// done 帧落一行观测日志:grep "ai_chat_done" 统计调用量/P95 延迟/轮次与 token 成本分布
		if ev.Type == "done" {
			log.Infof("ai_chat_done stop=%s rounds=%d duration_ms=%d input_tokens=%d output_tokens=%d question=%q",
				ev.Stop, ev.Rounds, ev.DurationMs, ev.InputTokens, ev.OutputTokens, req.Question)
		}
		return nil
	})
	if err != nil {
		log.Errorf("StreamChat error: %v, question: %s", err, req.Question)
		writeSseError(c.Writer, 500, "AI 问答失败: "+err.Error())
		return
	}
}

// ---------- SSE 帧写入 ----------

// writeSseEvent 写自定义事件的 {"text": ...} 帧(thinking 等与 chat 同构的简单文本事件)
func writeSseEvent(w http.ResponseWriter, event, text string) error {
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
	if err != nil {
		return err
	}
	flushWriter(w)
	return nil
}

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
