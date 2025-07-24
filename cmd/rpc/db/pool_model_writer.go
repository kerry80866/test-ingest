package db

import (
	"context"
	"database/sql"
	"dex-ingest-sol/cmd/rpc/common"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	poolBatchSize    = 1000
	poolFieldCount11 = 11 // 包含 create_at
	poolFieldCount10 = 10 // 不包含 create_at
)

func genPlaceholders(n int) string {
	if n <= 0 {
		return "()"
	}
	return "(" + strings.Repeat("?,", n-1) + "?)"
}

var (
	poolPlaceholders11 = genPlaceholders(poolFieldCount11)
	poolPlaceholders10 = genPlaceholders(poolFieldCount10)
)

func InsertPools(ctx context.Context, dbConn *sql.DB, pools []*common.Pool) error {
	withCreateAtPools, withoutCreateAtPools := dedupAndSplitPools(pools)

	const maxWorkers = 10
	const minBatchSize = 100
	withBatchSize, withoutBatchSize := allocateBatchSize(len(withCreateAtPools), len(withoutCreateAtPools), maxWorkers, minBatchSize)

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	dispatch := func(pools []*common.Pool, withCreateAt bool, batchSize int) {
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
			go func(batch []*common.Pool, start, end int) {
				defer wg.Done()
				if err := insertPoolsSerial(ctx, dbConn, batch, start, end, withCreateAt); err != nil {
					once.Do(func() {
						firstErr = err
					})
				}
			}(batch, i, end)
		}
	}

	dispatch(withCreateAtPools, true, withBatchSize)
	dispatch(withoutCreateAtPools, false, withoutBatchSize)

	wg.Wait()
	return firstErr
}

func insertPoolsSerial(ctx context.Context, dbConn *sql.DB, pools []*common.Pool, startIndex, endIndex int, fullField bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("insertPoolsSerial panic: %v", r)
		}
	}()

	total := len(pools)
	estimatedRowSQLSize := len(poolPlaceholders11) + 32

	for i := 0; i < total; i += poolBatchSize {
		end := i + poolBatchSize
		if end > total {
			end = total
		}
		batch := pools[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedRowSQLSize)

		var args []any
		if fullField {
			builder.WriteString("INSERT INTO pool(" +
				"pool_address,account_key,dex,token_decimals,quote_decimals,token_address,quote_address," +
				"token_account,quote_account,create_at,update_at) VALUES")
			args = make([]any, 0, len(batch)*poolFieldCount11)

			for j, p := range batch {
				if j > 0 {
					builder.WriteByte(',')
				}
				builder.WriteString(poolPlaceholders11)
				args = append(args,
					p.PoolAddress, p.AccountKey, p.Dex, p.TokenDecimals, p.QuoteDecimals,
					p.TokenAddress, p.QuoteAddress, p.TokenAccount, p.QuoteAccount, p.CreateAt, p.UpdateAt,
				)
			}
		} else {
			builder.WriteString("INSERT INTO pool(" +
				"pool_address,account_key,dex,token_decimals,quote_decimals,token_address,quote_address," +
				"token_account,quote_account,update_at) VALUES")
			args = make([]any, 0, len(batch)*poolFieldCount10)

			for j, p := range batch {
				if j > 0 {
					builder.WriteByte(',')
				}
				builder.WriteString(poolPlaceholders10)
				args = append(args,
					p.PoolAddress, p.AccountKey, p.Dex, p.TokenDecimals, p.QuoteDecimals,
					p.TokenAddress, p.QuoteAddress, p.TokenAccount, p.QuoteAccount, p.UpdateAt,
				)
			}
		}

		query := builder.String()
		retryRange := fmt.Sprintf("[%d:%d]", startIndex+i, startIndex+end)

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

func allocateBatchSize(withCount, withoutCount, maxWorkers, minBatchSize int) (withBatchSize, withoutBatchSize int) {
	totalCount := withCount + withoutCount
	if totalCount == 0 {
		return 0, 0
	}

	if maxWorkers < 2 {
		// fallback 保底值，避免出现负值或分配失败
		maxWorkers = 2
	}

	maxWithWorkers := int(math.Round(float64(withCount) * float64(maxWorkers) / float64(totalCount)))
	maxWithoutWorkers := maxWorkers - maxWithWorkers

	// 修正边界，确保只要有数据就至少分配一个 worker
	if maxWithWorkers == 0 && withCount > 0 {
		maxWithWorkers++

	} else if maxWithoutWorkers == 0 && withoutCount > 0 {
		maxWithWorkers--
		maxWithoutWorkers++
	}

	if withCount > 0 {
		if withCount > minBatchSize*maxWithWorkers {
			withBatchSize = (withCount + maxWithWorkers - 1) / maxWithWorkers
		} else {
			withBatchSize = minBatchSize
		}
	}

	if withoutCount > 0 {
		if withoutCount > minBatchSize*maxWithoutWorkers {
			withoutBatchSize = (withoutCount + maxWithoutWorkers - 1) / maxWithoutWorkers
		} else {
			withoutBatchSize = minBatchSize
		}
	}
	return withBatchSize, withoutBatchSize
}

func dedupAndSplitPools(pools []*common.Pool) (
	withCreateAt []*common.Pool,
	withoutCreateAt []*common.Pool,
) {
	if len(pools) == 0 {
		return nil, nil
	}

	poolMap := make(map[string]*common.Pool, len(pools))
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
			continue
		}

		// 选择更早的 createAt
		if (existing.CreateAt == 0 && p.CreateAt != 0) || (p.CreateAt < existing.CreateAt && p.CreateAt != 0) {
			existing.CreateAt = p.CreateAt
		}
	}

	now := int32(time.Now().Unix())
	withCreateAt = make([]*common.Pool, 0, len(poolMap))
	withoutCreateAt = make([]*common.Pool, 0, len(poolMap))
	for _, p := range poolMap {
		p.UpdateAt = now
		if p.CreateAt > 0 {
			withCreateAt = append(withCreateAt, p)
		} else {
			withoutCreateAt = append(withoutCreateAt, p)
		}
	}
	return
}

func retryWithBackoff(ctx context.Context, op func() error) error {
	const maxRetries = 100000

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

		// 计算下一次延迟

		// 日志放在此处，避免第一次 op() 之前就打印
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("retry failed after %d attempts: %w", maxRetries, err)
}
