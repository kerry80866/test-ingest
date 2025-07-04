package utils

import (
	"github.com/hashicorp/golang-lru"
	"github.com/mr-tron/base58"
)

// NewBase58Cache 初始化 LRU 缓存（最大 20,000 个）
func NewBase58Cache() *lru.Cache {
	cache, err := lru.New(20000)
	if err != nil {
		panic(err)
	}
	return cache
}

// EncodeBase58Strict 编码固定长度为 32 字节的地址。
// 若 b 长度非法将 panic，适用于强约束场景。
func EncodeBase58Strict(cache *lru.Cache, b []byte) string {
	if len(b) != 32 {
		panic("EncodeBase58Cached: input must be 32 bytes")
	}

	// 对固定 32 字节切片，用 string(b) 做 map key 是安全且高效的
	key := string(b)
	if val, ok := knownTokenMintToID[key]; ok {
		return val
	}

	// 查询缓存
	if val, ok := cache.Get(key); ok {
		return val.(string)
	}

	decoded := base58.Encode(b)
	cache.Add(key, decoded)
	return decoded
}

// EncodeBase58Optional 若 b 为 nil 返回空字符串；
// 否则调用 EncodeBase58Strict（非法长度仍 panic）。
// 适用于允许空地址的容错场景。
func EncodeBase58Optional(cache *lru.Cache, b []byte) string {
	if b == nil {
		return ""
	}
	return EncodeBase58Strict(cache, b)
}
