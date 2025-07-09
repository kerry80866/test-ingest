package balance

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

func (s *QueryBalanceService) QueryBalancesByAccounts(ctx context.Context, req *pb.AccountsReq) (resp *pb.BalanceListResp, err error) {
	const (
		ErrCodeBase         = 60100
		ErrCodePanic        = ErrCodeBase + 32
		ErrCodeParamTooMany = ErrCodeBase + 1
		ErrCodeQueryFailed  = ErrCodeBase + 2
		ErrCodeScanFailed   = ErrCodeBase + 3
		ErrCodeRowsIter     = ErrCodeBase + 4
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryBalancesByAccounts: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	const maxAccounts = 2000

	accounts := req.Accounts
	if len(accounts) == 0 {
		return &pb.BalanceListResp{}, nil
	}
	if len(accounts) > maxAccounts {
		return nil, status.Errorf(codes.Internal, "[%d] at most %d accounts are allowed in a single request", ErrCodeParamTooMany, maxAccounts)
	}

	var (
		missingAddrs = make([]string, 0, len(accounts))
		balanceMap   = make(map[string]*pb.Balance, len(accounts))
	)

	// 先从缓存中尝试获取
	for _, addr := range accounts {
		found := false
		balancesByAccountsCache.Do(addr, func(e *db.Entry) {
			if !e.IsExpired() {
				if cached, ok := e.Result.(*pb.Balance); ok {
					balanceMap[addr] = cached
					found = true
					return
				}
			}
		})
		if !found {
			missingAddrs = append(missingAddrs, addr)
		}
	}
	if len(missingAddrs) == 0 {
		return makeResult(balanceMap, accounts), nil
	}

	// 构建查询语句
	query := `
		SELECT account_address, owner_address, token_address, balance, last_event_id
		FROM balance
		WHERE account_address IN (`
	placeholders := strings.Repeat("?,", len(missingAddrs))
	query += placeholders[:len(placeholders)-1] + ")"

	args := make([]any, len(missingAddrs))
	for i, addr := range missingAddrs {
		args[i] = addr
	}

	// 执行查询
	rows, queryErr := s.DB.QueryContext(ctx, query, args...)
	if queryErr != nil {
		logger.Errorf("QueryBalancesByAccounts query failed: %v", queryErr)
		return nil, status.Errorf(codes.Internal, "[%d] query error", ErrCodeQueryFailed)
	}
	defer rows.Close()

	// 构建 account -> balance 映射
	for rows.Next() {
		var addr string
		var balance string
		bal := &pb.Balance{}
		if queryErr = rows.Scan(&addr, &bal.OwnerAddress, &bal.TokenAddress, &balance, &bal.LastEventId); queryErr != nil {
			logger.Errorf("QueryBalancesByAccounts row scan failed: %v", queryErr)
			return nil, status.Errorf(codes.Internal, "[%d] data scan failed", ErrCodeScanFailed)
		}
		bal.Balance = utils.ParseUint64(balance)
		bal.TokenAddress = utils.DecodeTokenAddress(bal.TokenAddress)
		balanceMap[addr] = bal

		balancesByAccountsCache.Do(addr, func(e *db.Entry) {
			e.Result = bal
			e.SetValidAt(time.Now().Add(balancesByAccountsTTL))
		})
	}

	// 检查迭代过程中是否有错误
	if queryErr = rows.Err(); queryErr != nil {
		logger.Errorf("QueryBalancesByAccounts rows iteration error: %v", queryErr)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	return makeResult(balanceMap, accounts), nil
}

func makeResult(balanceMap map[string]*pb.Balance, accounts []string) *pb.BalanceListResp {
	result := make([]*pb.BalanceResult, 0, len(accounts))
	for _, addr := range accounts {
		result = append(result, &pb.BalanceResult{
			AccountAddress: addr,
			Balance:        balanceMap[addr],
		})
	}
	return &pb.BalanceListResp{Results: result}
}
