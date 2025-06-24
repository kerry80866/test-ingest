package svc

import (
	"database/sql"
	"dex-ingest-sol/internal/config"
	"dex-ingest-sol/internal/pkg/utils"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
)

type QueryServiceContext struct {
	Cfg         *config.QueryConfig
	DB          *sql.DB
	NacosClient naming_client.INamingClient
}

func NewQueryServiceContext(c *config.QueryConfig) *QueryServiceContext {
	// 初始化 Lindorm 数据库
	db := MustInitLindorm(c.Lindorm)
	if db == nil {
		return nil
	}

	// 初始化 Nacos 客户端
	nacosClient, err := NewNacosClient(&c.Nacos)
	if err != nil {
		panic(err)
	}

	return &QueryServiceContext{
		Cfg:         c,
		DB:          db,
		NacosClient: nacosClient,
	}
}

// RegisterNacos 注册当前服务实例到 Nacos
func (ctx *QueryServiceContext) RegisterNacos() error {
	ip, err := utils.GetLocalIP()
	if err != nil {
		return fmt.Errorf("获取本机 IP 失败: %w", err)
	}
	port := uint64(ctx.Cfg.Grpc.Port)
	return RegisterNacosInstance(ctx.NacosClient, &ctx.Cfg.Nacos, ip, port)
}

// DeregisterNacos 注销当前服务实例
func (ctx *QueryServiceContext) DeregisterNacos() error {
	ip, err := utils.GetLocalIP()
	if err != nil {
		return fmt.Errorf("获取本机 IP 失败: %w", err)
	}
	port := uint64(ctx.Cfg.Grpc.Port)
	return DeregisterNacosInstance(ctx.NacosClient, &ctx.Cfg.Nacos, ip, port)
}
