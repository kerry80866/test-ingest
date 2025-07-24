package db

import (
	"context"
	"database/sql"
	"dex-ingest-sol/cmd/rpc/common"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const poolFieldCount = 4
const poolPlaceholders = "(?,?,?,?)"

func InsertPools(dbConn *sql.DB, pools []*common.LindormResult) error {
	if len(pools) == 0 {
		return nil
	}
	const batchSize = 100

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	dispatch := func(pools []*common.LindormResult, batchSize int) {
		poolCount := len(pools)
		if poolCount == 0 || batchSize == 0 {
			return
		}

		for i := 0; i < poolCount; i += batchSize {
			end := i + batchSize
			if end > poolCount {
				end = poolCount
			}
			batch := pools[i:end]

			wg.Add(1)
			go func(batch []*common.LindormResult, start int) {
				defer wg.Done()
				if err := insertPoolsSerial(dbConn, batch, start); err != nil {
					once.Do(func() {
						firstErr = err
					})
				}
			}(batch, i)
		}
	}

	dispatch(pools, batchSize)

	wg.Wait()
	return firstErr
}

func insertPoolsSerial(dbConn *sql.DB, pools []*common.LindormResult, startIndex int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("insertPoolsSerial panic: %v", r)
		}
	}()

	total := len(pools)
	estimatedRowSQLSize := len(poolPlaceholders) + 32

	const poolBatchSize = 1000
	for i := 0; i < total; i += poolBatchSize {
		end := i + poolBatchSize
		if end > total {
			end = total
		}
		batch := pools[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedRowSQLSize)

		var args []any
		builder.WriteString("INSERT INTO pool(" +
			"pool_address,account_key,token_decimals,quote_decimals) VALUES ")
		args = make([]any, 0, len(batch)*poolFieldCount)

		for j, p := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(poolPlaceholders)
			args = append(args,
				p.PoolAddress, p.AccountKey, p.TokenDecimals, p.QuoteDecimals,
			)
		}

		query := builder.String()
		retryRange := fmt.Sprintf("[%d:%d]", startIndex+i, startIndex+end)

		ctx := context.Background()
		err = retryWithBackoff(ctx, func() error {
			_, execErr := dbConn.ExecContext(ctx, query, args...)
			if execErr != nil {
			}
			return execErr
		})
		if err != nil {
			return fmt.Errorf("insert pool %s failed after retries: %w", retryRange, err)
		}
	}

	return nil
}

func retryWithBackoff(ctx context.Context, op func() error) error {
	const maxRetries = 10

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err() // 主动取消，不再重试
		default:
		}

		err = op()
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) {
			return err
		}

		// 日志放在此处，避免第一次 op() 之前就打印
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("retry failed after %d attempts: %w", maxRetries, err)
}
