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
	balancesByAccountsCache = db.NewLockCache(300)
	balancesByOwnerCache    = db.NewLockCache(300)
	holderCountCache        = db.NewLockCache(300)
	topHoldersByTokenCache  = db.NewLockCache(100)
)
