package utils

import (
	"encoding/binary"
	"github.com/cespare/xxhash/v2"
	"github.com/mr-tron/base58"
)

func EventIdHash(eventId uint64) int32 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], eventId)
	h := xxhash.Sum64(buf[:])

	// 非连续位段采样（更分散），高位更随机
	part1 := (h >> 55) & 0x7F // bits 55~61 (7 bit)
	part2 := (h >> 45) & 0xFF // bits 45~52 (8 bit)
	part3 := (h >> 34) & 0xFF // bits 34~41 (8 bit)
	part4 := (h >> 3) & 0xFF  // bits 3~10  (8 bit) → 扰动

	// 拼接为 32 位 hash
	return int32((part1 << 24) | (part2 << 16) | (part3 << 8) | part4)
}

// TxHashToString 将 64 字节交易哈希转为 base58 字符串表示
func TxHashToString(hash []byte) string {
	if len(hash) != 64 {
		panic("TxHashToString: invalid hash length, expected 64 bytes")
	}
	return base58.Encode(hash)
}
