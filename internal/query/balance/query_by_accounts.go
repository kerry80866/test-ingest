package balance

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
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

	const maxAccounts = 200

	accounts := req.Accounts
	if len(accounts) == 0 {
		return &pb.BalanceListResp{}, nil
	}
	if len(accounts) > maxAccounts {
		return nil, status.Errorf(codes.Internal, "[%d] at most %d accounts are allowed in a single request", ErrCodeParamTooMany, maxAccounts)
	}

	// 构建查询语句
	query := `
		SELECT account_address, owner_address, token_address, balance, last_event_id
		FROM balance
		WHERE account_address IN (`
	placeholders := strings.Repeat("?,", len(accounts))
	query += placeholders[:len(placeholders)-1] + ")"

	args := make([]any, len(accounts))
	for i, addr := range accounts {
		args[i] = addr
	}

	// 执行查询
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Errorf("QueryBalancesByAccounts query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query error", ErrCodeQueryFailed)
	}
	defer rows.Close()

	// 构建 account -> balance 映射
	balanceMap := make(map[string]*pb.Balance, len(accounts))
	for rows.Next() {
		var addr string
		var balance string
		bal := &pb.Balance{}
		if err := rows.Scan(&addr, &bal.OwnerAddress, &bal.TokenAddress, &balance, &bal.LastEventId); err != nil {
			logger.Errorf("QueryBalancesByAccounts row scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] data scan failed", ErrCodeScanFailed)
		}
		bal.Balance = utils.ParseUint64(balance)
		bal.TokenAddress = utils.DecodeTokenAddress(bal.TokenAddress)
		balanceMap[addr] = bal
	}

	// 检查迭代过程中是否有错误
	if err := rows.Err(); err != nil {
		logger.Errorf("QueryBalancesByAccounts rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	// 按请求顺序构建结果
	results := make([]*pb.BalanceResult, 0, len(accounts))
	for _, addr := range accounts {
		br := &pb.BalanceResult{AccountAddress: addr}
		if bal, ok := balanceMap[addr]; ok {
			br.Balance = bal
		}
		results = append(results, br)
	}

	return &pb.BalanceListResp{Results: results}, nil
}
