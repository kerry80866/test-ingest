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
		fmt.Printf("  %s <token_address>                     # 使用 idx_balance_token\n", args[0])
		fmt.Printf("  %s <token_address> <owner_address>     # 精确匹配 token+owner\n", args[0])
		fmt.Printf("  %s -o <owner_address>                  # 使用 idx_balance_owner_token\n", args[0])
		fmt.Printf("  %s -o <owner_address> <token_address>  # 精确匹配 owner+token\n", args[0])
		os.Exit(1)
	}

	query := ""
	params := []any{}
	showHolderCount := false
	var tokenAddress string

	if args[1] == "-o" {
		if len(args) == 3 {
			query = "SELECT * FROM balance WHERE owner_address = ? ORDER BY token_address LIMIT 20"
			params = append(params, args[2])
		} else if len(args) == 4 {
			query = "SELECT * FROM balance WHERE owner_address = ? AND token_address = ? LIMIT 1"
			params = append(params, args[2], args[3])
		} else {
			log.Fatalf("用法错误：缺少 owner_address")
		}
	} else {
		if len(args) == 2 {
			tokenAddress = args[1]
			showHolderCount = true
			query = "SELECT * FROM balance WHERE token_address = ? ORDER BY owner_address LIMIT 20"
			params = append(params, args[1])
		} else if len(args) == 3 {
			query = "SELECT * FROM balance WHERE token_address = ? AND owner_address = ? LIMIT 1"
			params = append(params, args[1], args[2])
		}
	}

	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 如果只传 token_address，先统计 holder 数量
	if showHolderCount {
		countQuery := "SELECT COUNT(DISTINCT owner_address) FROM balance WHERE token_address = ?"
		var holderCount int
		if err := db.QueryRow(countQuery, tokenAddress).Scan(&holderCount); err != nil {
			log.Fatalf("查询 holder 数量失败: %v", err)
		}
		fmt.Printf("📊 当前 holder 数量: %d\n", holderCount)
	}

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
