package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql" // 或者你使用的 Lindorm 驱动
)

const (
	batchSize = 50
)

func main() {
	// 替换为你的数据库连接
	db, err := sql.Open("mysql", DSN) // 若为 Lindorm 用对应驱动
	if err != nil {
		log.Fatalf("open db error: %v", err)
	}
	defer db.Close()

	// 第一步：查询所有符合条件的 pool_address
	rows, err := db.Query("SELECT pool_address FROM pool WHERE dex = 6 AND account_key = 0")
	if err != nil {
		log.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	var addresses []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			log.Fatalf("scan error: %v", err)
		}
		addresses = append(addresses, addr)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("row error: %v", err)
	}

	log.Printf("Found %d records to delete", len(addresses))

	// 第二步：按 50 条一批进行删除
	for i := 0; i < len(addresses); i += batchSize {
		end := i + batchSize
		if end > len(addresses) {
			end = len(addresses)
		}
		if err := deleteBatch(db, addresses[i:end]); err != nil {
			log.Fatalf("delete batch failed: %v", err)
		}
		log.Printf("Deleted batch: %d - %d", i, end)
	}
}

func deleteBatch(db *sql.DB, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	query := "DELETE FROM pool WHERE "
	args := make([]interface{}, 0, len(addresses))
	for i, addr := range addresses {
		if i > 0 {
			query += " OR "
		}
		query += "(pool_address = ? AND account_key = ?)"
		args = append(args, addr, 0)
	}

	_, err := db.Exec(query, args...)
	return err
}
