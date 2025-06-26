package pool

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
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

	var (
		query  strings.Builder
		params []any
	)

	query.WriteString(`
		SELECT pool_address, dex, token_address, quote_address,
		       token_account, quote_account, create_at, update_at
		FROM pool
		WHERE token_address = ?
	`)
	params = append(params, utils.EncodeTokenAddress(base))

	if req.QuoteToken != nil {
		quoteToken := strings.TrimSpace(*req.QuoteToken)
		if quoteToken != "" {
			quote := utils.EncodeTokenAddress(quoteToken)
			query.WriteString(" AND quote_address = ?")
			params = append(params, quote)
		}
	}

	query.WriteString(" ORDER BY pool_address")

	rows, err := s.DB.QueryContext(ctx, query.String(), params...)
	if err != nil {
		logger.Errorf("QueryPoolsByToken query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
	}
	defer rows.Close()

	pools := make([]*pb.Pool, 0, 10)
	for rows.Next() {
		p := &pb.Pool{}
		if err := rows.Scan(
			&p.PoolAddress, &p.Dex, &p.TokenAddress, &p.QuoteAddress,
			&p.TokenAccount, &p.QuoteAccount, &p.CreateAt, &p.UpdateAt,
		); err != nil {
			logger.Errorf("QueryPoolsByToken row scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] failed to parse pool data", ErrCodeScanFailed)
		}
		// decode 转回原始地址
		p.TokenAddress = utils.DecodeTokenAddress(p.TokenAddress)
		p.QuoteAddress = utils.DecodeTokenAddress(p.QuoteAddress)

		pools = append(pools, p)
	}

	if err := rows.Err(); err != nil {
		logger.Errorf("QueryPoolsByToken rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	return &pb.PoolResp{Pools: pools}, nil
}
