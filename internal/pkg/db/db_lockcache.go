package db

import (
	"dex-ingest-sol/internal/pkg/logger"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const interval = 30 * time.Second

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
	mu           sync.Mutex
	entries      map[uint64]*Entry
	entriesStr   map[string]*Entry
	maxSize      int
	useStringKey bool
	cleanOnce    sync.Once
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

	// 获取或创建 entry
	lc.mu.Lock()
	entry, ok = lc.entriesStr[hash]
	if !ok {
		if len(lc.entriesStr) >= lc.maxSize {
			lc.mu.Unlock()
			lc.cleanupExpired()
			lc.mu.Lock()
			entry, ok = lc.entriesStr[hash]
		}
	}
	if !ok {
		entry = &Entry{CreatedAt: time.Now()}
		lc.entriesStr[hash] = entry
	}
	entry.inUse.Add(1)
	lc.mu.Unlock()

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

	// 获取或创建 entry
	lc.mu.Lock()
	entry, ok = lc.entries[hash]
	if !ok {
		if len(lc.entries) >= lc.maxSize {
			lc.mu.Unlock()
			lc.cleanupExpired()
			lc.mu.Lock()
			entry, ok = lc.entries[hash]
		}
	}
	if !ok {
		entry = &Entry{CreatedAt: time.Now()}
		lc.entries[hash] = entry
	}
	entry.inUse.Add(1)
	lc.mu.Unlock()

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
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			<-ticker.C
			lc.cleanupExpired()
		}
	}()
}

func (lc *LockCache) cleanupExpired() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.useStringKey {
		for key, entry := range lc.entriesStr {
			if entry.inUse.Load() == 0 && entry.IsExpired() {
				delete(lc.entriesStr, key)
			}
		}
	} else {
		for hash, entry := range lc.entries {
			if entry.inUse.Load() == 0 && entry.IsExpired() {
				delete(lc.entries, hash)
			}
		}
	}
}
