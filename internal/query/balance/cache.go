package balance

import (
	"dex-ingest-sol/internal/pkg/db"
	"time"
)

// 缓存 TTL 设置
const (
	balancesByAccountsTTL = 5 * time.Second
	balancesByOwnerTTL    = 10 * time.Second
	holderCountTTL        = 30 * time.Second
	topHoldersByTokenTTL  = 60 * time.Second
)

// 缓存实例
var (
	balancesByAccountsCache = db.NewLockCache(10000, true)
	balancesByOwnerCache    = db.NewLockCache(10000, true)
	holderCountCache        = db.NewLockCache(10000, true)
	topHoldersByTokenCache  = db.NewLockCache(1000, true)
)
