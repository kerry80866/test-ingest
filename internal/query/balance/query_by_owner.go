package balance

import (
	"context"
	"dex-ingest-sol/internal/pkg/db"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
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

	tokenAddr := ""
	if req.TokenAddress != nil {
		tokenAddr = strings.TrimSpace(*req.TokenAddress)
	}

	key := owner
	if tokenAddr != "" {
		tokenID := utils.EncodeTokenAddress(tokenAddr)
		query = `
			SELECT account_address, token_address, balance
			FROM balance
			WHERE owner_address = ? AND token_address = ?
			LIMIT ?`
		args = []any{owner, tokenID, maxResultLimit}

		key = fmt.Sprintf("%s:%s", owner, tokenID)
	} else {
		query = `
			SELECT account_address, token_address, balance
			FROM balance
			WHERE owner_address = ?
			LIMIT ?`
		args = []any{owner, maxResultLimit}
	}

	var (
		result   []*pb.Balance
		localErr error
	)

	balancesByOwnerCache.Do(key, func(e *db.Entry, created bool) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in QueryBalancesByOwner: %v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
			}
		}()

		if !created && !e.IsExpired() {
			cached, ok := e.Result.([]*pb.Balance)
			if ok {
				result = cached
				return
			}
		}

		rows, queryErr := s.DB.QueryContext(ctx, query, args...)
		if queryErr != nil {
			logger.Errorf("QueryBalancesByOwner query failed: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] query error", ErrCodeQueryFailed)
			return
		}
		defer rows.Close()

		result = make([]*pb.Balance, 0, 10)
		for rows.Next() {
			var balanceStr string
			bal := &pb.Balance{
				OwnerAddress: owner,
				LastEventId:  0, // 数据库没查出，可设为默认值
			}
			if queryErr = rows.Scan(&bal.AccountAddress, &bal.TokenAddress, &balanceStr); queryErr != nil {
				logger.Errorf("QueryBalancesByOwner row scan failed: %v", queryErr)
				localErr = status.Errorf(codes.Internal, "[%d] data scan failed", ErrCodeScanFailed)
				return
			}
			bal.Balance = utils.ParseUint64(balanceStr)
			bal.TokenAddress = utils.DecodeTokenAddress(bal.TokenAddress)
			result = append(result, bal)
		}

		if queryErr = rows.Err(); queryErr != nil {
			logger.Errorf("QueryBalancesByOwner rows iteration error: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
			return
		}

		e.Result = result
		e.SetValidAt(time.Now().Add(balancesByOwnerTTL))
	})

	if localErr != nil {
		return nil, localErr
	}
	return &pb.BalanceResp{Balances: result}, nil
}
