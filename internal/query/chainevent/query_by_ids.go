package chainevent

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
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

	const maxEventQueryCount = 500

	ids := req.EventIds
	if len(ids) == 0 {
		return &pb.EventListResp{}, nil
	}
	if len(ids) > maxEventQueryCount {
		return nil, status.Errorf(codes.Internal, "[%d] at most %d events can be queried", ErrCodeParamTooMany, maxEventQueryCount)
	}

	var query strings.Builder
	query.WriteString(`SELECT event_id, event_type, dex, user_wallet, to_wallet,
		pool_address, token, quote_token, token_amount, quote_amount,
		volume_usd, price_usd, tx_hash, signer, block_time, create_at
		FROM chain_event WHERE `)

	args := make([]any, 0, len(ids)*2)
	for i, eventID := range ids {
		if i > 0 {
			query.WriteString(" OR ")
		}
		hash := utils.EventIdHash(eventID)
		query.WriteString("(event_id_hash = ? AND event_id = ?)")
		args = append(args, hash, eventID)
	}

	rows, err := s.DB.QueryContext(ctx, query.String(), args...)
	if err != nil {
		logger.Errorf("QueryEventsByIDs query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query failed: %v", ErrCodeQueryFailed, err)
	}
	defer rows.Close()

	resultMap := make(map[uint64]*pb.ChainEvent, len(ids))
	for rows.Next() {
		ev := &pb.ChainEvent{}
		var tokenAmount, quoteAmount string
		if err := rows.Scan(
			&ev.EventId, &ev.EventType, &ev.Dex,
			&ev.UserWallet, &ev.ToWallet, &ev.PoolAddress,
			&ev.Token, &ev.QuoteToken, &tokenAmount, &quoteAmount,
			&ev.VolumeUsd, &ev.PriceUsd, &ev.TxHash, &ev.Signer,
			&ev.BlockTime, &ev.CreateAt,
		); err != nil {
			logger.Errorf("QueryEventsByIDs scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] failed to parse data: %v", ErrCodeQueryFailed, err)
		}
		if ev.Signer == "" {
			ev.Signer = ev.UserWallet
		}
		ev.EventIdHash = uint32(utils.EventIdHash(ev.EventId))
		ev.TokenAmount = utils.ParseUint64(tokenAmount)
		ev.QuoteAmount = utils.ParseUint64(quoteAmount)
		ev.Token = utils.DecodeTokenAddress(ev.Token)
		ev.QuoteToken = utils.DecodeTokenAddress(ev.QuoteToken)

		resultMap[ev.EventId] = ev
	}
	if err := rows.Err(); err != nil {
		logger.Errorf("QueryEventsByIDs rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error: %v", ErrCodeQueryFailed, err)
	}

	// 按请求顺序组装结果，查不到对应 event_id 则 Event 为 nil
	results := make([]*pb.ChainEventResult, 0, len(ids))
	for _, eventID := range ids {
		results = append(results, &pb.ChainEventResult{
			EventId: eventID,
			Event:   resultMap[eventID],
		})
	}

	return &pb.EventListResp{Results: results}, nil
}
