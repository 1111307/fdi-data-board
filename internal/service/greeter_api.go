package service

import (
	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"
	"fdi_data_board/api"
	"fdi_data_board/api/greeter"
	"fdi_data_board/internal/biz"
	"github.com/gin-gonic/gin"
)

// GreeterApiService is a greeter service.
type GreeterApiService struct {
	uc *biz.GreeterUsecase
}

// NewGreeterApiService new a greeter service.
func NewGreeterApiService(uc *biz.GreeterUsecase) *GreeterApiService {
	return &GreeterApiService{uc: uc}
}

// SayHello godoc
//	@Summary		Greeter
//	@Description	Greeter
//	@Tags			Greeter
//	@Accept			json
//	@Produce		json
//	@Security		OAuth2Password
//	@Param			name	path		string	true	"路径参数"
//	@Success		200		{object}	greeter_api.HelloReply
//	@Failure		400		{object}	greeter_api.HelloReply
//	@Router			/v1/greeter/{name} [get]
func (s *GreeterApiService) SayHello(ctx *gin.Context) (api.HttpResponse, error) {
	resp := &greeter_api.HelloReply{
		Code:    int32(gcode.CodeOK.Code()),
		Message: gcode.CodeOK.Message(),
	}
	req := &greeter_api.HelloRequest{}
	if err := ctx.ShouldBindUri(req); err != nil {
		resp.Code = int32(gcode.CodeInvalidRequest.Code())
		resp.Message = gcode.CodeInvalidRequest.Message()
		return resp, nil
	}

	g, err := s.uc.CreateGreeter(ctx, &biz.Greeter{User: req.Name})
	if err != nil {
		return nil, err
	}
	resp.Message = "Hello" + g.User
	return resp, nil
}
