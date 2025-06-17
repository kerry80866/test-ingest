package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 && len(args) != 2 {
		fmt.Println("用法: query_pool_by_token <token_address> [<quote_address>]")
		os.Exit(1)
	}

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	var rows *sql.Rows
	if len(args) == 1 {
		query := `SELECT * FROM pool WHERE token_address = ?`
		rows, err = db.Query(query, args[0])
	} else {
		query := `SELECT * FROM pool WHERE token_address = ? AND quote_address = ?`
		rows, err = db.Query(query, args[0], args[1])
	}
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
			log.Fatalf("扫描行失败: %v", err)
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
		found = true
	}

	if !found {
		fmt.Println("❌ 未查询到任何记录")
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("遍历结果出错: %v", err)
	}
}
