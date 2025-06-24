package config

import (
	"fmt"
	"github.com/zeromicro/go-zero/zrpc"
	"time"
)

type GrpcMiddlewaresConfig struct {
	Prometheus bool `yaml:"prometheus"`
}

type GrpcMethodTimeoutConfig struct {
	FullMethod string        `yaml:"full_method"`
	Timeout    time.Duration `yaml:"timeout"`
}

// GrpcConfig gRPC 服务相关配置
type GrpcConfig struct {
	Port           int                       `yaml:"port"`
	Timeout        int64                     `yaml:"timeout"`
	CpuThreshold   int64                     `yaml:"cpu_threshold"`
	Health         bool                      `yaml:"health"`
	Middlewares    GrpcMiddlewaresConfig     `yaml:"middlewares"`
	MethodTimeouts []GrpcMethodTimeoutConfig `yaml:"method_timeouts"`
}

func (c *GrpcConfig) ToRpcServerConf() zrpc.RpcServerConf {
	var mts []zrpc.MethodTimeoutConf
	for _, m := range c.MethodTimeouts {
		mts = append(mts, zrpc.MethodTimeoutConf{
			FullMethod: m.FullMethod,
			Timeout:    m.Timeout,
		})
	}
	return zrpc.RpcServerConf{
		ListenOn:     fmt.Sprintf("0.0.0.0:%d", c.Port),
		Timeout:      c.Timeout,
		CpuThreshold: c.CpuThreshold,
		Health:       c.Health,
		Middlewares: zrpc.ServerMiddlewaresConf{
			Prometheus: c.Middlewares.Prometheus,
		},
		MethodTimeouts: mts,
	}
}

// NacosConfig Nacos 注册中心相关配置
type NacosConfig struct {
	ServiceName         string   `yaml:"service_name"`            // 注册到 Nacos 的服务名称，用于服务发现
	GroupName           string   `yaml:"group_name"`              // Nacos 中的服分组名称，默认为 DEFAULT_GROUP
	Weight              int      `yaml:"weight"`                  // 服务权重，用于负载均衡
	Username            string   `yaml:"username"`                // 连接 Nacos 的用户名
	Password            string   `yaml:"password"`                // 连接 Nacos 的密码
	TimeoutMs           int      `yaml:"timeout_ms"`              // 与 Nacos 服务中心通信超时时间（毫秒）
	BeatIntervalMs      int      `yaml:"beat_interval_ms"`        // 心跳发送间隔（毫秒）
	NamespaceId         string   `yaml:"namespace_id"`            // Nacos 命名空间，空字符串表示默认公共命名空间
	NotLoadCacheAtStart bool     `yaml:"not_load_cache_at_start"` // 启动时是否读取本地缓存，true 表示不读取，防止脏数据
	LogLevel            string   `yaml:"log_level"`               // 日志级别，info/debug/warn/error
	CacheDir            string   `yaml:"cache_dir"`               // 本地缓存目录
	LogDir              string   `yaml:"log_dir"`                 // 日志文件目录
	Endpoint            string   `yaml:"endpoint"`                // Nacos 云端地址，留空表示不使用
	StaticServers       []string `yaml:"static_servers"`          // 静态 Nacos 服务列表，与 endpoint 二选一
}

type QueryConfig struct {
	Grpc    GrpcConfig    `yaml:"grpc"`    // gRPC 服务配置（支持 timeout、method_timeouts 等）
	Monitor MonitorConfig `yaml:"monitor"` // 监控配置
	LogConf LogConfig     `yaml:"logger"`  // 日志配置
	Lindorm LindormConf   `yaml:"lindorm"` // Lindorm 配置
	Nacos   NacosConfig   `yaml:"nacos"`   // Nacos 配置
}
