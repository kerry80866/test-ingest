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

	var (
		missingAddrs = make([]string, 0, len(addresses))
		poolMap      = make(map[string][]*pb.Pool, len(addresses))
	)

	// 先从缓存中获取
	for _, addr := range addresses {
		found := false
		poolsByAddressCache.DoRead(addr, func(e *db.Entry) {
			if cached, ok := e.Result.([]*pb.Pool); ok {
				poolMap[addr] = cached
				found = true
				return
			}
		})
		if !found {
			missingAddrs = append(missingAddrs, addr)
		}
	}
	if len(missingAddrs) == 0 {
		return makeResult(poolMap, addresses), nil
	}

	// 构建查询语句
	query := `
		SELECT pool_address, dex, token_address, quote_address, token_account, quote_account, create_at, update_at
		FROM pool
		WHERE pool_address IN (`
	placeholders := strings.Repeat("?,", len(missingAddrs))
	query += placeholders[:len(placeholders)-1] + ")"

	args := make([]any, len(missingAddrs))
	for i, addr := range missingAddrs {
		args[i] = addr
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Errorf("QueryPoolsByAddresses query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
	}
	defer rows.Close()

	// 构建 pool_address -> []*Pool 映射
	for rows.Next() {
		p := &pb.Pool{}
		var createAt *int32
		if err := rows.Scan(
			&p.PoolAddress, &p.Dex, &p.TokenAddress, &p.QuoteAddress,
			&p.TokenAccount, &p.QuoteAccount, &createAt, &p.UpdateAt,
		); err != nil {
			logger.Errorf("QueryPoolsByAddresses row scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] data scan failed", ErrCodeScanFailed)
		}
		if createAt != nil {
			p.CreateAt = uint32(*createAt)
		} else {
			p.CreateAt = 0
		}
		p.TokenAddress = utils.DecodeTokenAddress(p.TokenAddress)
		p.QuoteAddress = utils.DecodeTokenAddress(p.QuoteAddress)
		poolMap[p.PoolAddress] = append(poolMap[p.PoolAddress], p)
	}

	// 此设计允许“部分成功即缓存”，属于预期行为，可避免热点 pool 因边缘失败丢失缓存
	for key, pool := range poolMap {
		setPoolsCache(key, pool)
	}

	if err := rows.Err(); err != nil {
		logger.Errorf("QueryPoolsByAddresses rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	// 空缓存优先级较低，仅在整体处理完成后写入，用于防止缓存穿透
	for _, addr := range missingAddrs {
		if _, ok := poolMap[addr]; !ok {
			setPoolsCache(addr, []*pb.Pool{})
		}
	}
	return makeResult(poolMap, addresses), nil
}

func setPoolsCache(key string, value []*pb.Pool) {
	poolsByAddressCache.Do(key, true, func(e *db.Entry, onlyReady bool) (resp any, localErr error) {
		e.Result = value
		if len(value) == 0 {
			e.SetValidAt(time.Now().Add(poolsByAddressEmptyTTL))
		} else {
			e.SetValidAt(time.Now().Add(poolsByAddressTTL))
		}
		return value, nil
	})
}

func makeResult(poolMap map[string][]*pb.Pool, addresses []string) *pb.PoolListResp {
	// 构建结果
	results := make([]*pb.PoolResult, 0, len(addresses))
	for _, addr := range addresses {
		pools := poolMap[addr]
		if pools == nil {
			pools = make([]*pb.Pool, 0) // 显式设为空数组，避免 nil
		}
		results = append(results, &pb.PoolResult{
			PoolAddress: addr,
			Pools:       pools,
		})
	}
	return &pb.PoolListResp{Results: results}
}
