package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"log"
	"math/big"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("用法: %s <event_id>\n", os.Args[0])
		os.Exit(1)
	}

	// 解析 event_id
	eventID, ok := new(big.Int).SetString(os.Args[1], 10)
	if !ok {
		log.Fatalf("event_id 解析失败: %s", os.Args[1])
	}
	raw := eventID.Uint64()

	slot := raw >> 32
	txIndex := (raw >> 16) & 0xFFFF
	ixIndex := (raw >> 8) & 0xFF
	innerIndex := raw & 0xFF

	fmt.Println("🧩 解析结果：")
	fmt.Printf("  slot        = %d\n", slot)
	fmt.Printf("  tx_index    = %d\n", txIndex)
	fmt.Printf("  ix_index    = %d\n", ixIndex)
	fmt.Printf("  inner_index = %d\n", innerIndex)

	// 连接数据库（替换为你的连接信息）
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	hash := eventIdHash(raw)

	// 查询事件记录
	query := `
SELECT * FROM chain_event
WHERE event_id_hash = ? AND event_id = ?
LIMIT 1
`
	rows, err := db.Query(query, hash, raw)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		log.Fatalf("获取列名失败: %v", err)
	}

	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Println("\n✅ 查询结果：")
		for i, col := range cols {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				fmt.Printf("  %s = %s\n", col, string(v))
			default:
				fmt.Printf("  %s = %v\n", col, val)
			}
		}
	} else {
		fmt.Println("❌ 未查询到对应记录")
	}
}

func eventIdHash(eventId uint64) int32 {
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
