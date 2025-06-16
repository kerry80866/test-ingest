package mq

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaConsumerConf 定义 Kafka 消费者配置（适合单 topic 场景）
// 所有时间相关参数单位均为毫秒
type KafkaConsumerConf struct {
	Name                  string   `json:"name" yaml:"name"`                                         // 用于标识用途，如 event/balance
	Brokers               []string `json:"brokers" yaml:"brokers"`                                   // Kafka 集群 broker 地址列表
	Topic                 string   `json:"topic" yaml:"topic"`                                       // 订阅的 topic 名称
	GroupId               string   `json:"group_id" yaml:"group_id"`                                 // 消费者组 ID，同组实现高可用
	SessionTimeoutMs      int      `json:"session_timeout_ms" yaml:"session_timeout_ms"`             // 会话超时时间（ms）
	HeartbeatIntervalMs   int      `json:"heartbeat_interval_ms" yaml:"heartbeat_interval_ms"`       // 心跳间隔（ms）
	ReadTimeoutMs         int      `json:"read_timeout_ms" yaml:"read_timeout_ms"`                   // 拉取消息超时时间（ms）
	ReconnectBackoffMs    int      `json:"reconnect_backoff_ms" yaml:"reconnect_backoff_ms"`         // 第一次重连延迟
	ReconnectBackoffMaxMs int      `json:"reconnect_backoff_max_ms" yaml:"reconnect_backoff_max_ms"` // 最大重连间隔
	RetryBackoffMs        int      `json:"retry_backoff_ms" yaml:"retry_backoff_ms"`                 // 拉取失败重试间隔
}

type Handler func(msg *kafka.Message)

// KafkaConsumer 封装了 kafka-go 的单 topic 消费逻辑
type KafkaConsumer struct {
	Consumer *kafka.Consumer
	Conf     *KafkaConsumerConf
	Handler  Handler
	Done     chan struct{}
}

func buildClientID(service string) string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s-%s-%s", service, hostname, utils.GetLocalIP())
}

// NewKafkaConsumer 创建并初始化消费者实例
func NewKafkaConsumer(conf *KafkaConsumerConf, handler Handler) (*KafkaConsumer, error) {
	clientId := buildClientID(conf.Name)
	kconf := &kafka.ConfigMap{
		"bootstrap.servers":     strings.Join(conf.Brokers, ","),
		"group.id":              conf.GroupId,
		"session.timeout.ms":    conf.SessionTimeoutMs,
		"heartbeat.interval.ms": conf.HeartbeatIntervalMs,
		"auto.offset.reset":     "latest", // 默认只消费新数据
		"enable.auto.commit":    false,    // 只支持手动提交
		"client.id":             clientId,

		// 连接 & 重试相关
		"reconnect.backoff.ms":     conf.ReconnectBackoffMs,
		"reconnect.backoff.max.ms": conf.ReconnectBackoffMaxMs,
		"retry.backoff.ms":         conf.RetryBackoffMs,
	}
	c, err := kafka.NewConsumer(kconf)
	if err != nil {
		logger.Errorf("kafka consumer create error: %v", err)
		return nil, err
	}
	logger.Infof("kafka consumer created, brokers=%v, topic=%s, group=%s", conf.Brokers, conf.Topic, conf.GroupId)
	return &KafkaConsumer{
		Consumer: c,
		Conf:     conf,
		Handler:  handler,
		Done:     make(chan struct{}),
	}, nil
}

// Start 启动消费者主循环（不自动提交 offset）
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	err := kc.Consumer.SubscribeTopics([]string{kc.Conf.Topic}, nil)
	if err != nil {
		logger.Errorf("kafka subscribe topic error: %v", err)
		return err
	}
	logger.Infof("kafka consumer subscribed, topic=%s", kc.Conf.Topic)

	go func() {
		for {
			select {
			case <-kc.Done:
				logger.Infof("kafka consumer received done signal, exiting.")
				return
			case <-ctx.Done():
				logger.Infof("kafka consumer received context done, exiting.")
				return
			default:
				msg, err := kc.Consumer.ReadMessage(time.Duration(kc.Conf.ReadTimeoutMs) * time.Millisecond)
				if err != nil {
					if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
						continue
					}
					logger.Errorf("kafka consumer read message error: %v", err)
					continue
				}
				logger.Debugf("kafka consumer received message, topic=%s, partition=%d, offset=%d",
					*msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
				kc.Handler(msg) // 只回调，不自动提交 offset
			}
		}
	}()
	return nil
}

// Commit 提供手动提交 offset 的接口，支持单条或批量
func (kc *KafkaConsumer) Commit(messages ...*kafka.Message) error {
	if len(messages) == 0 {
		_, err := kc.Consumer.Commit()
		if err != nil {
			logger.Errorf("kafka commit offsets error: %v", err)
		} else {
			logger.Infof("kafka commit offsets success (batch)")
		}
		return err
	}
	var firstErr error
	for _, msg := range messages {
		_, err := kc.Consumer.CommitMessage(msg)
		if err != nil && firstErr == nil {
			firstErr = err
			logger.Errorf("kafka commit offset error: %v, topic=%s, partition=%d, offset=%d",
				err, *msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
		} else if err == nil {
			logger.Debugf("kafka commit offset success, topic=%s, partition=%d, offset=%d",
				*msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset)
		}
	}
	return firstErr
}

// Close 优雅关闭消费者
func (kc *KafkaConsumer) Close() {
	close(kc.Done)
	kc.Consumer.Close()
	logger.Infof("kafka consumer closed")
}
