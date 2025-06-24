package svc

import (
	"database/sql"
	"dex-ingest-sol/internal/config"

	"github.com/redis/go-redis/v9"
)

type IngestServiceContext struct {
	Cfg   *config.IngestConfig
	DB    *sql.DB
	Redis *redis.Client
}

func NewIngestServiceContext(c *config.IngestConfig) *IngestServiceContext {
	return &IngestServiceContext{
		Cfg:   c,
		DB:    MustInitLindorm(c.Lindorm),
		Redis: nil,
	}
}
