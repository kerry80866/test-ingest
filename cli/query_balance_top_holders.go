package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql" // 替换为你的 Lindorm 驱动
)

func main() {
	args := os.Args
	if len(args) < 2 || len(args) > 3 {
		fmt.Printf("用法:\n")
		fmt.Printf("  %s <token_address> [limit]\n", args[0])
		os.Exit(1)
	}

	token := args[1]
	limit := 100 // 默认 limit

	if len(args) == 3 {
		var err error
		limit, err = strconv.Atoi(args[2])
		if err != nil || limit <= 0 {
			fmt.Fprintf(os.Stderr, "无效的 limit 参数: %v\n", args[2])
			os.Exit(1)
		}
	}

	// 示例输出
	fmt.Printf("Token Address: %s\n", token)
	fmt.Printf("Limit: %d\n", limit)

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}
	defer db.Close()

	query := `
SELECT owner_address, balance
FROM balance
WHERE token_address = ?
ORDER BY balance DESC
LIMIT ?;
`
	rows, err := db.Query(query, token, limit)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Printf("Top %d holders for token %s:\n", limit, token)
	for rows.Next() {
		var owner string
		var balance string
		if err := rows.Scan(&owner, &balance); err != nil {
			log.Fatalf("Scan error: %v", err)
		}
		fmt.Printf("  %s\t%s\n", owner, balance)
	}
}
