package handler

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/db"
	"dex-ingest-sol/internal/pkg/logger"
	"fmt"
	"github.com/redis/go-redis/v9"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	chainEventsBatchSize        = 1000
	chainEventsUpsertFieldCount = 17
)

var chainEventsValuePlaceholder = genPlaceholders(chainEventsUpsertFieldCount)

func InsertChainEvents(ctx context.Context, db *sql.DB, events []*model.ChainEvent) error {
	if len(events) == 0 {
		return nil
	}
	eventsCount := len(events)

	batchSize := 100
	maxWorkers := 6

	if len(events) > batchSize*maxWorkers {
		// ceil(eventsCount / workers)：将 events 平均分配给 workers 个批次
		batchSize = (eventsCount + maxWorkers - 1) / maxWorkers
	}

	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	for i := 0; i < eventsCount; i += batchSize {
		end := i + batchSize
		if end > eventsCount {
			end = eventsCount
		}
		batch := events[i:end]

		wg.Add(1)
		go func(evts []*model.ChainEvent) {
			defer wg.Done()
			if err := insertChainEventsSerial(ctx, db, evts); err != nil {
				once.Do(func() {
					firstErr = err
				})
			}
		}(batch)
	}

	wg.Wait()
	return firstErr
}

func insertChainEventsSerial(ctx context.Context, dbConn *sql.DB, events []*model.ChainEvent) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("insertChainEventsSerial panic: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("insertChainEventsSerial panic: %v", r)
		}
	}()

	estimatedSqlLengthPerRow := len(chainEventsValuePlaceholder) + 32
	total := len(events)

	for i := 0; i < total; i += chainEventsBatchSize {
		end := i + chainEventsBatchSize
		if end > total {
			end = total
		}
		batch := events[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedSqlLengthPerRow)

		builder.WriteString("INSERT INTO chain_event(" +
			"event_id_hash,event_id,event_type,dex," +
			"user_wallet,to_wallet,pool_address,token,quote_token," +
			"token_amount,quote_amount,volume_usd,price_usd," +
			"tx_hash,signer,block_time,create_at) VALUES")

		args := make([]any, 0, len(batch)*chainEventsUpsertFieldCount)
		createAt := int32(time.Now().Unix())
		for j, e := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(chainEventsValuePlaceholder)
			args = append(args,
				e.EventIDHash, e.EventID, e.EventType, e.Dex,
				e.UserWallet, e.ToWallet, e.PoolAddress, e.Token, e.QuoteToken,
				e.TokenAmount, e.QuoteAmount, e.VolumeUsd, e.PriceUsd,
				e.TxHash, e.Signer, e.BlockTime, createAt,
			)
		}

		query := builder.String()
		retryRange := fmt.Sprintf("[%d:%d]", i, end)

		err = db.RetryWithBackoff(ctx, func() error {
			_, execErr := dbConn.ExecContext(ctx, query, args...)
			if execErr != nil {
				logger.Warnf("retrying chain_event insert %s: %v", retryRange, execErr)
			}
			return execErr
		})
		if err != nil {
			return fmt.Errorf("insert chain_event %s failed after retries: %w", retryRange, err)
		}
	}
	return nil
}

// SyncPoolCache 根据事件同步 Redis 中的 Pool 缓存（用于前端查询）
func SyncPoolCache(ctx context.Context, redisClient *redis.Client, events []*model.ChainEvent) error {
	return nil
}
