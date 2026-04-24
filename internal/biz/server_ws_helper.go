package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type ServerWsClientHelper struct {
	ctx context.Context

	queuedChan    chan *WsClient
	queuedClients map[string]*WsClient //请求进来，正在握手的ws client

	registerChan  chan *WsClient
	handleClients map[string]*WsClient //正在处理中的ws client

	unregisterChan chan *WsClient
}

func NewWsClientHelper(ctx context.Context) *ServerWsClientHelper {
	return &ServerWsClientHelper{
		ctx: ctx,

		registerChan:  make(chan *WsClient),
		handleClients: make(map[string]*WsClient),

		queuedChan:    make(chan *WsClient),
		queuedClients: make(map[string]*WsClient),

		unregisterChan: make(chan *WsClient),
	}
}

func (s *ServerWsClientHelper) QueuedChan() chan<- *WsClient {
	return s.queuedChan
}

func (s *ServerWsClientHelper) RegisterChan() chan<- *WsClient {
	return s.registerChan
}

func (s *ServerWsClientHelper) UnregisterChan() chan<- *WsClient {
	return s.unregisterChan
}

func (s *ServerWsClientHelper) Start() error {
	timer := time.NewTicker(time.Second * 10)
	defer timer.Stop()
	for {
		select {
		case <-s.ctx.Done():
			log.Info("Server Ws Client Helper stopping")
			return nil
		case client := <-s.queuedChan:
			s.queuedClients[client.Id()] = client
		case client := <-s.registerChan:
			delete(s.queuedClients, client.Id())
			s.handleClients[client.Id()] = client
		case client := <-s.unregisterChan:
			if _, ok := s.queuedClients[client.Id()]; ok {
				delete(s.queuedClients, client.Id())
				client.CloseWsClient()
			}
			if _, ok := s.handleClients[client.Id()]; ok {
				delete(s.handleClients, client.Id())
				client.CloseWsClient()
			}
		case <-timer.C:
			//定时关闭无效的websocket
			invalidIds := make([]string, 0)
			for id, client := range s.queuedClients {
				if client.IsHandshakeTimeout() {
					client.CloseWsClient()
					invalidIds = append(invalidIds, id)
				}
			}

			for _, id := range invalidIds {
				delete(s.queuedClients, id)
			}

		}
	}
}

func (s *ServerWsClientHelper) Stop() error {
	for id, client := range s.queuedClients {
		client.CloseWsClient()
		delete(s.queuedClients, id)
	}
	for id, client := range s.handleClients {
		client.CloseWsClient()
		delete(s.handleClients, id)
	}

	return nil
}

// ReportHelloWorld 通知前端Hello world
func (s *ServerWsClientHelper) ReportHelloWorld(_ string) {
	if len(s.handleClients) == 0 {
		return
	}

	for _, client := range s.handleClients {
		switch client.ConnType() {
		case ConnTypeHelloWorld:
			//TODO:发送给前端websocket的消息体
			replyBytes := []byte("hello world")

			if channel := client.SendChan(); channel != nil {
				channel <- replyBytes
			}
		}
	}
}
