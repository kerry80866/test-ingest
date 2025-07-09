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
	"sort"
	"strings"
	"time"
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

	var (
		holderList []*pb.Holder
		localErr   error
	)

	key := fmt.Sprintf("%s:%d", encoded, fetchLimit)
	topHoldersByTokenCache.Do(key, func(e *db.Entry) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in topHoldersByTokenCache func: %+v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
			}
		}()

		if !e.IsExpired() {
			if cached, ok := e.Result.([]*pb.Holder); ok {
				holderList = cached
				return
			}
		}

		rows, queryErr := s.DB.QueryContext(ctx, query, encoded, fetchLimit)
		if queryErr != nil {
			logger.Errorf("QueryTopHoldersByToken query failed: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
			return
		}
		defer rows.Close()

		// 多个账户地址可能属于同一个 owner，我们需要合并这些账户的余额，得到每个 owner 的总余额
		holderMap := make(map[string]*pb.Holder)
		for rows.Next() {
			var owner string
			var balance string

			if queryErr = rows.Scan(&owner, &balance); queryErr != nil {
				logger.Errorf("QueryTopHoldersByToken row scan failed: %v", queryErr)
				localErr = status.Errorf(codes.Internal, "[%d] failed to parse row", ErrCodeScanFailed)
				return
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

		if queryErr = rows.Err(); queryErr != nil {
			logger.Errorf("QueryTopHoldersByToken rows iteration error: %v", queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
			return
		}

		holderList = make([]*pb.Holder, 0, len(holderMap))
		for _, h := range holderMap {
			holderList = append(holderList, h)
		}

		sort.Slice(holderList, func(i, j int) bool {
			return holderList[i].Balance > holderList[j].Balance
		})

		e.Result = holderList
		e.SetValidAt(time.Now().Add(topHoldersByTokenTTL))
	})

	if localErr != nil {
		return nil, localErr
	}

	// 截断前 limit 个
	if len(holderList) > limit {
		holderList = holderList[:limit]
	}
	return &pb.HolderListResp{Holders: holderList}, nil
}
