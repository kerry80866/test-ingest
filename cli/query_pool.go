package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	args := os.Args
	if len(args) < 2 || len(args) > 4 {
		fmt.Printf("用法:\n")
		fmt.Printf("  %s <pool_address>                         # 使用主键查询 pool\n", args[0])
		fmt.Printf("  %s -t <token_address>                     # 使用 token_address 查询池子\n", args[0])
		fmt.Printf("  %s -t <token_address> <quote_address>     # 使用 token+quote 查询池子\n", args[0])
		os.Exit(1)
	}

	var query string
	var params []any

	if args[1] == "-t" {
		if len(args) == 3 {
			query = "SELECT * FROM pool WHERE token_address = ? DESC LIMIT 10"
			params = append(params, args[2])
		} else if len(args) == 4 {
			query = "SELECT * FROM pool WHERE token_address = ? AND quote_address = ? LIMIT 1"
			params = append(params, args[2], args[3])
		} else {
			log.Fatalf("参数错误：-t 模式下需要 1 或 2 个参数")
		}
	} else {
		query = "SELECT * FROM pool WHERE pool_address = ? LIMIT 1"
		params = append(params, args[1])
	}

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
