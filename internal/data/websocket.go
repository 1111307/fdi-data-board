package data

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/conf"
)

var _ biz.WebsocketRepo = (*websocketRepo)(nil)

type websocketRepo struct {
	*baseRepo
	cd *conf.Data
}

// NewWebsocketRepo .
func NewWebsocketRepo(data *Data, c *conf.Data) biz.WebsocketRepo {
	return &websocketRepo{
		baseRepo: &baseRepo{
			data: data,
		},
		cd: c,
	}
}

func (r *websocketRepo) PublishWsNotifyMsg(ctx context.Context, topic, message string) error {
	return r.data.rDB.Publish(ctx, fmt.Sprintf("%v.%v.%v", r.cd.GetRuntime().GetEnv(), biz.KafkaTopicWsNotify, topic), message).Err()
}

func (r *websocketRepo) SubWsNotifyMsg(ctx context.Context) *redis.PubSub {
	return r.data.rDB.PSubscribe(ctx, fmt.Sprintf("%v.%v.*", r.cd.GetRuntime().GetEnv(), biz.KafkaTopicWsNotify))
}
