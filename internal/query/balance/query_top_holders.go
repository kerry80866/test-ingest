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

func (s *QueryBalanceService) QueryTopHoldersByToken(ctx context.Context, req *pb.TokenTopReq) (_ *pb.HolderListResp, err error) {
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
		extraFetchFactor = 1.1 // 多查 10%，用于防止多个 account 属于同一个 owner 时影响前 N 名准确性
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

	key := fmt.Sprintf("%s:%d", encoded, fetchLimit)
	resp, localErr := topHoldersByTokenCache.Do(key, false, func(e *db.Entry, onlyReady bool) (resp any, localErr error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in topHoldersByTokenCache func: %+v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
				resp = nil
			}
		}()

		if !e.IsExpired() {
			if cached, success := e.Result.([]*pb.Holder); success {
				return &pb.HolderListResp{Holders: cached}, nil
			}
		}
		if onlyReady {
			return nil, status.Errorf(codes.NotFound, "cache not ready")
		}

		rows, queryErr := s.DB.QueryContext(ctx, query, encoded, fetchLimit)
		if queryErr != nil {
			logger.Errorf("QueryTopHoldersByToken query failed: %v", queryErr)
			return nil, status.Errorf(codes.Internal, "[%d] query failed", ErrCodeQueryFailed)
		}
		defer rows.Close()

		// 多个账户地址可能属于同一个 owner，我们需要合并这些账户的余额，得到每个 owner 的总余额
		holderMap := make(map[string]*pb.Holder)
		for rows.Next() {
			var owner string
			var balance string

			if queryErr = rows.Scan(&owner, &balance); queryErr != nil {
				logger.Errorf("QueryTopHoldersByToken row scan failed: %v", queryErr)
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

		if queryErr = rows.Err(); queryErr != nil {
			logger.Errorf("QueryTopHoldersByToken rows iteration error: %v", queryErr)
			return nil, status.Errorf(codes.Internal, "[%d] rows iteration error", ErrCodeRowsIter)
		}

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

		e.Result = holderList
		e.SetValidAt(time.Now().Add(topHoldersByTokenTTL))
		return &pb.HolderListResp{Holders: holderList}, nil
	})

	if r, ok := resp.(*pb.HolderListResp); ok {
		return r, nil
	}
	return nil, localErr
}
