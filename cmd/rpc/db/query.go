package db

import (
	"context"
	"database/sql"
	"dex-ingest-sol/cmd/rpc/common"
	"fmt"
	"time"
)

func Query(db *sql.DB) []*common.LindormResult {
	for {
		list, e := query(db)
		if e != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		return list
	}
}

func query(db *sql.DB) (l []*common.LindormResult, e error) {
	defer func() {
		if r := recover(); r != nil {
			l = nil
			e = fmt.Errorf("insertChainEventsSerial panic: %v", r)
		}
	}()

	sqlQuery := `
		SELECT pool_address, account_key, token_address, quote_address
		FROM pool
		WHERE  token_decimals IS NULL LIMIT 490`

	ctx := context.Background()
	// 执行查询
	rows, err := db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// 解析结果
	result := make([]*common.LindormResult, 0, 200)
	for rows.Next() {
		r := &common.LindormResult{}
		if queryErr := rows.Scan(
			&r.PoolAddress, &r.AccountKey, &r.TokenAddress, &r.QuoteAddress,
		); queryErr != nil {
			return nil, queryErr
		}

		result = append(result, r)
	}

	if queryErr := rows.Err(); queryErr != nil {
		return nil, queryErr
	}

	return result, nil
}
