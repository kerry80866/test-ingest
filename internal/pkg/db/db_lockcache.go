package db

import (
	"dex-ingest-sol/internal/pkg/logger"
	"github.com/cespare/xxhash/v2"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	interval   = 120 * time.Second
	shardCount = 1 << 5 // shardCount 必须是 2 的幂
)

type Entry struct {
	mu        sync.Mutex
	Result    any
	Raw       any
	CreatedAt time.Time
	validAt   atomic.Uint32 // 存储 Unix 秒级时间戳
	inUse     atomic.Int32
}

type LockCache struct {
	shards [shardCount]*shard
}

type shard struct {
	maxSize int
	mu      sync.RWMutex
	entries map[string]*Entry
}

func (e *Entry) SetValidAt(t time.Time) {
	e.validAt.Store(uint32(t.Unix()))
}

func (e *Entry) IsExpired() bool {
	sec := e.validAt.Load()
	return time.Now().Unix() > int64(sec)
}

func NewLockCache(maxSize int) *LockCache {
	lc := &LockCache{}
	for i := range lc.shards {
		lc.shards[i] = &shard{
			entries: make(map[string]*Entry),
			maxSize: maxSize,
		}
	}
	go lc.startAutoCleanup()
	return lc
}

func (lc *LockCache) getShard(key string) *shard {
	h := xxhash.Sum64String(key)
	// 混合高位和低位，避免结构化 key 导致分布不均
	mixed := h ^ (h >> 32)
	return lc.shards[mixed&(shardCount-1)]
}

func (lc *LockCache) Do(hash string, fn func(e *Entry)) {
	s := lc.getShard(hash)
	var entry *Entry
	var entriesLen int

	// 获取或创建 entry
	s.mu.RLock()
	entry, ok := s.entries[hash]
	entriesLen = len(s.entries)
	s.mu.RUnlock()

	if !ok && entriesLen >= s.maxSize {
		s.cleanupExpired()

		s.mu.RLock()
		entry, ok = s.entries[hash]
		s.mu.RUnlock()
	}

	if !ok {
		s.mu.Lock()
		entry, ok = s.entries[hash]
		if !ok {
			entry = &Entry{CreatedAt: time.Now()}
			s.entries[hash] = entry
		}
		entry.inUse.Add(1)
		s.mu.Unlock()
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
	for {
		for _, s := range lc.shards {
			s.cleanupExpired()
			time.Sleep(interval / shardCount)
		}
	}
}

func (s *shard) cleanupExpired() {
	var expiredKeys []string

	s.mu.RLock()
	if len(s.entries) < s.maxSize/2 {
		s.mu.RUnlock()
		return
	}

	for key, entry := range s.entries {
		if entry.inUse.Load() == 0 && entry.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}
	s.mu.RUnlock()

	if len(expiredKeys) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) < s.maxSize/2 {
		return
	}

	for _, key := range expiredKeys {
		delete(s.entries, key)
	}
}
