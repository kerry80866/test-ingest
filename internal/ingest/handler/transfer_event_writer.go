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
	transferEventsBatchSize        = 1000
	transferEventsUpsertFieldCount = 11
)

var transferEventsValuePlaceholder = genPlaceholders(transferEventsUpsertFieldCount)

func InsertTransferEvents(ctx context.Context, dbConn *sql.DB, events []*model.TransferEvent) error {
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
		go func(start, end int, evts []*model.TransferEvent) {
			defer wg.Done()
			if err := insertTransferEventsSerial(ctx, dbConn, evts, start, end); err != nil {
				logger.Errorf("InsertTransferEvents batch [%d:%d] failed: %v", start, end, err)
				once.Do(func() {
					firstErr = err
				})
			}
		}(i, end, batch)
	}

	wg.Wait()
	return firstErr
}

// insertTransferEventsSerial 单批次插入 TransferEvent，带重试机制
func insertTransferEventsSerial(ctx context.Context, dbConn *sql.DB, events []*model.TransferEvent, startIndex, endIndex int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("insertTransferEventsSerial panic [%d:%d]: %v\n%s", startIndex, endIndex, r, debug.Stack())
			err = fmt.Errorf("insertTransferEventsSerial panic: %v", r)
		}
	}()

	estimatedSqlLengthPerRow := len(transferEventsValuePlaceholder) + 32
	total := len(events)

	for i := 0; i < total; i += transferEventsBatchSize {
		end := i + transferEventsBatchSize
		if end > total {
			end = total
		}
		batch := events[i:end]

		var builder strings.Builder
		builder.Grow(512 + len(batch)*estimatedSqlLengthPerRow)

		builder.WriteString("INSERT INTO transfer_event(" +
			"event_id_hash,event_id," +
			"from_wallet,to_wallet,token,amount,decimals," +
			"tx_hash,signer,block_time,create_at) VALUES")

		args := make([]any, 0, len(batch)*transferEventsUpsertFieldCount)
		createAt := int32(time.Now().Unix())
		for j, e := range batch {
			if j > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(transferEventsValuePlaceholder)
			args = append(args,
				e.EventIDHash, e.EventID,
				e.FromWallet, e.ToWallet, e.Token, e.Amount, e.Decimals,
				e.TxHash, e.Signer, e.BlockTime, createAt,
			)
		}

		query := builder.String()
		retryRange := fmt.Sprintf("[%d:%d sub=%d:%d]", startIndex, endIndex, i, end)

		err = db.RetryWithBackoff(ctx, func() error {
			_, execErr := dbConn.ExecContext(ctx, query, args...)
			if execErr != nil {
				logger.Warnf("retrying transfer_event insert %s: %v", retryRange, execErr)
			}
			return execErr
		})
		if err != nil {
			return fmt.Errorf("insert transfer_event %s failed after retries: %w", retryRange, err)
		}
	}
	return nil
}
