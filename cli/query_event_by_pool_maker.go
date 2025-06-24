package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	args := os.Args
	if len(args) < 2 || len(args) > 6 {
		fmt.Printf("用法: %s <pool_address> [maker] [event_type] [event_id] [limit]\n", args[0])
		os.Exit(1)
	}

	pool := args[1]

	query := "SELECT * FROM chain_event WHERE pool_address = ?"
	params := []any{pool}

	// 可选：maker
	if len(args) >= 3 && args[2] != "-" {
		query += " AND user_wallet = ?"
		params = append(params, args[2])
	}

	// 可选：event_type
	if len(args) >= 4 && args[3] != "-" {
		eventType, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatalf("event_type 应为整数: %v", err)
		}
		query += " AND event_type = ?"
		params = append(params, eventType)
	}

	// 可选：event_id（分页）
	if len(args) >= 5 && args[4] != "-" {
		eventID, ok := new(big.Int).SetString(args[4], 10)
		if !ok {
			log.Fatalf("event_id 格式非法: %s", args[4])
		}
		query += " AND event_id < ?"
		params = append(params, eventID.String())
	}

	// 可选：limit
	limit := 10
	if len(args) == 6 {
		l, err := strconv.Atoi(args[5])
		if err != nil || l <= 0 {
			log.Fatalf("limit 应为正整数: %v", err)
		}
		limit = l
	}

	query += " ORDER BY event_id DESC LIMIT ?"
	params = append(params, limit)

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(query, params...)
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

	found := false
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			log.Fatalf("扫描失败: %v", err)
		}
		if !found {
			fmt.Println("✅ 查询结果：")
			found = true
		}
		fmt.Println("-----")
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				fmt.Printf("  %s = %s\n", col, string(v))
			default:
				fmt.Printf("  %s = %v\n", col, v)
			}
		}
	}
	if !found {
		fmt.Println("❌ 未查询到任何记录")
	}
}
