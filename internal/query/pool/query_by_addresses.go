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

func (s *QueryPoolService) QueryPoolsByAddresses(ctx context.Context, req *pb.PoolAddressesReq) (resp *pb.PoolListResp, err error) {
	const (
		ErrCodeBase         = 60800
		ErrCodePanic        = ErrCodeBase + 32
		ErrCodeParamTooMany = ErrCodeBase + 1
		ErrCodeQueryFailed  = ErrCodeBase + 2
		ErrCodeScanFailed   = ErrCodeBase + 3
		ErrCodeRowsIter     = ErrCodeBase + 4
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryPoolsByAddresses: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	const maxAddresses = 500

	addresses := req.PoolAddresses
	if len(addresses) == 0 {
		return &pb.PoolListResp{}, nil
	}
	if len(addresses) > maxAddresses {
		return nil, status.Errorf(codes.Internal, "[%d] at most %d pool addresses are allowed in a single request", ErrCodeParamTooMany, maxAddresses)
	}

	// 构建查询语句
	query := `
		SELECT pool_address, dex, token_address, quote_address, token_account, quote_account, create_at, update_at
		FROM pool
		WHERE pool_address IN (`
	placeholders := strings.Repeat("?,", len(addresses))
	query += placeholders[:len(placeholders)-1] + ")"

	args := make([]any, len(addresses))
	for i, addr := range addresses {
		args[i] = addr
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Errorf("QueryPoolsByAddresses query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
	}
	defer rows.Close()

	// 构建 pool_address -> []*Pool 映射
	poolMap := make(map[string][]*pb.Pool, len(addresses))
	for rows.Next() {
		p := &pb.Pool{}
		if err := rows.Scan(
			&p.PoolAddress, &p.Dex, &p.TokenAddress, &p.QuoteAddress,
			&p.TokenAccount, &p.QuoteAccount, &p.CreateAt, &p.UpdateAt,
		); err != nil {
			logger.Errorf("QueryPoolsByAddresses row scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] data scan failed", ErrCodeScanFailed)
		}
		p.TokenAddress = utils.DecodeTokenAddress(p.TokenAddress)
		p.QuoteAddress = utils.DecodeTokenAddress(p.QuoteAddress)
		poolMap[p.PoolAddress] = append(poolMap[p.PoolAddress], p)
	}

	if err := rows.Err(); err != nil {
		logger.Errorf("QueryPoolsByAddresses rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	// 构建结果
	results := make([]*pb.PoolResult, 0, len(addresses))
	for _, addr := range addresses {
		pools := poolMap[addr]
		if pools == nil {
			pools = make([]*pb.Pool, 0) // 👈 显式设为空数组，避免 nil
		}
		pr := &pb.PoolResult{
			PoolAddress: addr,
			Pools:       pools,
		}
		results = append(results, pr)
	}

	return &pb.PoolListResp{Results: results}, nil
}
