package db

import (
	"context"
	"database/sql"
	"dex-ingest-sol/cmd/rpc/common"
	"log"
	"time"
)

func LoadPools(db *sql.DB, lastAddr string) (string, []*common.Pool) {
	for {
		var pools []*common.Pool
		lastAddr, pools = loadPools(db, lastAddr)
		if pools != nil {
			return lastAddr, pools
		}
		time.Sleep(20 * time.Millisecond) // 小延迟防止打满数据库
	}
}

func loadPools(db *sql.DB, lastAddr string) (string, []*common.Pool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("loadPools panic: %v", r)
		}
	}()

	sqlQuery := `
		SELECT pool_address, dex, token_decimals, quote_decimals,
		       token_address, quote_address, token_account, quote_account,
		       create_at, update_at
		FROM pool
		WHERE pool_address > ?
		ORDER BY pool_address ASC
		LIMIT 1000`

	ctx := context.Background()
	rows, err := db.QueryContext(ctx, sqlQuery, lastAddr)
	if err != nil {
		log.Printf("query error: %v", err)
		return lastAddr, nil
	}
	defer rows.Close()

	pools := make([]*common.Pool, 0, 1000)
	for rows.Next() {
		p := &common.Pool{}
		var createAt *int32
		if err := rows.Scan(
			&p.PoolAddress, &p.Dex, &p.TokenDecimals, &p.QuoteDecimals,
			&p.TokenAddress, &p.QuoteAddress, &p.TokenAccount, &p.QuoteAccount,
			&createAt, &p.UpdateAt,
		); err != nil {
			log.Printf("scan error: %v", err)
			return lastAddr, nil
		}

		if createAt != nil {
			p.CreateAt = *createAt
		}
		pools = append(pools, p)
		lastAddr = p.PoolAddress
	}

	if err := rows.Err(); err != nil {
		log.Printf("rows iteration error: %v", err)
		return lastAddr, nil
	}

	return lastAddr, pools
}
