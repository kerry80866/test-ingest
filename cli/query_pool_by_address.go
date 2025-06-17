package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("用法: %s <pool_address>\n", os.Args[0])
		os.Exit(1)
	}
	addr := os.Args[1]

	query := `
SELECT * FROM pool WHERE pool_address = ? LIMIT 1
`

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(query, addr)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	printRows(rows)
}

func printRows(rows *sql.Rows) {
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
			log.Fatalf("扫描失败: %v", err)
		}
		fmt.Println("✅ 查询结果：")
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				fmt.Printf("  %s = %s\n", col, string(v))
			default:
				fmt.Printf("  %s = %v\n", col, v)
			}
		}
	} else {
		fmt.Println("❌ 未查询到任何记录")
	}
}
