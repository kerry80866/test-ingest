package db

import (
	"dex-ingest-sol/internal/pkg/logger"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const interval = 60 * time.Second

type Entry struct {
	mu        sync.Mutex
	Result    any
	Raw       any
	CreatedAt time.Time
	validAt   atomic.Uint32 // 存储 Unix 秒级时间戳
	inUse     atomic.Int32
}

func (e *Entry) SetValidAt(t time.Time) {
	e.validAt.Store(uint32(t.Unix()))
}

func (e *Entry) IsExpired() bool {
	sec := e.validAt.Load()
	return time.Now().Unix() > int64(sec)
}

type LockCache struct {
	mu           sync.RWMutex
	entries      map[uint64]*Entry
	entriesStr   map[string]*Entry
	maxSize      int
	useStringKey bool
}

func NewLockCache(maxSize int, useStringKey bool) *LockCache {
	lc := &LockCache{
		maxSize:      maxSize,
		useStringKey: useStringKey,
	}
	if useStringKey {
		lc.entriesStr = make(map[string]*Entry)
	} else {
		lc.entries = make(map[uint64]*Entry)
	}
	go lc.startAutoCleanup()
	return lc
}

func (lc *LockCache) Do(key any, fn func(e *Entry)) {
	if lc.useStringKey {
		lc.doStr(key, fn)
	} else {
		lc.doUint64(key, fn)
	}
}

func (lc *LockCache) doStr(key any, fn func(e *Entry)) {
	hash, ok := key.(string)
	if !ok {
		panic("LockCache: expected string key")
	}

	var entry *Entry
	var entriesLen int

	// 获取或创建 entry
	lc.mu.RLock()
	entry, ok = lc.entriesStr[hash]
	entriesLen = len(lc.entriesStr)
	lc.mu.RUnlock()

	if !ok && entriesLen >= lc.maxSize {
		lc.cleanupExpired()

		lc.mu.RLock()
		entry, ok = lc.entriesStr[hash]
		lc.mu.RUnlock()
	}

	if !ok {
		lc.mu.Lock()
		entry, ok = lc.entriesStr[hash]
		if !ok {
			entry = &Entry{CreatedAt: time.Now()}
			lc.entriesStr[hash] = entry
		}
		entry.inUse.Add(1)
		lc.mu.Unlock()
	} else {
		// 没有加锁，但可以接受：
		// - 命中路径的 entry 通常是活跃的，不会过期；
		// - 就算在极端并发下被清理线程误删，也只会导致下一次重新构建 entry；
		// - 不影响功能，仅多了一次请求（容忍型设计，性能优先）
		entry.inUse.Add(1)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()
	defer entry.inUse.Add(-1)

	lc.safeRun(entry, fn)
}

func (lc *LockCache) doUint64(key any, fn func(e *Entry)) {
	hash, ok := key.(uint64)
	if !ok {
		panic("LockCache: expected uint64 key")
	}

	var entry *Entry
	var entriesLen int

	// 获取或创建 entry
	lc.mu.RLock()
	entry, ok = lc.entries[hash]
	entriesLen = len(lc.entries)
	lc.mu.RUnlock()

	if !ok && entriesLen >= lc.maxSize {
		lc.cleanupExpired()

		lc.mu.RLock()
		entry, ok = lc.entries[hash]
		lc.mu.RUnlock()
	}

	if !ok {
		lc.mu.Lock()
		entry, ok = lc.entries[hash]
		if !ok {
			entry = &Entry{CreatedAt: time.Now()}
			lc.entries[hash] = entry
		}
		entry.inUse.Add(1)
		lc.mu.Unlock()
	} else {
		// 没有加锁，但可以接受：
		// - 命中路径的 entry 通常是活跃的，不会过期；
		// - 就算在极端并发下被清理线程误删，也只会导致下一次重新构建 entry；
		// - 不影响功能，仅多了一次请求（容忍型设计，性能优先）
		entry.inUse.Add(1)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()
	defer entry.inUse.Add(-1)

	lc.safeRun(entry, fn)
}

func (lc *LockCache) safeRun(e *Entry, fn func(e *Entry)) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[LockCache] panic in fn: %v\n%s", r, debug.Stack())
		}
	}()
	fn(e)
}

func (lc *LockCache) startAutoCleanup() {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		lc.cleanupExpired()
	}
}

func (lc *LockCache) cleanupExpired() {
	if lc.useStringKey {
		lc.cleanupExpiredStr()
	} else {
		lc.cleanupExpiredUint64()
	}
}

func (lc *LockCache) cleanupExpiredStr() {
	var expiredKeys []string

	lc.mu.RLock()
	if len(lc.entriesStr) < lc.maxSize/2 {
		lc.mu.RUnlock()
		return
	}

	for key, entry := range lc.entriesStr {
		if entry.inUse.Load() == 0 && entry.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}
	lc.mu.RUnlock()

	if len(expiredKeys) == 0 {
		return
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.entriesStr) < lc.maxSize/2 {
		return
	}

	for _, key := range expiredKeys {
		delete(lc.entriesStr, key)
	}
}

func (lc *LockCache) cleanupExpiredUint64() {
	var expiredKeys []uint64

	lc.mu.RLock()
	if len(lc.entries) < lc.maxSize/2 {
		lc.mu.RUnlock()
		return
	}

	for key, entry := range lc.entries {
		if entry.inUse.Load() == 0 && entry.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}
	lc.mu.RUnlock()

	if len(expiredKeys) == 0 {
		return
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 再次确认是否仍需清理（可能期间其他 goroutine 已清理或 map 缩小）
	if len(lc.entries) < lc.maxSize/2 {
		return
	}

	for _, key := range expiredKeys {
		delete(lc.entries, key)
	}
}
