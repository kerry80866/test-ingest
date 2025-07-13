package chainevent

import (
	"context"
	"dex-ingest-sol/internal/pkg/db"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
	"time"
)

func (s *QueryChainEventService) QueryEventsByPool(ctx context.Context, req *pb.PoolEventReq) (_ *pb.EventResp, err error) {
	const (
		ErrCodeBase        = 60600
		ErrCodePanic       = ErrCodeBase + 32
		ErrCodeInvalidArg  = ErrCodeBase + 1
		ErrCodeQueryFailed = ErrCodeBase + 2
		ErrCodeScanFailed  = ErrCodeBase + 3
		ErrCodeRowsIter    = ErrCodeBase + 4
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryEventsByPool: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	const (
		DefaultLimit = 10
		MaxLimit     = 1000
	)

	if req.PoolAddress == "" {
		return nil, status.Errorf(codes.Internal, "[%d] pool_address is required", ErrCodeInvalidArg)
	}

	var (
		query  strings.Builder
		key    strings.Builder
		params []any
	)

	query.WriteString(`
		SELECT event_id, event_type, dex, user_wallet, to_wallet,
		       pool_address, token, quote_token, token_amount, quote_amount,
		       volume_usd, price_usd, tx_hash, signer, block_time, create_at
		FROM chain_event
		WHERE pool_address = ?
	`)
	params = append(params, req.PoolAddress)
	key.WriteString(req.PoolAddress)

	// 安全地构建事件类型列表
	eventTypes := req.EventType
	if len(eventTypes) == 0 {
		eventTypes = []uint32{
			uint32(pb.EventType_TRADE_BUY),
			uint32(pb.EventType_TRADE_SELL),
			uint32(pb.EventType_ADD_LIQUIDITY),
			uint32(pb.EventType_REMOVE_LIQUIDITY),
			uint32(pb.EventType_BURN),
		}
	}

	// 拼接 IN 子句
	query.WriteString(" AND event_type IN (")
	query.WriteString(strings.TrimRight(strings.Repeat("?,", len(eventTypes)), ","))
	query.WriteString(")")
	for _, et := range eventTypes {
		params = append(params, et)
		key.WriteByte(':')
		key.WriteString(strconv.FormatUint(uint64(et), 16))
	}

	// 游标翻页
	if req.EventId != nil && *req.EventId != 0 {
		query.WriteString(" AND event_id < ?")
		params = append(params, *req.EventId)
		key.WriteString(":i")
		key.WriteString(strconv.FormatUint(*req.EventId, 16))
	}

	query.WriteString(" ORDER BY event_id DESC")

	// 限制返回条数
	limit := DefaultLimit
	if req.Limit != nil && *req.Limit > 0 {
		limit = int(*req.Limit)
		if limit > MaxLimit {
			logger.Warnf("QueryEventsByPool: limit too large (%d), trimmed to %d", limit, MaxLimit)
			limit = MaxLimit
		}
	}
	query.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	key.WriteString(":l")
	key.WriteString(strconv.FormatUint(uint64(limit), 16))

	resp, localErr := chainEventsByPoolCache.Do(key.String(), false, func(e *db.Entry, onlyReady bool) (resp any, localErr error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in QueryEventsByPool cache func: %v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
				resp = nil
			}
		}()

		if !e.IsExpired() {
			cached, success := e.Result.([]*pb.ChainEvent)
			if success {
				return &pb.EventResp{Events: cached}, nil
			}
		}
		if onlyReady {
			return nil, status.Errorf(codes.NotFound, "cache not ready")
		}

		// 执行查询
		rows, queryErr := s.DB.QueryContext(ctx, query.String(), params...)
		if queryErr != nil {
			logger.Errorf("QueryEventsByPool query failed, req=%+v, err=%v", req, queryErr)
			return nil, status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
		}
		defer rows.Close()

		// 解析结果
		result := make([]*pb.ChainEvent, 0, limit)
		for rows.Next() {
			ev := &pb.ChainEvent{}
			var tokenAmount, quoteAmount string
			if queryErr = rows.Scan(
				&ev.EventId, &ev.EventType, &ev.Dex,
				&ev.UserWallet, &ev.ToWallet, &ev.PoolAddress,
				&ev.Token, &ev.QuoteToken, &tokenAmount, &quoteAmount,
				&ev.VolumeUsd, &ev.PriceUsd, &ev.TxHash, &ev.Signer,
				&ev.BlockTime, &ev.CreateAt,
			); queryErr != nil {
				logger.Errorf("QueryEventsByPool row scan failed: %v", queryErr)
				return nil, status.Errorf(codes.Internal, "[%d] failed to parse event data", ErrCodeScanFailed)
			}
			if ev.Signer == "" {
				ev.Signer = ev.UserWallet
			}
			ev.EventIdHash = uint32(utils.EventIdHash(ev.EventId))
			ev.TokenAmount = utils.ParseUint64(tokenAmount)
			ev.QuoteAmount = utils.ParseUint64(quoteAmount)
			ev.Token = utils.DecodeTokenAddress(ev.Token)
			ev.QuoteToken = utils.DecodeTokenAddress(ev.QuoteToken)

			result = append(result, ev)
		}

		if queryErr = rows.Err(); queryErr != nil {
			logger.Errorf("QueryEventsByPool rows iteration error: %v", queryErr)
			return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
		}

		e.Result = result
		if len(result) == 0 {
			e.SetValidAt(time.Now().Add(chainEventsByPoolEmptyTTL))
		} else {
			e.SetValidAt(time.Now().Add(chainEventsByPoolTTL))
		}
		return &pb.EventResp{Events: result}, nil
	})

	if r, ok := resp.(*pb.EventResp); ok {
		return r, nil
	}
	return nil, localErr
}
