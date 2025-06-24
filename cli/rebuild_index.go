package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // 替换为你的驱动
	"log"
)

func main() {
	// 替换为你实际的 Lindorm 数据库连接信息
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer db.Close()

	// 1. 删除旧索引
	dropSQL := `DROP INDEX idx_balance_token ON balance`
	if _, err := db.Exec(dropSQL); err != nil {
		log.Fatalf("drop index failed: %v", err)
	}
	fmt.Println("Dropped index idx_balance_token")

	// 2. 创建新索引
	createSQL := `
CREATE INDEX IF NOT EXISTS idx_balance_token_owner
ON balance(token_address, owner_address)
INCLUDE (balance)
`
	if _, err := db.Exec(createSQL); err != nil {
		log.Fatalf("create index failed: %v", err)
	}
	fmt.Println("Created index idx_balance_token_owner")
}
