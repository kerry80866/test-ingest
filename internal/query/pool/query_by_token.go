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

func (s *QueryPoolService) QueryPoolsByToken(ctx context.Context, req *pb.PoolTokenReq) (_ *pb.PoolResp, err error) {
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
	resp, localErr := poolsByTokenCache.Do(key, false, func(e *db.Entry, onlyReady bool) (resp any, localErr error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in QueryPoolsByToken cache func: %v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
				resp = nil
			}
		}()

		if !e.IsExpired() {
			if cached, ok := e.Result.([]*pb.Pool); ok {
				return &pb.PoolResp{Pools: cached}, nil
			}
		}
		if onlyReady {
			return nil, status.Errorf(codes.NotFound, "cache not ready")
		}

		rows, queryErr := s.DB.QueryContext(ctx, query.String(), params...)
		if queryErr != nil {
			logger.Errorf("QueryPoolsByToken query failed: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
			return nil, localErr
		}
		defer rows.Close()

		pools := make([]*pb.Pool, 0, 10)
		for rows.Next() {
			p := &pb.Pool{}
			var createAt *int32
			if queryErr = rows.Scan(
				&p.PoolAddress, &p.Dex, &p.TokenAddress, &p.QuoteAddress,
				&p.TokenAccount, &p.QuoteAccount, &createAt, &p.UpdateAt,
			); queryErr != nil {
				logger.Errorf("QueryPoolsByToken row scan failed: %v", queryErr)
				localErr = status.Errorf(codes.Internal, "[%d] failed to parse pool data", ErrCodeScanFailed)
				return nil, localErr
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
			return nil, localErr
		}

		e.Result = pools
		if len(pools) == 0 {
			e.SetValidAt(time.Now().Add(poolsByTokenEmptyTTL)) // 空结果 TTL
		} else {
			e.SetValidAt(time.Now().Add(poolsByTokenTTL)) // 有效结果 TTL
		}
		return &pb.PoolResp{Pools: pools}, nil
	})

	if r, ok := resp.(*pb.PoolResp); ok {
		return r, nil
	}
	return nil, localErr
}
