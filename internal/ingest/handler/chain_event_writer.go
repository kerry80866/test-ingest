package handler

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/model"
	"dex-ingest-sol/pkg/logger"
	"fmt"
	"github.com/redis/go-redis/v9"
	"runtime/debug"
	"strings"
	"time"
)

const (
	chainEventsBatchSize        = 1000
	chainEventsUpsertFieldCount = 17
)

var chainEventsValuePlaceholder = "(" + strings.Repeat("?,", chainEventsUpsertFieldCount-1) + "?)"

func InsertChainEvents(ctx context.Context, db *sql.DB, events []*model.ChainEvent) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("InsertChainEvents panic: %v\n%s", r, debug.Stack())
			err = fmt.Errorf("InsertChainEvents panic: %v", r)
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

		builder.WriteString(" ON DUPLICATE KEY IGNORE")

		query := builder.String()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("insert chain_event [%d:%d] failed: %w", i, end, err)
		}
	}
	return nil
}

// SyncPoolCache 根据事件同步 Redis 中的 Pool 缓存（用于前端查询）
func SyncPoolCache(ctx context.Context, redisClient *redis.Client, events []*model.ChainEvent) error {
	return nil
}
