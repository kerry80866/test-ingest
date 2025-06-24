package main

import (
	"dex-ingest-sol/internal/config"
	"dex-ingest-sol/internal/pkg/configloader"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/monitor"
	"dex-ingest-sol/internal/query"
	"dex-ingest-sol/internal/svc"
	"dex-ingest-sol/pb"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	zerosvc "github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"runtime/debug"
)

var configFile = flag.String("f", "etc/query-service.yaml", "the config file")

func main() {
	// 捕获 panic，打印堆栈信息
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic: %+v\nstack: %s", r, debug.Stack())
		}
	}()
	defer logger.Sync()

	flag.Parse()
	logger.Infof("Loading config from %s", *configFile)

	// ========== 1. 加载配置 ==========
	var c config.QueryConfig
	if err := configloader.LoadConfig(*configFile, &c); err != nil {
		panic(fmt.Sprintf("配置加载失败: %v", err))
	}

	// ========== 2. 初始化日志 ==========
	logger.InitLogger(c.LogConf.ToLogOption())
	logx.SetWriter(logger.ZapWriter{})

	// ========== 3. 初始化上下文 ==========
	svcCtx := svc.NewQueryServiceContext(&c)

	// ========== 4. 构建服务组 ==========
	sg := zerosvc.NewServiceGroup()

	// 添加 Prometheus + pprof 监控服务（可选）
	if c.Monitor.Port > 0 {
		monitorServer := monitor.NewMonitorServer(c.Monitor.Port)
		sg.Add(monitorServer)
	}

	// ========== 5. 构建并注册 gRPC 服务 ==========
	queryService := query.NewQueryService(svcCtx.DB)
	rpcServer := zrpc.MustNewServer(c.Grpc.ToRpcServerConf(), func(grpcServer *grpc.Server) {
		pb.RegisterIngestQueryServiceServer(grpcServer, queryService)
	})
	sg.Add(rpcServer)

	// ========== 6. 注册到 Nacos ==========
	if err := svcCtx.RegisterNacos(); err != nil {
		panic(fmt.Sprintf("注册 Nacos 失败: %v", err))
	}
	defer svcCtx.DeregisterNacos()

	// ========== 7. 启动服务 ==========
	logger.Infof("服务启动成功，gRPC端口: %d，Nacos服务名: %s", c.Grpc.Port, c.Nacos.ServiceName)
	sg.Start()
}
