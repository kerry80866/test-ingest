package handler

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/db"
	"dex-ingest-sol/internal/pkg/logger"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	poolBatchSize        = 1000
	poolUpsertFieldCount = 9
)

var poolValuePlaceholder = "(" + strings.Repeat("?,", poolUpsertFieldCount-1) + "?)"

func InsertPools(ctx context.Context, dbConn *sql.DB, pools []*model.Pool) error {
	uniquePools := dedupAndValidatePools(pools)
	if len(uniquePools) == 0 {
		return nil
	}

	poolCount := len(uniquePools)
	batchSize := 100
	maxWorkers := 4

	if poolCount > batchSize*maxWorkers {
		batchSize = (poolCount + maxWorkers - 1) / maxWorkers
	}

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	for i := 0; i < poolCount; i += batchSize {
		end := i + batchSize
		if end > poolCount {
			end = poolCount
		}
		batch := uniquePools[i:end]

		wg.Add(1)
		go func(batch []*model.Pool, start, end int) {
			defer wg.Done()
			if err := insertPoolsSerial(ctx, dbConn, batch, start, end); err != nil {
				once.Do(func() {
					firstErr = err
				})
			}
		}(batch, i, end)
	}

	wg.Wait()
	return firstErr
}

func insertPoolsSerial(ctx context.Context, dbConn *sql.DB, pools []*model.Pool, startIndex, endIndex int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("insertPoolsSerial panic [%d:%d]: %v\n%s", startIndex, endIndex, r, debug.Stack())
			err = fmt.Errorf("insertPoolsSerial panic: %v", r)
		}
	}()

	total := len(pools)
	estimatedRowSQLSize := len(poolValuePlaceholder) + 32

	const upsertSQL = " ON DUPLICATE KEY UPDATE " +
		"token_account=VALUES(token_account)," +
		"quote_account=VALUES(quote_account)," +
		"dex=VALUES(dex)," +
		"token_address=VALUES(token_address)," +
		"quote_address=VALUES(quote_address)," +
		"update_at=VALUES(update_at)," +
		"create_at=IF(create_at=0, VALUES(create_at), create_at)"

	for i := 0; i < total; i += poolBatchSize {
		end := i + poolBatchSize
		if end > total {
			end = total
		}
		batch := pools[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedRowSQLSize)

		builder.WriteString("INSERT INTO pool(" +
			"pool_address,account_key,dex,token_address,quote_address," +
			"token_account,quote_account,create_at,update_at) VALUES")

		args := make([]any, 0, len(batch)*poolUpsertFieldCount)
		for j, p := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(poolValuePlaceholder)
			args = append(args,
				p.PoolAddress, p.AccountKey, p.Dex, p.TokenAddress, p.QuoteAddress,
				p.TokenAccount, p.QuoteAccount, p.CreateAt, p.UpdateAt,
			)
		}

		builder.WriteString(upsertSQL)

		query := builder.String()
		retryRange := fmt.Sprintf("[%d:%d]", startIndex+i, startIndex+end)

		err = db.RetryWithBackoff(ctx, 30, func() error {
			_, execErr := dbConn.ExecContext(ctx, query, args...)
			if execErr != nil {
				logger.Warnf("retrying pool insert %s: %v", retryRange, execErr)
			}
			return execErr
		})
		if err != nil {
			return fmt.Errorf("insert pool %s failed after retries: %w", retryRange, err)
		}
	}

	return nil
}

func dedupAndValidatePools(pools []*model.Pool) []*model.Pool {
	if len(pools) == 0 {
		return nil
	}

	poolMap := make(map[string]*model.Pool, len(pools))
	for _, p := range pools {
		key := fmt.Sprintf("%s|%d|%s|%s", p.PoolAddress, p.AccountKey, p.TokenAccount, p.QuoteAccount)
		existing, ok := poolMap[key]
		if !ok {
			poolMap[key] = p
			continue
		}

		// 字段冲突检查（可选，防御性编程）
		var conflicts []string
		if p.Dex != existing.Dex {
			conflicts = append(conflicts, fmt.Sprintf("Dex: %d != %d", p.Dex, existing.Dex))
		}
		if p.TokenAddress != existing.TokenAddress {
			conflicts = append(conflicts, fmt.Sprintf("TokenAddress: %s != %s", p.TokenAddress, existing.TokenAddress))
		}
		if p.QuoteAddress != existing.QuoteAddress {
			conflicts = append(conflicts, fmt.Sprintf("QuoteAddress: %s != %s", p.QuoteAddress, existing.QuoteAddress))
		}
		if len(conflicts) > 0 {
			logger.Warnf("conflict pool definition for %s (dex=%d):\n  %s",
				p.PoolAddress, p.Dex, strings.Join(conflicts, "\n  "))
			continue
		}

		// 选择更早的 createAt
		if (existing.CreateAt == 0 && p.CreateAt != 0) || (p.CreateAt < existing.CreateAt && p.CreateAt != 0) {
			existing.CreateAt = p.CreateAt
		}
	}

	now := int32(time.Now().Unix())
	result := make([]*model.Pool, 0, len(poolMap))
	for _, p := range poolMap {
		p.UpdateAt = now
		result = append(result, p)
	}
	return result
}
