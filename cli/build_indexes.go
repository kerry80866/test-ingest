package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	indexes := []string{
		"idx_user_token_type_id_desc",
	}

	for _, idx := range indexes {
		sql := fmt.Sprintf("BUILD INDEX %s ON chain_event", idx)
		log.Printf("正在构建索引: %s", idx)

		_, err := db.Exec(sql)
		if err != nil {
			log.Printf("❌ 构建失败: %v", err)
		} else {
			log.Printf("✅ 构建成功: %s", idx)
		}
	}
}
