package ingest

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/mq"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ConsumerRunner 结构体封装 consumer 启动逻辑
type ConsumerRunner struct {
	Kafka   *mq.KafkaConsumer
	Context context.Context
	Cancel  context.CancelFunc
}

// NewConsumerRunner 构造函数
func NewConsumerRunner(conf *mq.KafkaConsumerConf, router *PartitionRouter) (*ConsumerRunner, error) {
	ctx, cancel := context.WithCancel(context.Background())

	kc, err := mq.NewKafkaConsumer(conf, func(msg *kafka.Message) {
		router.Dispatch(msg)
	})
	if err != nil {
		cancel() // 避免泄漏
		return nil, err
	}

	router.SetKafkaConsumer(kc.Consumer)
	return &ConsumerRunner{
		Kafka:   kc,
		Context: ctx,
		Cancel:  cancel,
	}, nil
}

// Start 启动消费主循环（非阻塞）
func (cr *ConsumerRunner) Start() {
	if err := cr.Kafka.Start(cr.Context); err != nil {
		// 记录错误日志
		logger.Errorf("Kafka consumer start failed: %v", err)

		// 推荐：触发 panic，让主程序终止（服务治理平台可以重启）
		panic(err)
	}
}

// Stop 优雅退出
func (cr *ConsumerRunner) Stop() {
	cr.Cancel()
	cr.Kafka.Close()
}

// CommitOffset 包装提交方法（可选扩展）
func (cr *ConsumerRunner) CommitOffset(messages ...*kafka.Message) error {
	return cr.Kafka.Commit(messages...)
}
