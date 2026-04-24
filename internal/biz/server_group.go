package biz

import (
	"context"
	"strconv"
	"strings"

	"github.com/IBM/sarama"
	"golang.org/x/sync/errgroup"

	"fdi_data_board/internal/conf"
)

type BackendServerGroup struct {
	config *conf.Data

	ctx    context.Context
	cancel context.CancelFunc

	eg       *errgroup.Group
	wsHelper *ServerWsClientHelper
	wsServer *ServerWebsocket

	probeServer *ServerProbe
}

func NewBackendServerGroup(c *conf.Data,
	wsRepo WebsocketRepo,
) *BackendServerGroup {
	ctx, cancel := context.WithCancel(context.Background())
	wsHelper := NewWsClientHelper(ctx)
	return &BackendServerGroup{
		config: c,

		ctx:    ctx,
		cancel: cancel,
		eg:     new(errgroup.Group),

		wsHelper: wsHelper,
		wsServer: NewServerWebsocket(ctx, c, wsHelper, wsRepo),

		probeServer: NewServerProbe(ctx, c, wsHelper, wsRepo),
	}
}

func (bg *BackendServerGroup) WsClientHelper() *ServerWsClientHelper {
	return bg.wsHelper
}

func (bg *BackendServerGroup) Start(ctx context.Context) error {
	bg.eg.Go(func() error {
		return bg.wsHelper.Start()
	})

	bg.eg.Go(func() error {
		return bg.wsServer.Start()
	})

	if on, _ := strconv.ParseBool(bg.config.GetKafka().GetFisProbeOn()); on {
		bg.eg.Go(func() error {
			return bg.probeServer.Start()
		})
	}

	return nil
}

func (bg *BackendServerGroup) Stop(ctx context.Context) error {
	bg.wsHelper.Stop()

	bg.cancel()

	return bg.eg.Wait()
}

// kafka consumer
type Consumer struct {
	fisChannelHandleChan chan<- []byte
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if session.Context().Err() != nil || message == nil {
				return nil
			}

			if strings.Contains(message.Topic, KafkaTopicFisProbe) {
				consumer.fisChannelHandleChan <- message.Value
			}

			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}
