package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/go-kratos/kratos/v2/log"

	"fdi_data_board/internal/conf"
)

const (
	KafkaTopicFisProbe = "fis_channel"
)

var FDIKpiModuleName = []string{
	"filter_status",
	"fdr_status",
	"fcl_status",
}

type ProbeMessage struct {
	PlateNumber   string
	FdiType       string `json:"fdi_type"`
	SwVersion     string `json:"sw_version"`
	Timestamp     int64
	Source        string
	Type          string
	SchemaVersion string `json:"schema_version"`
	CacheMB       uint32 `json:"cache_mb"`
	FWL           uint32 `json:"fwl"`
	HWL           uint32 `json:"hwl"`
	Data          interface{}
}

type FFFTriggerEventItem struct {
	Uuid        string `json:"uuid"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	Name        string `json:"name"`
	FilterName  string `json:"filter_name"`
	CollectType string `json:"collect_type"`
	TriggerTime int64  `json:"trigger_time"`
	TriggerType string `json:"trigger_type"`
}

type FFFTriggerEvent struct {
	Events []*FFFTriggerEventItem `json:"events"`
}

type FDRTriggerEventItem struct {
	Uuid              string `json:"uuid"`
	Status            string `json:"status"`
	Detail            string `json:"detail"`
	EventName         string `json:"event_name"`
	BeginTimestampUts int64  `json:"begin_timestamp_uts"`
	DumpTimestamp     int64  `json:"dump_timestamp"`
	EndTimestampUts   int64  `json:"end_timestamp_uts"`
	TriggerTimestamp  int64  `json:"trigger_timestamp"`
	Dse               string `json:"dse"`
	SwVersion         string `json:"sw_version"`
}

type FDRTriggerEvent struct {
	Events []*FDRTriggerEventItem `json:"events"`
}

type FCLTriggerEventItem struct {
	Uuid             string `json:"uuid"`
	Status           string `json:"status"`
	Detail           string `json:"detail"`
	EventName        string `json:"event_name"`
	CompletePercent  int8   `json:"complete_percent"`
	PrefixStitch     string `json:"prefix_stitch"`
	TriggerTimestamp int64  `json:"trigger_timestamp"`
	TriggerSource    string `json:"trigger_source"` // FFF、Forever_log、FDC
}

type FCLTriggerEvent struct {
	Events []*FCLTriggerEventItem `json:"events"`
}

type ServerProbe struct {
	ctx context.Context
	cd  *conf.Data

	wg sync.WaitGroup

	wsHelper *ServerWsClientHelper
	wsRepo   WebsocketRepo
}

func NewServerProbe(ctx context.Context, cd *conf.Data, wsHelper *ServerWsClientHelper,
	wsRepo WebsocketRepo) *ServerProbe {
	return &ServerProbe{
		ctx: ctx,
		cd:  cd,

		wsHelper: wsHelper,

		wsRepo: wsRepo,
	}
}

func (s *ServerProbe) Start() error {
	//kafka消费任务
	//TODO 修改groupId
	groupId := fmt.Sprintf("%v-mmt_kratos_layout", s.cd.GetRuntime().GetEnv())
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.ClientID = groupId
	kafkaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	client, err := sarama.NewConsumerGroup(strings.Split(s.cd.GetKafka().GetBrokers(), ","), groupId, kafkaConfig)
	defer client.Close()
	if err != nil {
		log.Fatalf("Create Kafka consumer group failed: %v", err)
	}

	fisChannelHandleChan := make(chan []byte)
	consumer := Consumer{
		fisChannelHandleChan: fisChannelHandleChan,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			if err := client.Consume(s.ctx, []string{KafkaTopicFisProbe}, &consumer); err != nil {
				log.Fatalf("Error from kafka consumer: %v", err)
			}
			if s.ctx.Err() != nil {
				return
			}
		}
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case msg := <-fisChannelHandleChan: //fis 埋点
				s.handleFisProbe(msg)
			}
		}
	}()

	s.wg.Wait()
	log.Info("Server Probe stopping")
	return nil
}

func (s *ServerProbe) handleFisProbe(ss []byte) {
	type FisProbe struct {
		PlateNumber string `json:"plate_number"`
		FdiType     string `json:"fdi_type"`
		SwVersion   string `json:"sw_version"`
		ModuleName  string `json:"module_name"`
		ModuleJson  string `json:"module_json"`
	}

	type ProbeModuleJsonData struct {
		Data *ProbeMessage `json:"data"`
	}

	var probe *FisProbe
	if err := json.Unmarshal(ss, &probe); err != nil {
		log.Errorf("unmarshal Fis Probe payload failed: %v, msg:%v", err, string(ss))
		return
	}
	var mjd *ProbeModuleJsonData
	if err := json.Unmarshal([]byte(probe.ModuleJson), &mjd); err != nil {
		log.Errorf("unmarshal Fis ModuleJson failed: %v, msg:%v", err, string(ss))
		return
	}
	mprobe := mjd.Data
	mprobe.PlateNumber = probe.PlateNumber
	mprobe.FdiType = probe.FdiType
	mprobe.SwVersion = probe.SwVersion

	//只处理cfdi数据
	if mprobe.FdiType != "cfdi" {
		return
	}

	if probe.ModuleName == "filter_status" && mprobe.Source == "fff" &&
		mprobe.Type == "trigger" {
		//TODO 处理FFF数据
	}

	if probe.ModuleName == "fdr_status" && mprobe.Source == "fdr" && mprobe.Type == "trigger" {
		// TODO 处理FDR数据
	}

	if probe.ModuleName == "fcl_status" && mprobe.Source == "fcl" && mprobe.Type == "trigger" {
		//TODO 处理FCL数据
	}
}
