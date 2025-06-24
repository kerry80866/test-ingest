package svc

import (
	"dex-ingest-sol/internal/config"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"strconv"
	"strings"
)

func NewNacosClient(cfg *config.NacosConfig) (naming_client.INamingClient, error) {
	var serverConfigs []constant.ServerConfig
	var endpoint string

	// 判断使用哪种模式
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	} else if len(cfg.StaticServers) > 0 {
		serverConfigs = make([]constant.ServerConfig, 0, len(cfg.StaticServers))
		for _, s := range cfg.StaticServers {
			parts := strings.Split(s, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid static server format: %s", s)
			}
			port, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port in static server %s: %w", s, err)
			}
			serverConfigs = append(serverConfigs, constant.ServerConfig{
				IpAddr: parts[0],
				Port:   uint64(port),
				Scheme: "http",
			})
		}
	} else {
		return nil, fmt.Errorf("either endpoint or static_servers must be configured")
	}

	clientConfig := constant.ClientConfig{
		Endpoint:            endpoint,
		NamespaceId:         cfg.NamespaceId,
		TimeoutMs:           uint64(cfg.TimeoutMs),
		NotLoadCacheAtStart: cfg.NotLoadCacheAtStart,
		LogDir:              cfg.LogDir,
		CacheDir:            cfg.CacheDir,
		LogLevel:            cfg.LogLevel,
		Username:            cfg.Username,
		Password:            cfg.Password,
		BeatInterval:        int64(cfg.BeatIntervalMs),
	}

	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	})
	if err != nil {
		return nil, fmt.Errorf("create nacos naming client failed: %w", err)
	}
	return client, nil
}

// RegisterNacosInstance 注册服务实例到 Nacos
func RegisterNacosInstance(client naming_client.INamingClient, cfg *config.NacosConfig, ip string, port uint64) error {
	param := vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: cfg.ServiceName,
		Weight:      float64(cfg.Weight),
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	}

	success, err := client.RegisterInstance(param)
	if err != nil {
		return fmt.Errorf("register instance failed: %w", err)
	}
	if !success {
		return fmt.Errorf("register instance returned false")
	}
	return nil
}

// DeregisterNacosInstance 注销服务实例
func DeregisterNacosInstance(client naming_client.INamingClient, cfg *config.NacosConfig, ip string, port uint64) error {
	param := vo.DeregisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: cfg.ServiceName,
		Ephemeral:   true,
	}
	success, err := client.DeregisterInstance(param)
	if err != nil {
		return fmt.Errorf("deregister instance failed: %w", err)
	}
	if !success {
		return fmt.Errorf("deregister instance returned false")
	}
	return nil
}
