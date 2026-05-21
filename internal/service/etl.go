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

// RunETL godoc
//
//	@Summary		手动触发指定日期的 ETL
//	@Tags			ETL
//	@Produce		json
//	@Param			dt	query	string	true	"日期，格式 2006-01-02"
//	@Success		200	{object}	dashboard_api.EtlRunResponse
//	@Router			/internal/etl/run [POST]
func (s *EtlService) RunETL(c *gin.Context) (api.HttpResponse, error) {
	dt := c.Query("dt")
	if dt == "" {
		return &dashboard_api.EtlRunResponse{
			BaseResponse: dashboard_api.BaseResponse{
				Code:    int32(gcode.CodeInvalidParameter.Code()),
				Message: "dt is required",
			},
		}, nil
	}

	cnt, err := s.uc.RunForDate(c.Request.Context(), dt)
	if err != nil {
		return &dashboard_api.EtlRunResponse{
			BaseResponse: dashboard_api.BaseResponse{
				Code:    int32(gcode.CodeInternalError.Code()),
				Message: err.Error(),
			},
		}, nil
	}

	return &dashboard_api.EtlRunResponse{
		BaseResponse: dashboard_api.BaseResponse{
			Code:    int32(gcode.CodeOK.Code()),
			Message: "ok",
		},
		Cnt: cnt,
	}, nil
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
