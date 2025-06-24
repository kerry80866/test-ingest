package query

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/query/balance"
	"dex-ingest-sol/internal/query/chainevent"
	"dex-ingest-sol/internal/query/pool"
	"dex-ingest-sol/pb"
)

type QueryService struct {
	pb.UnimplementedIngestQueryServiceServer

	balanceService    *balance.QueryBalanceService
	chainEventService *chainevent.QueryChainEventService
	poolService       *pool.QueryPoolService
}

func NewQueryService(db *sql.DB) *QueryService {
	return &QueryService{
		balanceService:    balance.NewQueryBalanceService(db),
		chainEventService: chainevent.NewQueryChainEventService(db),
		poolService:       pool.NewQueryPoolService(db),
	}
}

// Balance 相关
func (s *QueryService) QueryBalancesByAccounts(ctx context.Context, req *pb.AccountsReq) (*pb.BalanceListResp, error) {
	return s.balanceService.QueryBalancesByAccounts(ctx, req)
}

func (s *QueryService) QueryBalancesByOwner(ctx context.Context, req *pb.OwnerReq) (*pb.BalanceResp, error) {
	return s.balanceService.QueryBalancesByOwner(ctx, req)
}

func (s *QueryService) QueryTopHoldersByToken(ctx context.Context, req *pb.TokenTopReq) (*pb.HolderListResp, error) {
	return s.balanceService.QueryTopHoldersByToken(ctx, req)
}

func (s *QueryService) QueryHolderCountByToken(ctx context.Context, req *pb.TokenReq) (*pb.HolderCountResp, error) {
	return s.balanceService.QueryHolderCountByToken(ctx, req)
}

// Event 相关
func (s *QueryService) QueryEventsByIDs(ctx context.Context, req *pb.EventIDsReq) (*pb.EventListResp, error) {
	return s.chainEventService.QueryEventsByIDs(ctx, req)
}

func (s *QueryService) QueryEventsByUser(ctx context.Context, req *pb.UserEventReq) (*pb.EventResp, error) {
	return s.chainEventService.QueryEventsByUser(ctx, req)
}

func (s *QueryService) QueryEventsByPool(ctx context.Context, req *pb.PoolEventReq) (*pb.EventResp, error) {
	return s.chainEventService.QueryEventsByPool(ctx, req)
}

// Pool 相关
func (s *QueryService) QueryPoolsByAddresses(ctx context.Context, req *pb.PoolAddressesReq) (resp *pb.PoolListResp, err error) {
	return s.poolService.QueryPoolsByAddresses(ctx, req)
}

func (s *QueryService) QueryPoolsByToken(ctx context.Context, req *pb.PoolTokenReq) (*pb.PoolResp, error) {
	return s.poolService.QueryPoolsByToken(ctx, req)
}
