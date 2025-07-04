package handler

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/db"
	"dex-ingest-sol/internal/pkg/logger"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
)

const (
	balanceBatchSize        = 2000 // delete / insert / update 都用这个
	balanceUpsertFieldCount = 5    // account_address, owner_address, token_address, balance, last_event_id
)

var balanceValuePlaceholder = genPlaceholders(balanceUpsertFieldCount)

func InsertBalances(ctx context.Context, db *sql.DB, balances []*model.Balance, isRealTime bool) error {
	if len(balances) == 0 {
		return nil
	}

	// 第一步：AccountAddress 去重，保留 lastEventID 最大的记录
	latestMap := make(map[string]*model.Balance, len(balances))
	for _, b := range balances {
		existing, ok := latestMap[b.AccountAddress]
		if !ok || b.LastEventID > existing.LastEventID {
			latestMap[b.AccountAddress] = b
		}
	}

	// 第二步：按 balance 分类成更新数组和删除数组
	toUpdate := make([]*model.Balance, 0, len(balances))
	toDelete := make([]*model.Balance, 0, max(10, len(balances)/3))
	for _, b := range latestMap {
		if b.Balance == "0" {
			toDelete = append(toDelete, b)
		} else {
			toUpdate = append(toUpdate, b)
		}
	}

	// 第三步：并发执行更新和删除
	var wg sync.WaitGroup
	var errInsert, errDelete error

	if len(toUpdate) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errInsert = insertOrUpdateBalances(ctx, db, toUpdate, isRealTime)
		}()
	}

	if len(toDelete) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errDelete = deleteBalances(ctx, db, toDelete, isRealTime)
		}()
	}

	wg.Wait()

	if errInsert != nil || errDelete != nil {
		var msgs []string
		if errInsert != nil {
			msgs = append(msgs, fmt.Sprintf("insertOrUpdate error: %v", errInsert))
		}
		if errDelete != nil {
			msgs = append(msgs, fmt.Sprintf("delete error: %v", errDelete))
		}
		return errors.New(strings.Join(msgs, "; "))
	}
	return nil
}

func insertOrUpdateBalances(ctx context.Context, db *sql.DB, balances []*model.Balance, isRealTime bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("insertOrUpdateBalances panic recovered: %v", r)
		}
	}()

	if isRealTime {
		return upsertBalancesRealtime(ctx, db, balances)
	}
	return updateBalancesHistorical(ctx, db, balances)
}

func deleteBalances(ctx context.Context, db *sql.DB, balances []*model.Balance, isRealTime bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("deleteBalances panic: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("deleteBalances panic recovered: %v", r)
		}
	}()

	if isRealTime {
		return deleteBalancesRealtime(ctx, db, balances)
	}
	return deleteBalancesHistorical(ctx, db, balances)
}

func updateBalancesHistorical(ctx context.Context, db *sql.DB, balances []*model.Balance) error {
	toUpsert, err := filterBalancesByLastEventID(ctx, db, balances, false)
	if err != nil {
		return err
	}
	if len(toUpsert) == 0 {
		return nil
	}

	// 执行批量 UPSERT
	return upsertBalancesRealtime(ctx, db, toUpsert)
}

func upsertBalancesRealtime(ctx context.Context, dbConn *sql.DB, balances []*model.Balance) error {
	total := len(balances)
	if total == 0 {
		return nil
	}

	estimatedSqlLengthPerRow := len(balanceValuePlaceholder) + 32 // 预估 SQL 长度，避免 builder 扩容
	for i := 0; i < total; i += balanceBatchSize {
		end := i + balanceBatchSize
		if end > total {
			end = total
		}
		batch := balances[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedSqlLengthPerRow)
		builder.WriteString("INSERT INTO balance(account_address,owner_address,token_address,balance,last_event_id) VALUES")

		args := make([]any, 0, len(batch)*balanceUpsertFieldCount)
		for j, b := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(balanceValuePlaceholder)
			args = append(args,
				b.AccountAddress,
				b.OwnerAddress,
				b.TokenAddress,
				b.Balance,
				b.LastEventID,
			)
		}

		query := builder.String()

		retryRange := fmt.Sprintf("[%d:%d]", i, end)
		err := db.RetryWithBackoff(ctx, 30, func() error {
			_, execErr := dbConn.ExecContext(ctx, query, args...)
			if execErr != nil {
				logger.Warnf("retrying balance upsert %s: %v", retryRange, execErr)
			}
			return execErr
		})
		if err != nil {
			return fmt.Errorf("upsert balance %s failed after retries: %w (first account: %s)", retryRange, err, batch[0].AccountAddress)
		}
	}

	return nil
}

