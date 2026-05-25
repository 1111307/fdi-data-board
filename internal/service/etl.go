package service

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"

	"fdi_data_board/api"
	dashboard_api "fdi_data_board/api/dashboard"
	"fdi_data_board/internal/biz"
)

type EtlService struct {
	uc *biz.ETLUseCase
}

func NewEtlService(uc *biz.ETLUseCase) *EtlService {
	return &EtlService{uc: uc}
}

// GetStatus godoc
//
//	@Summary		查询 ETL 任务日志
//	@Tags			ETL
//	@Produce		json
//	@Param			limit	query	int	false	"返回条数，默认 30"
//	@Success		200		{object}	dashboard_api.EtlStatusResponse
//	@Router			/internal/etl/status [GET]
func (s *EtlService) GetStatus(c *gin.Context) (api.HttpResponse, error) {
	limit := 30
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	logs, err := s.uc.ListLogs(c.Request.Context(), limit)
	if err != nil {
		return &dashboard_api.EtlStatusResponse{
			BaseResponse: dashboard_api.BaseResponse{
				Code:    int32(gcode.CodeInternalError.Code()),
				Message: err.Error(),
			},
		}, nil
	}

	return &dashboard_api.EtlStatusResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeOK.Code()),
			Message: "ok",
		},
		List: logs,
	}, nil
}
