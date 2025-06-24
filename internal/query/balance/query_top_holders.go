package balance

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sort"
	"strings"
)

func (s *QueryBalanceService) QueryTopHoldersByToken(ctx context.Context, req *pb.TokenTopReq) (resp *pb.HolderListResp, err error) {
	const (
		ErrCodeBase        = 60400
		ErrCodePanic       = ErrCodeBase + 32
		ErrCodeInvalidArg  = ErrCodeBase + 1
		ErrCodeQueryFailed = ErrCodeBase + 2
		ErrCodeScanFailed  = ErrCodeBase + 3
		ErrCodeRowsIter    = ErrCodeBase + 4
	)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryTopHoldersByToken: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	const (
		defaultLimit     = 100
		maxLimit         = 1000
		extraFetchFactor = 1.1 // 多查 10%，便于合并后再截断
	)

	token := strings.TrimSpace(req.TokenAddress)
	if token == "" {
		return nil, status.Errorf(codes.Internal, "[%d] token_address is required", ErrCodeInvalidArg)
	}
	encoded := utils.EncodeTokenAddress(token)

	limit := defaultLimit
	if req.Limit != nil && *req.Limit > 0 {
		limit = int(*req.Limit)
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	fetchLimit := int(float64(limit) * extraFetchFactor)

	query := `
		SELECT owner_address, balance
		FROM balance
		WHERE token_address = ?
		ORDER BY balance DESC
		LIMIT ?`

	rows, err := s.DB.QueryContext(ctx, query, encoded, fetchLimit)
	if err != nil {
		logger.Errorf("QueryTopHoldersByToken query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
	}
	defer rows.Close()

	holderMap := make(map[string]*pb.Holder)
	for rows.Next() {
		var owner string
		var balance string

		if err := rows.Scan(&owner, &balance); err != nil {
			logger.Errorf("QueryTopHoldersByToken row scan failed: %v", err)
			return nil, status.Errorf(codes.Internal, "[%d] failed to parse row", ErrCodeScanFailed)
		}

		bal := utils.ParseUint64(balance)
		if h, ok := holderMap[owner]; ok {
			h.Balance += bal
		} else {
			holderMap[owner] = &pb.Holder{
				OwnerAddress: owner,
				Balance:      bal,
			}
		}
	}

	if err := rows.Err(); err != nil {
		logger.Errorf("QueryTopHoldersByToken rows iteration error: %v", err)
		return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
	}

	// 转 slice 并排序
	holderList := make([]*pb.Holder, 0, len(holderMap))
	for _, h := range holderMap {
		holderList = append(holderList, h)
	}

	sort.Slice(holderList, func(i, j int) bool {
		return holderList[i].Balance > holderList[j].Balance
	})

	// 截断前 limit 个
	if len(holderList) > limit {
		holderList = holderList[:limit]
	}

	return &pb.HolderListResp{Holders: holderList}, nil
}
