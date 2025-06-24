package main

import (
	"dex-ingest-sol/internal/config"
	"dex-ingest-sol/internal/ingest"
	"dex-ingest-sol/internal/pkg/configloader"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/monitor"
	"dex-ingest-sol/internal/svc"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	zerosvc "github.com/zeromicro/go-zero/core/service"
	"runtime/debug"
)

var configFile = flag.String("f", "etc/ingest-event.yaml", "the config file")

func main() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic: %+v\nstack: %s", r, debug.Stack())
		}
	}()
	defer logger.Sync()

	flag.Parse()
	logger.Infof("Loading config from %s", *configFile)

	// 加载配置
	var c config.IngestConfig
	if err := configloader.LoadConfig(*configFile, &c); err != nil {
		panic(fmt.Sprintf("配置加载失败: %v", err))
	}

	// 初始化 zap 日志
	logger.InitLogger(c.LogConf.ToLogOption())
	logx.SetWriter(logger.ZapWriter{})

	// 初始化依赖注入上下文
	svcCtx := svc.NewIngestServiceContext(&c)

	// 解析 ingest_type 并构建 PartitionRouter
	routerType, err := ingest.ParseRouterType(c.IngestType)
	if err != nil {
		logger.Errorf("无效的 ingest_type: %s", c.IngestType)
		panic(fmt.Errorf("配置错误: %w", err))
	}
	partitionRouter := ingest.NewPartitionRouter(svcCtx.DB, svcCtx.Redis, &c.Worker, routerType)

	// 构建 Kafka 消费核心组件
	consumerRunner, err := ingest.NewConsumerRunner(&c.KafkaConsumer, partitionRouter)
	if err != nil {
		panic(err)
	}

	// 构造 go-zero ServiceGroup 管理服务
	sg := zerosvc.NewServiceGroup()
	sg.Add(partitionRouter)
	sg.Add(consumerRunner)

	if c.Monitor.Port > 0 {
		monitorServer := monitor.NewMonitorServer(c.Monitor.Port)
		sg.Add(monitorServer)
	}

	logger.Infof("落库服务启动成功, ingest_type=%s, topic=%s",
		c.IngestType, c.KafkaConsumer.Topic)
	sg.Start()
}
