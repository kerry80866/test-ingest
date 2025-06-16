package xredis

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"dex-ingest-sol/internal/pkg/logger"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	MOD_REDIS    = "xredis"
	DEFAULT_NAME = "default"
)

type RedisConfig struct {
	Name     string   `json:"name" yaml:"name"`         // 实例名称（可用于区分多 Redis 实例）
	Addr     []string `json:"addr" yaml:"addr"`         // 地址列表（支持单机、哨兵、集群多地址）
	Username string   `json:"username" yaml:"username"` // 用户名（一般 Redis 不需要，云厂商/ACL 需填写）
	Password string   `json:"password" yaml:"password"` // 密码
	DB       int      `json:"db" yaml:"db"`             // 数据库编号（0~15，单机/哨兵模式下有效）

	PoolSize     int `json:"pool_size" yaml:"pool_size"`           // 最大连接池大小
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns"` // 最小空闲连接数
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"` // 最大空闲连接数

	DialTimeout  int `json:"dial_timeout" yaml:"dial_timeout"`   // 建立连接超时时间（秒）
	ReadTimeout  int `json:"read_timeout" yaml:"read_timeout"`   // 读超时时间（秒）
	WriteTimeout int `json:"write_timeout" yaml:"write_timeout"` // 写超时时间（秒）

	TlsEnabled         bool `json:"tls_enabled" yaml:"tls_enabled"`                   // 是否启用 TLS
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify"` // 跳过 TLS 证书校验（开发环境可用，生产慎用）
}

type RedisClient struct {
	c      redis.UniversalClient
	config RedisConfig
}

func setupRedis(redisConfig *RedisConfig) (*RedisClient, error) {
	ensureConfig(redisConfig)

	opts := redis.UniversalOptions{
		Addrs:                 redisConfig.Addr,
		Username:              redisConfig.Username,
		Password:              redisConfig.Password,
		DB:                    redisConfig.DB,
		DialTimeout:           time.Duration(redisConfig.DialTimeout) * time.Second,
		ReadTimeout:           time.Duration(redisConfig.ReadTimeout) * time.Second,
		WriteTimeout:          time.Duration(redisConfig.WriteTimeout) * time.Second,
		PoolSize:              redisConfig.PoolSize,
		MinIdleConns:          redisConfig.MinIdleConns,
		MaxIdleConns:          redisConfig.MaxIdleConns,
		ContextTimeoutEnabled: true,
	}

	if redisConfig.TlsEnabled {
		systemRoots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("get system cert pool error: %w", err)
		}

		tlsConfig := &tls.Config{
			RootCAs:            systemRoots,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: redisConfig.InsecureSkipVerify,
		}
		opts.TLSConfig = tlsConfig
	}

	c := redis.NewUniversalClient(&opts)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), time.Duration(redisConfig.ReadTimeout)*time.Second)
	defer pingCancel()
	_, err := c.Ping(pingCtx).Result()
	if err != nil {
		logger.Errorf("[%s] redis ping error: %v", MOD_REDIS, err)
		return nil, err
	}

	logger.Infof("[%s] redis connected: addr=%s, db=%d, user=%s, pool_size=%d, min_idle=%d, max_idle=%d, dial_timeout=%d, read_timeout=%d, write_timeout=%d",
		MOD_REDIS,
		redisConfig.Addr[0],
		redisConfig.DB,
		redisConfig.Username,
		redisConfig.PoolSize,
		redisConfig.MinIdleConns,
		redisConfig.MaxIdleConns,
		redisConfig.DialTimeout,
		redisConfig.ReadTimeout,
		redisConfig.WriteTimeout,
	)

	return &RedisClient{
		c:      c,
		config: *redisConfig,
	}, nil
}

func (c *RedisClient) close() error {
	err := c.c.Close()
	logger.Infof("[%s] redis closed: name=%s", MOD_REDIS, c.config.Name)
	return err
}

func (c *RedisClient) getClient() redis.UniversalClient {
	return c.c
}

func ensureConfig(config *RedisConfig) {
	if config.Name == "" {
		config.Name = DEFAULT_NAME
	}
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		if config.MaxIdleConns == 0 {
			config.MinIdleConns = 5
		} else {
			config.MinIdleConns = config.MaxIdleConns / 2
		}
		config.MinIdleConns = 5
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 5
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30
	}
}
