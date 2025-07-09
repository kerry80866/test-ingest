package pool

import (
	"dex-ingest-sol/internal/pkg/db"
	"time"
)

// 缓存 TTL 设置
const (
	poolsByAddressTTL      = 120 * time.Second // 正常数据 TTL
	poolsByAddressEmptyTTL = 20 * time.Second  // 空结果 TTL，防止穿透
	poolsByTokenTTL        = 60 * time.Second
	poolsByTokenEmptyTTL   = 20 * time.Second // 空结果 TTL，防止穿透
)

// 缓存实例
var (
	poolsByAddressCache = db.NewLockCache(10000, true)
	poolsByTokenCache   = db.NewLockCache(10000, true)
)
