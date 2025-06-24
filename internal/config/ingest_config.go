package config

import (
	"dex-ingest-sol/internal/pkg/mq"
	"time"
)

type WorkerConfig struct {
	MaxBlockHold  int           `yaml:"max_block_hold"`  // 缓冲区区块数达到 N 条即触发 flush
	MaxBatchFlush int           `yaml:"max_batch_flush"` // 每批最大 flush 的区块数量
	FlushInterval time.Duration `yaml:"flush_interval"`  // 超时时间间隔（如 "3s"）
}

type IngestConfig struct {
	Monitor       MonitorConfig        `yaml:"monitor"`     // 监控配置
	LogConf       LogConfig            `yaml:"logger"`      // 日志配置
	KafkaConsumer mq.KafkaConsumerConf `yaml:"kafka"`       // Kafka 消费者配置
	Lindorm       LindormConf          `yaml:"lindorm"`     // Lindorm 配置
	Worker        WorkerConfig         `yaml:"worker"`      // Worker 批处理配置
	IngestType    string               `yaml:"ingest_type"` // balance或者event
}
