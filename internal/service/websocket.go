package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/conf"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WsService struct {
	bg *biz.BackendServerGroup
	c  *conf.Data
}

// NewWsService new a websocket service.
func NewWsService(cd *conf.Data, bg *biz.BackendServerGroup) *WsService {
	return &WsService{
		bg: bg,
		c:  cd,
	}
}

func (s *WsService) ReceiveWs(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	client := biz.NewWsClient(s.c, conn, s.bg.WsClientHelper())
	go client.SendData()
	go client.ReceiveData()
}
