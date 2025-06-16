package svc

import (
	"database/sql"
	"dex-ingest-sol/internal/config"

	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Cfg   *config.Config // 改为指针
	DB    *sql.DB
	Redis *redis.Client
}

func NewServiceContext(c *config.Config) *ServiceContext {
	return &ServiceContext{
		Cfg:   c,
		DB:    MustInitLindorm(c.Lindorm),
		Redis: nil,
	}
}
