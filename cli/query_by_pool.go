package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	args := os.Args
	if len(args) < 2 || len(args) > 4 {
		fmt.Printf("用法:\n")
		fmt.Printf("  %s <pool_address>\n", args[0])
		fmt.Printf("  %s <pool_address> <event_type>\n", args[0])
		fmt.Printf("  %s <pool_address> <event_type> <event_id>\n", args[0])
		os.Exit(1)
	}

	var (
		query  string
		params []any
	)

	poolAddress := args[1]
	query = "SELECT * FROM chain_event WHERE pool_address = ?"
	params = append(params, poolAddress)

	if len(args) >= 3 {
		eventType, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalf("event_type 参数格式错误: %v", err)
		}
		query += " AND event_type = ?"
		params = append(params, eventType)
	}

	if len(args) == 4 {
		eventID, err := strconv.ParseUint(args[3], 10, 64)
		if err != nil {
			log.Fatalf("event_id 参数格式错误: %v", err)
		}
		query += " AND event_id < ?"
		params = append(params, eventID)
	}

	query += " ORDER BY event_id DESC LIMIT 10"

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
