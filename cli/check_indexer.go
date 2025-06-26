package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var tables = []string{
	"chain_event",
	"balance",
	"token",
	"pool",
	"transfer_event",
}

func main() {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	for _, table := range tables {
		fmt.Printf("\n✅ 表 [%s] 的索引如下：\n", table)

		rows, err := db.Query("SHOW INDEX FROM " + table)
		if err != nil {
			log.Printf("❌ 表 [%s] 索引查询失败: %v", table, err)
			continue
		}
		cols, _ := rows.Columns()
		values := make([]interface{}, len(cols))
		scanArgs := make([]interface{}, len(cols))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		hasRows := false
		for rows.Next() {
			hasRows = true
			if err := rows.Scan(scanArgs...); err != nil {
				log.Printf("索引信息扫描失败: %v", err)
				break
			}
			fmt.Println("🧩 一条索引记录：")
			for i, col := range cols {
				switch v := values[i].(type) {
				case []byte:
					fmt.Printf("  %s = %s\n", col, string(v))
				default:
					fmt.Printf("  %s = %v\n", col, v)
				}
			}
			fmt.Println()
		}

		if !hasRows {
			fmt.Printf("⚠️ 表 [%s] 未找到任何索引记录\n", table)
		}

		rows.Close()
	}
}
