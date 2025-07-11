package handler

import (
	"dex-ingest-sol/pb"
	"encoding/binary"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/hashicorp/golang-lru/v2/simplelru"
)

const cacheSize = 50000

// PoolKey 表示缓存中的唯一键：32字节池地址 + 8字节 accountKey
type PoolKey [40]byte

// PoolCache 是一个 LRU 缓存，key 为 PoolKey
type PoolCache struct {
	cache *simplelru.LRU[PoolKey, struct{}]
	buf   [64]byte // 每个实例独占
}

// NewPoolCache 初始化 PoolCache
func NewPoolCache() *PoolCache {
	c, err := simplelru.NewLRU[PoolKey, struct{}](cacheSize, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create LRU cache: %v", err))
	}
	return &PoolCache{cache: c}
}

// CheckAndInsert 插入包含 base + quote 的 pool cache（适用于 CLMM 等）
func (pc *PoolCache) CheckAndInsert(pool, base, quote []byte, dex uint32) (accountKey int64, exists bool) {
	switch pb.DexType(dex) {
	case pb.DexType_DEX_RAYDIUM_CLMM,
		pb.DexType_DEX_METEORA_DLMM,
		pb.DexType_DEX_ORCA_WHIRLPOOL:
		accountKey = pc.calcAccountKey(base, quote)
	}

	key := pc.buildKey(pool, accountKey)
	_, exists = pc.cache.Get(key)
	if !exists {
		pc.cache.Add(key, struct{}{})
	}
	return
}

// calcAccountKey 计算 base+quote 的 xxhash64，确保最高 bit 为 0（兼容正数 int64）
func (pc *PoolCache) calcAccountKey(base, quote []byte) int64 {
	n := copy(pc.buf[:], base)
	copy(pc.buf[n:], quote)
	h := xxhash.Sum64(pc.buf[:n+len(quote)])
	return int64(h & 0x7FFFFFFFFFFFFFFF) // 保证是正数（最高 bit 为 0）
}

// buildKey 构造 PoolKey：32字节池地址 + 8字节 accountKey（LE编码）
func (pc *PoolCache) buildKey(pool []byte, accountKey int64) PoolKey {
	var key PoolKey
	copy(key[:32], pool)
	binary.LittleEndian.PutUint64(key[32:], uint64(accountKey))
	return key
}
