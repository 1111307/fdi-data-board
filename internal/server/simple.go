package server

import (
	"fdi_data_board/internal/biz"
	"fdi_data_board/utils/transport/simple"
)

// NewSimpleServer new a simple server.
func NewSimpleServer(bsGroup *biz.BackendServerGroup) *simple.Server {
	srv := simple.NewServer()
	srv.RegisterSimpleServer(bsGroup)
	return srv
}
