package db

import (
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
	validAt   atomic.Int64 // 存储 UnixNano
}

func (e *Entry) SetValidAt(t time.Time) {
	e.validAt.Store(t.UnixNano())
}

func (e *Entry) GetValidAt() time.Time {
	return time.Unix(0, e.validAt.Load())
}

func (e *Entry) IsExpired() bool {
	return time.Now().UnixNano() > e.validAt.Load()
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

func (lc *LockCache) Do(key any, fn func(e *Entry, created bool)) {
	if lc.useStringKey {
		lc.doStr(key, fn)
	} else {
		lc.doUint64(key, fn)
	}
}

func (lc *LockCache) doUint64(key any, fn func(e *Entry, created bool)) {
	hash, ok := key.(uint64)
	if !ok {
		panic("LockCache: expected uint64 key")
	}

	var entry *Entry
	created := false

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
		created = true
	}
	lc.mu.Unlock()

	entry.mu.Lock()
	fn(entry, created)
	entry.mu.Unlock()
}

func (lc *LockCache) doStr(key any, fn func(e *Entry, created bool)) {
	hash, ok := key.(string)
	if !ok {
		panic("LockCache: expected string key")
	}

	var entry *Entry
	created := false

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
		created = true
	}
	lc.mu.Unlock()

	entry.mu.Lock()
	fn(entry, created)
	entry.mu.Unlock()
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
	now := time.Now().UnixNano()
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.useStringKey {
		for key, entry := range lc.entriesStr {
			if now > entry.validAt.Load() {
				delete(lc.entriesStr, key)
			}
		}
	} else {
		for hash, entry := range lc.entries {
			if now > entry.validAt.Load() {
				delete(lc.entries, hash)
			}
		}
	}
}
