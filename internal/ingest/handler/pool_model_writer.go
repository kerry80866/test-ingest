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
	"time"
)

const (
	poolBatchSize        = 1000
	poolUpsertFieldCount = 8
)

var poolValuePlaceholder = "(" + strings.Repeat("?,", poolUpsertFieldCount-1) + "?)"

func InsertPools(ctx context.Context, dbConn *sql.DB, pools []*model.Pool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("InsertPools panic: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("InsertPools panic: %v", r)
		}
	}()

	uniquePools := dedupAndValidatePools(pools)
	if len(uniquePools) == 0 {
		return nil
	}

	estimatedRowSQLSize := len(poolValuePlaceholder) + 32
	for i := 0; i < len(uniquePools); i += poolBatchSize {
		end := i + poolBatchSize
		if end > len(uniquePools) {
			end = len(uniquePools)
		}
		batch := uniquePools[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedRowSQLSize)
		builder.WriteString("INSERT INTO pool(" +
			"pool_address,dex,token_address,quote_address," +
			"token_account,quote_account,create_at,update_at) VALUES")

		args := make([]any, 0, len(batch)*poolUpsertFieldCount)
		for j, p := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(poolValuePlaceholder)
			args = append(args,
				p.PoolAddress, p.Dex, p.TokenAddress, p.QuoteAddress,
				p.TokenAccount, p.QuoteAccount, p.CreateAt, p.UpdateAt,
			)
		}

		// 主键冲突忽略（Lindorm 会做版本保留）
		builder.WriteString(" ON DUPLICATE KEY IGNORE")

		query := builder.String()
		retryRange := fmt.Sprintf("[%d:%d]", i, end)
		err = db.RetryWithBackoff(ctx, 10, func() error {
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
		key := fmt.Sprintf("%s|%s|%s", p.PoolAddress, p.TokenAccount, p.QuoteAccount)
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
