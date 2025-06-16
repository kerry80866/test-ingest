package handler

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/model"
	"dex-ingest-sol/pkg/logger"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	tokenBatchSize        = 1000
	tokenUpsertFieldCount = 10
)

var tokenValuePlaceholder = "(" + strings.Repeat("?,", tokenUpsertFieldCount-1) + "?)"

func InsertTokens(ctx context.Context, db *sql.DB, tokens []*model.Token) error {
	insertList, updateList := splitTokensForInsertAndUpdate(tokens)
	if len(insertList) == 0 && len(updateList) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var errInsert, errUpdate error

	if len(insertList) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errInsert = execTokenBatch(ctx, db, insertList, false)
		}()
	}

	if len(updateList) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errUpdate = execTokenBatch(ctx, db, updateList, true)
		}()
	}

	wg.Wait()

	if errInsert != nil || errUpdate != nil {
		var msgs []string
		if errInsert != nil {
			msgs = append(msgs, fmt.Sprintf("insert error: %v", errInsert))
		}
		if errUpdate != nil {
			msgs = append(msgs, fmt.Sprintf("update error: %v", errUpdate))
		}
		return errors.New(strings.Join(msgs, "; "))
	}

	return nil
}

func splitTokensForInsertAndUpdate(tokens []*model.Token) (insertList []*model.Token, updateList []*model.Token) {
	if len(tokens) == 0 {
		return nil, nil
	}

	tokenMap := make(map[string]*model.Token, len(tokens))

	for _, t := range tokens {
		existing, ok := tokenMap[t.TokenAddress]
		if !ok {
			tokenMap[t.TokenAddress] = t
			continue
		}

		// Decimals 不一致报警
		if t.Decimals != existing.Decimals {
			logger.Warnf("conflict decimals for token %s: %d != %d",
				t.TokenAddress, existing.Decimals, t.Decimals)
		}

		// Launchpad 的 creator 非空，优先更新
		if t.Creator != "" && existing.Creator == "" {
			tokenMap[t.TokenAddress] = t
		}
	}

	// 统一设置 UpdateAt，并分类 insert / update
	now := int32(time.Now().Unix())
	insertList = make([]*model.Token, 0, len(tokenMap))
	updateList = make([]*model.Token, 0, len(tokenMap))

	for _, t := range tokenMap {
		t.UpdateAt = now
		if t.Creator != "" {
			updateList = append(updateList, t)
		} else {
			insertList = append(insertList, t)
		}
	}

	return insertList, updateList
}

func execTokenBatch(ctx context.Context, db *sql.DB, tokens []*model.Token, update bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("execTokenBatch panic: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("execTokenBatch panic: %v", r)
		}
	}()

	estimatedRowSQLSize := len(tokenValuePlaceholder) + 32
	for i := 0; i < len(tokens); i += tokenBatchSize {
		end := i + tokenBatchSize
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := tokens[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedRowSQLSize)
		builder.WriteString("INSERT INTO token(" +
			"token_address,decimals,source,total_supply,name,symbol,uri,creator,create_at,update_at) VALUES")

		args := make([]any, 0, len(batch)*tokenUpsertFieldCount)
		for j, t := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(tokenValuePlaceholder)
			args = append(args,
				t.TokenAddress, t.Decimals, t.Source, t.TotalSupply,
				t.Name, t.Symbol, t.URI, t.Creator,
				t.CreateAt, t.UpdateAt,
			)
		}

		if update {
			builder.WriteString(" ON DUPLICATE KEY UPDATE " +
				"decimals=VALUES(decimals)," +
				"source=VALUES(source)," +
				"total_supply=VALUES(total_supply)," +
				"name=VALUES(name)," +
				"symbol=VALUES(symbol)," +
				"uri=VALUES(uri)," +
				"creator=VALUES(creator)," +
				"update_at=VALUES(update_at)")
		} else {
			builder.WriteString(" ON DUPLICATE KEY IGNORE")
		}

		query := builder.String()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			action := "insert"
			if update {
				action = "update"
			}
			return fmt.Errorf("%s token [%d:%d] failed: %w", action, i, end, err)
		}
	}
	return nil
}
