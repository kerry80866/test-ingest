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

func (s *QueryBalanceService) QueryHolderCountByToken(ctx context.Context, req *pb.TokenReq) (resp *pb.HolderCountResp, err error) {
	const (
		ErrCodeBase        = 60300
		ErrCodePanic       = ErrCodeBase + 32
		ErrCodeInvalidArg  = ErrCodeBase + 1
		ErrCodeQueryFailed = ErrCodeBase + 2
	)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in QueryHolderCountByToken: %v", r)
			err = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
		}
	}()

	token := strings.TrimSpace(req.TokenAddress)
	if token == "" {
		return nil, status.Errorf(codes.Internal, "[%d] token_address is required", ErrCodeInvalidArg)
	}

	encoded := utils.EncodeTokenAddress(token)
	if utils.IsKnownToken(encoded) {
		return &pb.HolderCountResp{Count: 0}, nil
	}

	var (
		count    int64
		localErr error
	)

	holderCountCache.Do(encoded, func(e *db.Entry) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in QueryHolderCountByToken cache func: %v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
			}
		}()

		if !e.IsExpired() {
			cached, ok := e.Result.(int64)
			if ok {
				count = cached
				return
			}
		}

		queryErr := s.DB.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT owner_address)
			FROM balance
			WHERE token_address = ?
		`, encoded).Scan(&count)
		if queryErr != nil {
			logger.Errorf("QueryHolderCountByToken failed: token=%s, err=%v", token, queryErr)
			localErr = status.Errorf(codes.Internal, "[%d] query holder count failed", ErrCodeQueryFailed)
			return
		}

		e.Result = count
		e.SetValidAt(time.Now().Add(getHolderCountTTL(count)))
	})

	if localErr != nil {
		return nil, localErr
	}
	return &pb.HolderCountResp{Count: uint64(count)}, nil
}

func getHolderCountTTL(count int64) time.Duration {
	switch {
	case count < 500:
		return 5 * time.Second
	case count < 1_000:
		return 10 * time.Second
	case count < 5_000:
		return 15 * time.Second
	case count < 10_000:
		return 30 * time.Second
	case count < 20_000:
		return 1 * time.Minute
	case count < 50_000:
		return 3 * time.Minute
	default:
		return 5 * time.Minute
	}
}
