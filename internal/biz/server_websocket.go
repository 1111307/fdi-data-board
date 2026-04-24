package biz

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	"fdi_data_board/internal/conf"
)

const KafkaTopicWsNotify = "ws_notify"

// WebsocketRepo is a Vehicle repo.
type WebsocketRepo interface {
	SubWsNotifyMsg(ctx context.Context) *redis.PubSub
	PublishWsNotifyMsg(ctx context.Context, topic, message string) error
}

type ServerWebsocket struct {
	ctx context.Context
	cd  *conf.Data

	wg     sync.WaitGroup
	wsRepo WebsocketRepo

	wsHelper *ServerWsClientHelper
}

func NewServerWebsocket(ctx context.Context, cd *conf.Data, wsHelper *ServerWsClientHelper, wsRepo WebsocketRepo) *ServerWebsocket {
	return &ServerWebsocket{
		ctx: ctx,
		cd:  cd,

		wsHelper: wsHelper,
		wsRepo:   wsRepo,
	}
}

func (s *ServerWebsocket) Start() error {
	psub := s.wsRepo.SubWsNotifyMsg(s.ctx)
	defer psub.Close()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case msg := <-psub.Channel():
				s.handleWsNotify(msg)
			}
		}
	}()

	s.wg.Wait()
	log.Info("Server Websocket stopping")
	return nil
}

func (s *ServerWebsocket) handleWsNotify(msg *redis.Message) {
	prefix := fmt.Sprintf("%v.%v.", s.cd.GetRuntime().GetEnv(), KafkaTopicWsNotify)
	key := strings.TrimPrefix(msg.Channel, prefix)
	switch key {
	case ReporterTypeHelloWorld:
		sessionId := msg.Payload
		s.wsHelper.ReportHelloWorld(sessionId)
	default:
		return
	}
}
