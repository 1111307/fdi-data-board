package server

import (
	"fdi_data_board/utils/transport/simple"
)

func NewSimpleServer() *simple.Server {
	return simple.NewServer()
}
