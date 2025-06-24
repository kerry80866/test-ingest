package balance

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *QueryBalanceService) QueryBalancesByOwner(ctx context.Context, req *pb.OwnerReq) (resp *pb.BalanceResp, err error) {
	const (
		ErrCodeBase        = 60200
		ErrCodePanic       = ErrCodeBase + 32
		ErrCodeInvalidArg  = ErrCodeBase + 1
		ErrCodeQueryFailed = ErrCodeBase + 2
		ErrCodeScanFailed  = ErrCodeBase + 3
		ErrCodeRowsIter    = ErrCodeBase + 4
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryBalancesByOwner: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	const maxResultLimit = 1000 // 防御性限制，避免 owner 拥有太多账户

	owner := req.OwnerAddress
	if owner == "" {
		return nil, status.Errorf(codes.Internal, "[%d] owner_address cannot be empty", ErrCodeInvalidArg)
	}

	var (
		query string
		args  []any
	)

	if req.TokenAddress != nil && *req.TokenAddress != "" {
		tokenID := utils.EncodeTokenAddress(*req.TokenAddress)
		query = `
			SELECT account_address, token_address, balance
			FROM balance
			WHERE owner_address = ? AND token_address = ?
			LIMIT ?`
		args = []any{owner, tokenID, maxResultLimit}
	} else {
		query = `
			SELECT account_address, token_address, balance
			FROM balance
			WHERE owner_address = ?
			LIMIT ?`
		args = []any{owner, maxResultLimit}
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		logger.Errorf("QueryBalancesByOwner query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query error", ErrCodeQueryFailed)
	}
	defer rows.Close()

	balances := make([]*pb.Balance, 0, 10)
	for rows.Next() {
		var balanceStr string
		bal := &pb.Balance{
			OwnerAddress: owner,
			LastEventId:  0, // 数据库没查出，可设为默认值
		}
		if err := rows.Scan(&bal.AccountAddress, &bal.TokenAddress, &balanceStr); err != nil {
			logger.Errorf("QueryBalancesByOwner row scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] data scan failed", ErrCodeScanFailed)
		}
		bal.Balance = utils.ParseUint64(balanceStr)
		bal.TokenAddress = utils.DecodeTokenAddress(bal.TokenAddress)
		balances = append(balances, bal)
	}

	if err := rows.Err(); err != nil {
		logger.Errorf("QueryBalancesByOwner rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	return &pb.BalanceResp{Balances: balances}, nil
}
