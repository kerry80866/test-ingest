package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 修改为你的数据库连接信息
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	tables := []string{
		//		"chain_event",
		"balance",
		"token",
		"pool",
	}

	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			log.Printf("查询表 %s 行数失败: %v", table, err)
			continue
		}
		fmt.Printf("表 %s 行数: %d\n", table, count)
	}
}
