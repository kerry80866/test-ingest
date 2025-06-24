package chainevent

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"encoding/binary"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
)

func (s *QueryChainEventService) QueryEventsByIDs(ctx context.Context, req *pb.EventIDsReq) (resp *pb.EventListResp, err error) {
	const (
		ErrCodeBase         = 60500
		ErrCodePanic        = ErrCodeBase + 32
		ErrCodeParamTooMany = ErrCodeBase + 1
		ErrCodeQueryFailed  = ErrCodeBase + 2
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryEventsByIDs: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	const (
		maxConcurrentQuery = 3
		maxEventQueryCount = 20
	)

	ids := req.EventIds
	if len(ids) == 0 {
		return &pb.EventListResp{}, nil
	}
	if len(ids) > maxEventQueryCount {
		return nil, status.Errorf(codes.Internal, "[%d] at most %d events can be queried", ErrCodeParamTooMany, maxEventQueryCount)
	}

	// 单条查询优化
	if len(ids) == 1 {
		ev, err := s.querySingleEventByID(ctx, ids[0])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[%d] query error", ErrCodeQueryFailed)
		}
		return &pb.EventListResp{
			Results: []*pb.ChainEventResult{
				{
					EventId: ids[0],
					Event:   ev,
				},
			},
		}, nil
	}

	results := make([]*pb.ChainEventResult, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentQuery)

	for i, eventID := range ids {
		i := i

		wg.Add(1)
		sem <- struct{}{}
		go func(eventID uint64) {
			defer wg.Done()
			defer func() { <-sem }()

			// panic recover
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("panic while querying event_id=%d: %v", eventID, r)
					results[i] = &pb.ChainEventResult{
						EventId: eventID,
						Event:   nil,
					}
				}
			}()

			ev, err := s.querySingleEventByID(ctx, eventID)
			if err != nil {
				// 可选记录日志，不阻断整体返回
				logger.Warnf("query event_id=%d failed: %v", eventID, err)
			}
			results[i] = &pb.ChainEventResult{
				EventId: eventID,
				Event:   ev, // ev 可能为 nil，表示未查到
			}
		}(eventID)
	}

	wg.Wait()

	return &pb.EventListResp{Results: results}, nil
}

func (s *QueryChainEventService) querySingleEventByID(ctx context.Context, eventID uint64) (*pb.ChainEvent, error) {
	hash := eventIdHash(eventID)

	query := `SELECT event_id, event_type, dex, user_wallet, to_wallet,
		pool_address, token, quote_token, token_amount, quote_amount,
		volume_usd, price_usd, tx_hash, signer, block_time, create_at
		FROM chain_event
		WHERE event_id_hash = ? AND event_id = ? LIMIT 1`

	row := s.DB.QueryRowContext(ctx, query, hash, eventID)
	ev := &pb.ChainEvent{}
	var tokenAmount, quoteAmount string
	err := row.Scan(
		&ev.EventId, &ev.EventType, &ev.Dex,
		&ev.UserWallet, &ev.ToWallet, &ev.PoolAddress,
		&ev.Token, &ev.QuoteToken, &tokenAmount, &quoteAmount,
		&ev.VolumeUsd, &ev.PriceUsd, &ev.TxHash, &ev.Signer,
		&ev.BlockTime, &ev.CreateAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query event_id=%d: %w", eventID, err)
	}
	if ev.Signer == "" {
		ev.Signer = ev.UserWallet
	}
	ev.TokenAmount = utils.ParseUint64(tokenAmount)
	ev.QuoteAmount = utils.ParseUint64(quoteAmount)
	ev.Token = utils.DecodeTokenAddress(ev.Token)
	ev.QuoteToken = utils.DecodeTokenAddress(ev.QuoteToken)

	return ev, nil
}

func eventIdHash(eventId uint64) int32 {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], eventId)
	h := xxhash.Sum64(buf[:])

	// 非连续位段采样（更分散），高位更随机
	part1 := (h >> 55) & 0x7F // bits 55~61 (7 bit)
	part2 := (h >> 45) & 0xFF // bits 45~52 (8 bit)
	part3 := (h >> 34) & 0xFF // bits 34~41 (8 bit)
	part4 := (h >> 3) & 0xFF  // bits 3~10  (8 bit) → 扰动

	// 拼接为 32 位 hash
	return int32((part1 << 24) | (part2 << 16) | (part3 << 8) | part4)
}