func deleteBalancesHistorical(ctx context.Context, db *sql.DB, balances []*model.Balance) error {
	toDelete, err := filterBalancesByLastEventID(ctx, db, balances, true)
	if err != nil {
		return err
	}
	if len(toDelete) == 0 {
		return nil
	}

	// 执行批量 DELETE
	return deleteBalancesRealtime(ctx, db, toDelete)
}

func deleteBalancesRealtime(ctx context.Context, dbConn *sql.DB, balances []*model.Balance) error {
	if len(balances) == 0 {
		return nil
	}

	for i := 0; i < len(balances); i += balanceBatchSize {
		end := i + balanceBatchSize
		if end > len(balances) {
			end = len(balances)
		}
		batch := balances[i:end]

		// 提取 address
		args := make([]any, len(batch))
		for j, b := range batch {
			args[j] = b.AccountAddress
		}

		placeholders := strings.Repeat("?,", len(batch))
		placeholders = placeholders[:len(placeholders)-1] // 去掉最后一个逗号
		query := "DELETE FROM balance WHERE account_address IN (" + placeholders + ")"

		retryRange := fmt.Sprintf("[%d:%d]", i, end)
		err := db.RetryWithBackoff(ctx, 30, func() error {
			_, execErr := dbConn.ExecContext(ctx, query, args...)
			if execErr != nil {
				logger.Warnf("retrying balance delete %s: %v", retryRange, execErr)
			}
			return execErr
		})
		if err != nil {
			return fmt.Errorf("deleteBalancesRealtime %s failed after retries: %w (first account: %s)", retryRange, err, batch[0].AccountAddress)
		}
	}
	return nil
}

// filterBalancesByLastEventID 查询已有 last_event_id，返回需要执行操作的 balances（isDelete 表示是否为删除操作）
func filterBalancesByLastEventID(ctx context.Context, dbConn *sql.DB, balances []*model.Balance, isDelete bool) ([]*model.Balance, error) {
	if len(balances) == 0 {
		return nil, nil
	}

	// 构建地址映射
	addrToBalance := make(map[string]*model.Balance, len(balances))
	addresses := make([]string, 0, len(balances))
	for _, b := range balances {
		addrToBalance[b.AccountAddress] = b
		addresses = append(addresses, b.AccountAddress)
	}

	// 查询数据库中已有的 last_event_id
	existing := make(map[string]int64, len(balances))
	for i := 0; i < len(addresses); i += balanceBatchSize {
		end := i + balanceBatchSize
		if end > len(addresses) {
			end = len(addresses)
		}
		batch := addresses[i:end]

		placeholders := strings.Repeat("?,", len(batch))
		placeholders = placeholders[:len(placeholders)-1] // 去掉最后一个逗号
		query := "SELECT account_address, last_event_id FROM balance WHERE account_address IN (" + placeholders + ")"

		args := make([]any, len(batch))
		for j, addr := range batch {
			args[j] = addr
		}

		err := db.RetryWithBackoff(ctx, 30, func() error {
			rows, queryErr := dbConn.QueryContext(ctx, query, args...)
			if queryErr != nil {
				logger.Warnf("retrying select balances [%d:%d]: %v", i, end, queryErr)
				return queryErr
			}
			defer rows.Close()

			var addr string
			var lastID int64
			for rows.Next() {
				if scanErr := rows.Scan(&addr, &lastID); scanErr != nil {
					return fmt.Errorf("scan error in [%d:%d]: %w", i, end, scanErr)
				}
				existing[addr] = lastID
			}
			return rows.Err()
		})
		if err != nil {
			return nil, err
		}
	}

	// 根据 last_event_id 判断是否需要处理
	result := make([]*model.Balance, 0, len(balances))
	if isDelete {
		for addr, b := range addrToBalance {
			lastID, ok := existing[addr]
			if ok && b.LastEventID >= lastID {
				result = append(result, b)
			}
		}
	} else {
		for addr, b := range addrToBalance {
			lastID, ok := existing[addr]
			if !ok || b.LastEventID > lastID {
				result = append(result, b)
			}
		}
	}
	return result, nil
}
