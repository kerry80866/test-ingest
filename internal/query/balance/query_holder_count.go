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

	var count int64
	err = s.DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT owner_address)
		FROM balance
		WHERE token_address = ?
	`, encoded).Scan(&count)
	if err != nil {
		logger.Errorf("QueryHolderCountByToken failed: token=%s, err=%v", token, err)
		return nil, status.Errorf(codes.Internal, "[%d] query holder count failed", ErrCodeQueryFailed)
	}

	return &pb.HolderCountResp{Count: uint64(count)}, nil
}
