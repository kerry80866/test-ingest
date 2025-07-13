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

func (s *QueryBalanceService) QueryHolderCountByToken(ctx context.Context, req *pb.TokenReq) (_ *pb.HolderCountResp, err error) {
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
	// 对于已知常用 Token（如 sol/usdc/usdt），由于持有人过多，统计代价高、实际用处不大，直接跳过
	if utils.IsKnownToken(encoded) {
		return &pb.HolderCountResp{Count: 0}, nil
	}

	resp, localErr := holderCountCache.Do(encoded, false, func(e *db.Entry, onlyReady bool) (resp any, localErr error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in QueryHolderCountByToken cache func: %v", r)
				localErr = status.Errorf(codes.Internal, "[%d] server panic", ErrCodePanic)
				resp = nil
			}
		}()

		if !e.IsExpired() {
			cached, success := e.Result.(int64)
			if success {
				return &pb.HolderCountResp{Count: uint64(cached)}, nil
			}
		}
		if onlyReady {
			return nil, status.Errorf(codes.NotFound, "cache not ready")
		}

		var count int64
		queryErr := s.DB.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT owner_address)
			FROM balance
			WHERE token_address = ?
		`, encoded).Scan(&count)
		if queryErr != nil {
			logger.Errorf("QueryHolderCountByToken failed: token=%s, err=%v", token, queryErr)
			return nil, status.Errorf(codes.Internal, "[%d] query holder count failed", ErrCodeQueryFailed)
		}

		e.Result = count
		e.SetValidAt(time.Now().Add(getHolderCountTTL(count)))
		return &pb.HolderCountResp{Count: uint64(count)}, nil
	})

	if r, ok := resp.(*pb.HolderCountResp); ok {
		return r, nil
	}
	return nil, localErr
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
		return 45 * time.Second
	case count < 50_000:
		return 1 * time.Minute
	default:
		return 5 * time.Minute
	}
}
