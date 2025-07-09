package chainevent

import (
	"dex-ingest-sol/internal/pkg/db"
	"time"
)

// 缓存 TTL 设置
const (
	chainEventsByIDsTTL       = 5 * time.Second
	chainEventsByPoolTTL      = 10 * time.Second
	chainEventsByPoolEmptyTTL = 3 * time.Second
	chainEventsByUserTTL      = 30 * time.Second
	transferEventsTTL         = 30 * time.Second
)

// 缓存实例
var (
	chainEventsByPoolCache = db.NewLockCache(300)
)
