package service

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

type AlertService struct {
	uc *biz.AlertUseCase
}

func NewAlertService(uc *biz.AlertUseCase) *AlertService {
	return &AlertService{uc: uc}
}

// GrafanaWebhook godoc
//
//	@Summary	接收 Grafana 告警 webhook 并发送飞书通知
//	@Tags		Alert
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		biz.GrafanaAlertPayload	true	"Grafana 告警内容"
//	@Success	200		{object}	dashboard_api.BaseResponse
//	@Router		/alert/grafana/webhook [POST]
func (s *AlertService) GrafanaWebhook(c *gin.Context) (api.HttpResponse, error) {
	var payload biz.GrafanaAlertPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Errorf("GrafanaWebhook bind json error: %v", err)
		return &dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeInvalidRequest.Code()),
			Message: "invalid request body: " + err.Error(),
		}, nil
	}

	log.Infof("GrafanaWebhook received: title=%q state=%q alerts=%d", payload.Title, payload.State, len(payload.Alerts))

	if err := s.uc.SendGrafanaAlert(c.Request.Context(), &payload); err != nil {
		log.Errorf("GrafanaWebhook send feishu error: %v", err)
		return &dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeInternalError.Code()),
			Message: "failed to send feishu notification: " + err.Error(),
		}, nil
	}

	return &dashboard_api.BaseResponse{
		Code:    int32(gcode.CodeOK.Code()),
		Message: "ok",
	}, nil
}
