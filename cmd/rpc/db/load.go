package db

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func LoadTokenDecimals(db *sql.DB, tokenMap map[string]int16) {
	lastTokenAddr := ""
	for {
		var finished bool
		lastTokenAddr, finished = loadDecimals(db, lastTokenAddr, tokenMap)
		log.Printf("[loadTokenDecimals] current size: %d", len(tokenMap))

		if finished {
			log.Printf("[loadTokenDecimals] finished, total: %d tokens loaded", len(tokenMap))
			return
		}
		time.Sleep(20 * time.Millisecond) // 小延迟防止打满数据库
	}
}

func loadDecimals(db *sql.DB, lastTokenAddr string, tokenMap map[string]int16) (string, bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("loadDecimals panic: %v", r)
		}
	}()

	sqlQuery := `
		SELECT token_address, token_decimals, quote_address, quote_decimals
		FROM pool
		WHERE token_address > ?
		ORDER BY token_address ASC
		LIMIT 1000`

	ctx := context.Background()
	rows, err := db.QueryContext(ctx, sqlQuery, lastTokenAddr)
	if err != nil {
		log.Printf("query error: %v", err)
		return lastTokenAddr, false // 返回 false 表示失败，不退出主循环
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tokenAddr string
		var tokenDecimals int16
		var quoteAddr string
		var quoteDecimals int16
		if err := rows.Scan(&tokenAddr, &tokenDecimals, &quoteAddr, &quoteDecimals); err != nil {
			log.Printf("scan error: %v", err)
			return lastTokenAddr, false
		}
		tokenMap[tokenAddr] = tokenDecimals
		tokenMap[quoteAddr] = quoteDecimals
		lastTokenAddr = tokenAddr
		count++
	}

	if err := rows.Err(); err != nil {
		log.Printf("rows iteration error: %v", err)
		return lastTokenAddr, false
	}

	// count == 0 表示已经查不到更多数据了，finished=true
	return lastTokenAddr, count == 0
}
