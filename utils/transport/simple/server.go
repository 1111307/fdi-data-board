package simple

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server = (*Server)(nil)
)

// ServerOption is simple server option.
type ServerOption func(o *Server)

type SimpleServerInterface interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Server is a simple server wrapper.
type Server struct {
	baseCtx context.Context

	ssv SimpleServerInterface
}

// NewServer creates a simple server by options.
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx: context.Background(),
	}
	for _, o := range opts {
		o(srv)
	}
	return srv
}

func (s *Server) RegisterSimpleServer(si SimpleServerInterface) {
	s.ssv = si
}

// Start start the simple server.
func (s *Server) Start(ctx context.Context) error {
	s.baseCtx = ctx
	log.Info("[simple] server starting...")
	if s.ssv != nil {
		return s.ssv.Start(ctx)
	}

	log.Warn("[simple] server is None")
	return nil
}

// Stop stop the simple server.
func (s *Server) Stop(ctx context.Context) error {
	log.Info("[simple] server stopping")

	if s.ssv != nil {
		return s.ssv.Stop(ctx)
	}

	log.Warn("[simple] server is None")
	return nil
}
