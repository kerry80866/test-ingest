package pool

import (
	"context"
	"dex-ingest-sol/internal/pkg/db"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

func (s *QueryPoolService) QueryPoolsByToken(ctx context.Context, req *pb.PoolTokenReq) (resp *pb.PoolResp, err error) {
	const (
		ErrCodeBase        = 60900
		ErrCodePanic       = ErrCodeBase + 32
		ErrCodeInvalidArg  = ErrCodeBase + 1
		ErrCodeQueryFailed = ErrCodeBase + 2
		ErrCodeScanFailed  = ErrCodeBase + 3
		ErrCodeRowsIter    = ErrCodeBase + 4
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryPoolsByToken: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	base := strings.TrimSpace(req.BaseToken)
	if base == "" {
		return nil, status.Errorf(codes.Internal, "[%d] base_token is required", ErrCodeInvalidArg)
	}

	encodedBase := utils.EncodeTokenAddress(base)
	key := encodedBase

	var (
		query  strings.Builder
		params []any
	)

	query.WriteString(`
		SELECT pool_address, dex, token_address, quote_address,
		       token_account, quote_account, create_at, update_at
		FROM pool
		WHERE token_address = ?`)
	params = append(params, encodedBase)

	if req.QuoteToken != nil {
		quoteToken := strings.TrimSpace(*req.QuoteToken)
		if quoteToken != "" {
			encodedQuote := utils.EncodeTokenAddress(quoteToken)
			query.WriteString(" AND quote_address = ?")
			params = append(params, encodedQuote)
			key = key + ":" + encodedQuote
		}
	}

	query.WriteString(" ORDER BY pool_address")

	// ========== 尝试走缓存 ==========
	var (
		pools    []*pb.Pool
		localErr error
	)

	poolsByTokenCache.Do(key, func(e *db.Entry) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in QueryPoolsByToken cache func: %v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
			}
		}()

		if !e.IsExpired() {
			if cached, ok := e.Result.([]*pb.Pool); ok {
				pools = cached
				return
			}
		}

		rows, queryErr := s.DB.QueryContext(ctx, query.String(), params...)
		if queryErr != nil {
			logger.Errorf("QueryPoolsByToken query failed: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
			return
		}
		defer rows.Close()

		pools = make([]*pb.Pool, 0, 10)
		for rows.Next() {
			p := &pb.Pool{}
			var createAt *int32
			if queryErr = rows.Scan(
				&p.PoolAddress, &p.Dex, &p.TokenAddress, &p.QuoteAddress,
				&p.TokenAccount, &p.QuoteAccount, &createAt, &p.UpdateAt,
			); queryErr != nil {
				logger.Errorf("QueryPoolsByToken row scan failed: %v", queryErr)
				localErr = status.Errorf(codes.Internal, "[%d] failed to parse pool data", ErrCodeScanFailed)
				return
			}
			if createAt != nil {
				p.CreateAt = uint32(*createAt)
			}
			p.TokenAddress = utils.DecodeTokenAddress(p.TokenAddress)
			p.QuoteAddress = utils.DecodeTokenAddress(p.QuoteAddress)
			pools = append(pools, p)
		}

		if queryErr = rows.Err(); queryErr != nil {
			logger.Errorf("QueryPoolsByToken rows iteration error: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
			return
		}

		e.Result = pools
		if len(pools) == 0 {
			e.SetValidAt(time.Now().Add(poolsByTokenEmptyTTL)) // 空结果 TTL
		} else {
			e.SetValidAt(time.Now().Add(poolsByTokenTTL)) // 有效结果 TTL
		}
	})

	if localErr != nil {
		return nil, localErr
	}
	return &pb.PoolResp{Pools: pools}, nil
}
