package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	args := os.Args
	if len(args) < 3 || len(args) > 5 {
		fmt.Printf("用法: %s <user_wallet> <limit> [event_id] [event_type]\n", args[0])
		os.Exit(1)
	}

	userWallet := args[1]
	limit, err := strconv.Atoi(args[2])
	if err != nil || limit <= 0 {
		log.Fatalf("limit 应为正整数")
	}

	var (
		query  string
		params []any
	)

	query = `SELECT event_id, event_type, pool_address, token, quote_token, token_amount, quote_amount, volume_usd, price_usd, block_time
		FROM chain_event
		WHERE user_wallet = ?`
	params = append(params, userWallet)

	if len(args) >= 4 {
		eventID, err := strconv.ParseInt(args[3], 10, 64)
		if err != nil {
			log.Fatalf("event_id 格式不合法: %v", err)
		}
		query += " AND event_id < ?"
		params = append(params, eventID)
	}

	if len(args) == 5 {
		eventType, err := strconv.Atoi(args[4])
		if err != nil {
			log.Fatalf("event_type 应为整数: %v", err)
		}
		query += " AND event_type = ?"
		params = append(params, eventType)
	}

	query += " ORDER BY event_id DESC LIMIT ?"
	params = append(params, limit)

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

	fmt.Println("✅ 查询结果:")
	count := 0
	for rows.Next() {
		var (
			eventID, eventType                            int64
			pool, token, quote                            string
			tokenAmount, quoteAmount, volumeUsd, priceUsd float64
			blockTime                                     int64
		)
		err := rows.Scan(&eventID, &eventType, &pool, &token, &quote, &tokenAmount, &quoteAmount, &volumeUsd, &priceUsd, &blockTime)
		if err != nil {
			log.Fatalf("扫描失败: %v", err)
		}
		fmt.Printf("event_id=%d type=%d token=%s/%s amount=%.3f/%.3f price=%.4f time=%d\n",
			eventID, eventType, token, quote, tokenAmount, quoteAmount, priceUsd, blockTime)
		count++
	}
	if count == 0 {
		fmt.Println("❌ 没查到记录")
	}
}
