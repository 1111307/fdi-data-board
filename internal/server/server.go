package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/wire"

	"fdi_data_board/internal/conf"
	"fdi_data_board/internal/route"
	"fdi_data_board/internal/service"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewGRPCServer, NewSimpleServer, NewAllHttpServer, NewETLServer)

func NewAllHttpServer(c *conf.Server, d *conf.Data, logger log.Logger,
	urls []route.GroupUrl, greeter *service.GreeterService) []*http.Server {
	return []*http.Server{
		NewHTTPServer(c, greeter, logger),
		NewGinServer(c, d, logger, urls),
	}
}
