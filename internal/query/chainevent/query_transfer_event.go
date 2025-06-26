package chainevent

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime/debug"
	"strings"
	"sync"
)

const (
	TransferErrCodeBase        = 61000
	TransferErrCodePanic       = TransferErrCodeBase + 32
	TransferErrCodeInvalidArg  = TransferErrCodeBase + 1
	TransferErrCodeQueryFailed = TransferErrCodeBase + 2
	TransferErrCodeScanFailed  = TransferErrCodeBase + 3
	TransferErrCodeRowsIter    = TransferErrCodeBase + 4
)

func (s *QueryChainEventService) QueryTransferEvents(ctx context.Context, req *pb.TransferEventQueryReq) (resp *pb.EventResp, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryTransferEvents: %v\n%s", r, debug.Stack())
			err = status.Errorf(codes.Internal, "[%d] server panic", TransferErrCodePanic)
		}
	}()

	userWallet := strings.TrimSpace(req.UserWallet)
	if userWallet == "" {
		return nil, status.Errorf(codes.Internal, "[%d] user_wallet is required", TransferErrCodeInvalidArg)
	}

	const (
		DefaultLimit = 20
		MaxLimit     = 1000
	)

	limit := DefaultLimit
	if req.Limit != nil && *req.Limit > 0 {
		limit = int(*req.Limit)
		if limit > MaxLimit {
			limit = MaxLimit
		}
	}

	var beforeEventID *uint64
	if req.EventId != nil && *req.EventId != 0 {
		beforeEventID = req.EventId
	}

	switch req.QueryType {
	case pb.TransferQueryType_FROM_WALLET:
		return s.queryTransferEventsBySide(ctx, userWallet, true, beforeEventID, limit)

	case pb.TransferQueryType_TO_WALLET:
		return s.queryTransferEventsBySide(ctx, userWallet, false, beforeEventID, limit)

	case pb.TransferQueryType_ALL:
		var wg sync.WaitGroup
		wg.Add(2)

		var fromResp, toResp *pb.EventResp
		var fromErr, toErr error

		go func() {
			defer wg.Done()
			fromResp, fromErr = s.queryTransferEventsBySide(ctx, userWallet, true, beforeEventID, limit)
		}()

		go func() {
			defer wg.Done()
			toResp, toErr = s.queryTransferEventsBySide(ctx, userWallet, false, beforeEventID, limit)
		}()

		wg.Wait()

		if fromErr != nil {
			return nil, fromErr
		}
		if toErr != nil {
			return nil, toErr
		}
		return mergeTransferEventResponses(fromResp, toResp, limit), nil

	default:
		return nil, status.Errorf(codes.Internal, "[%d] invalid query_type %d", TransferErrCodeInvalidArg, req.QueryType)
	}
}

// queryTransferEventsBySide 根据是否查询 from_wallet 或 to_wallet 方向查询转账事件
func (s *QueryChainEventService) queryTransferEventsBySide(
	ctx context.Context,
	userWallet string,
	isFromWallet bool,
	beforeEventID *uint64,
	limit int,
) (resp *pb.EventResp, err error) {
	fieldName := "to_wallet"
	if isFromWallet {
		fieldName = "from_wallet"
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in queryTransferEventsBySide (field=%s): %v\n%s", fieldName, r, debug.Stack())
			err = status.Errorf(codes.Internal, "[%d] panic occurred: %v", TransferErrCodePanic, r)
		}
	}()

	var query strings.Builder
	var params []any

	query.WriteString(`
		SELECT event_id, from_wallet, to_wallet, token, amount, 
		       tx_hash, signer, block_time, create_at
		FROM transfer_event
		WHERE ` + fieldName + ` = ?
	`)
	params = append(params, userWallet)

	if beforeEventID != nil {
		query.WriteString(" AND event_id < ?")
		params = append(params, *beforeEventID)
	}

	query.WriteString(" ORDER BY event_id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT %d", limit))

	rows, err := s.DB.QueryContext(ctx, query.String(), params...)
	if err != nil {
		logger.Errorf("queryTransferEventsBySide query failed (field=%s): %v", fieldName, err)
		return nil, status.Errorf(codes.Internal, "[%d] query failed: %v", TransferErrCodeQueryFailed, err)
	}
	defer rows.Close()

	var events []*pb.ChainEvent
	for rows.Next() {
		ev := &pb.ChainEvent{}
		var tokenAmount string

		if err := rows.Scan(
			&ev.EventId,
			&ev.UserWallet,
			&ev.ToWallet,
			&ev.Token,
			&tokenAmount,
			&ev.TxHash,
			&ev.Signer,
			&ev.BlockTime,
			&ev.CreateAt,
		); err != nil {
			logger.Errorf("queryTransferEventsBySide scan failed (field=%s): %v", fieldName, err)
			return nil, status.Errorf(codes.Internal, "[%d] failed to parse data: %v", TransferErrCodeScanFailed, err)
		}

		if ev.Signer == "" {
			ev.Signer = ev.UserWallet
		}
		ev.EventIdHash = uint32(utils.EventIdHash(ev.EventId))
		ev.EventType = uint32(pb.EventType_TRANSFER)
		ev.TokenAmount = utils.ParseUint64(tokenAmount)
		ev.Token = utils.DecodeTokenAddress(ev.Token)

		events = append(events, ev)
	}

	if err := rows.Err(); err != nil {
		logger.Errorf("queryTransferEventsBySide rows iteration error (field=%s): %v", fieldName, err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error: %v", TransferErrCodeRowsIter, err)
	}

	return &pb.EventResp{Events: events}, nil
}

// mergeTransferEventResponses 合并来自 from_wallet 和 to_wallet 两个查询结果，并按 event_id 降序排序，截断到 limit 长度
func mergeTransferEventResponses(fromResp, toResp *pb.EventResp, limit int) *pb.EventResp {
	merged := make([]*pb.ChainEvent, 0, limit)
	i, j := 0, 0
	var lastEventId = ^uint64(0) // 用于去重，初始化为最大值

	for len(merged) < limit && (i < len(fromResp.Events) || j < len(toResp.Events)) {
		var ev *pb.ChainEvent
		if i < len(fromResp.Events) &&
			(j >= len(toResp.Events) || fromResp.Events[i].EventId > toResp.Events[j].EventId) {
			ev = fromResp.Events[i]
			i++
		} else if j < len(toResp.Events) {
			ev = toResp.Events[j]
			j++
		}

		if ev == nil {
			break
		}

		// 跳过重复 eventId
		if ev.EventId == lastEventId {
			continue
		}
		lastEventId = ev.EventId

		merged = append(merged, ev)
	}

	return &pb.EventResp{Events: merged}
}
