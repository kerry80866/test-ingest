package config

import (
	"dex-ingest-sol/pkg/logger"
	"dex-ingest-sol/pkg/mq"
	"time"
)

type LogConfig struct {
	Format   string `yaml:"format"`   // 日志格式，可选 "console"（开发调试）或 "json"（结构化，推荐生产使用）
	LogDir   string `yaml:"log_dir"`  // 日志文件目录，可为相对路径或绝对路径
	Level    string `yaml:"level"`    // 日志级别：debug / info / warn / error
	Compress bool   `yaml:"compress"` // 是否压缩旧日志文件
}

func (c *LogConfig) ToLogOption() logger.LogOption {
	return logger.LogOption{
		Format:   c.Format,
		LogDir:   c.LogDir,
		Level:    c.Level,
		Compress: c.Compress,
	}
}

type WorkerConfig struct {
	MaxBlockHold  int           `yaml:"max_block_hold"`  // 缓冲区区块数达到 N 条即触发 flush
	MaxBatchFlush int           `yaml:"max_batch_flush"` // 每批最大 flush 的区块数量
	FlushInterval time.Duration `yaml:"flush_interval"`  // 超时时间间隔（如 "3s"）
}

type LindormConf struct {
	User            string `yaml:"user"`               // MySQL 用户名
	Password        string `yaml:"password"`           // 密码
	Host            string `yaml:"host"`               // 主机名或 IP
	Port            int    `yaml:"port"`               // 固定为 33060（Lindorm 专用端口）
	Database        string `yaml:"database"`           // 数据库名
	Timeout         string `yaml:"timeout"`            // 初始连接超时时间（格式如 "5s"）
	MaxOpenConns    int    `yaml:"max_open_conns"`     // 最大连接数
	MaxIdleConns    int    `yaml:"max_idle_conns"`     // 最大空闲连接数
	ConnMaxIdleTime string `yaml:"conn_max_idle_time"` // 空闲连接最大保持时间（如 "5m"）
}

type Config struct {
	LogConf       LogConfig            `yaml:"logger"`      // 日志配置
	KafkaConsumer mq.KafkaConsumerConf `yaml:"kafka"`       // Kafka 消费者配置
	Lindorm       LindormConf          `yaml:"lindorm"`     // Lindorm 配置
	Worker        WorkerConfig         `yaml:"worker"`      // Worker 批处理配置
	IngestType    string               `yaml:"ingest_type"` // balance或者event
}
