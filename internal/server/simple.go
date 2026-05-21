package server

import (
	"fdi_data_board/utils/transport/simple"
)

func NewSimpleServer(etl *ETLServer) *simple.Server {
	srv := simple.NewServer()
	srv.RegisterSimpleServer(etl)
	return srv
}
