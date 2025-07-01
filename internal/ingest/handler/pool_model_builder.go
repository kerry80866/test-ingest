package handler

import (
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"github.com/hashicorp/golang-lru"
)

func BuildPoolModels(events *pb.Events, tokenCache *lru.Cache, poolCache *PoolCache) []*model.Pool {
	initialCap := max(10, len(events.Events)/3)
	result := make([]*model.Pool, 0, initialCap)
	for _, e := range events.Events {
		var p *model.Pool
		switch inner := e.Event.(type) {
		case *pb.Event_Trade:
			p = buildPoolModel(inner.Trade, tokenCache, poolCache)
		case *pb.Event_Liquidity:
			p = buildPoolModelFromLiquidity(inner.Liquidity, tokenCache, poolCache)

		default:
			continue
		}
		if p != nil {
			result = append(result, p)
		}
	}
	return result
}

func buildPoolModel(event *pb.TradeEvent, tokenCache *lru.Cache, poolCache *PoolCache) *model.Pool {
	accountKey, exists := poolCache.CheckAndInsert(event.PairAddress, event.TokenAccount, event.QuoteTokenAccount, event.Dex)
	if exists {
		return nil
	}
	return &model.Pool{
		PoolAddress:  utils.EncodeBase58Strict(tokenCache, event.PairAddress),
		AccountKey:   accountKey,
		Dex:          int16(event.Dex),
		TokenAddress: utils.EncodeBase58Strict(tokenCache, event.Token),
		QuoteAddress: utils.EncodeBase58Strict(tokenCache, event.QuoteToken),
		TokenAccount: utils.EncodeBase58Strict(tokenCache, event.TokenAccount),
		QuoteAccount: utils.EncodeBase58Strict(tokenCache, event.QuoteTokenAccount),
		CreateAt:     0,
		UpdateAt:     int32(event.BlockTime),
	}
}

func buildPoolModelFromLiquidity(event *pb.LiquidityEvent, tokenCache *lru.Cache, poolCache *PoolCache) *model.Pool {
	accountKey, exists := poolCache.CheckAndInsert(event.PairAddress, event.TokenAccount, event.QuoteTokenAccount, event.Dex)
	if exists {
		return nil
	}
	return &model.Pool{
		PoolAddress:  utils.EncodeBase58Strict(tokenCache, event.PairAddress),
		AccountKey:   accountKey,
		Dex:          int16(event.Dex),
		TokenAddress: utils.EncodeBase58Strict(tokenCache, event.Token),
		QuoteAddress: utils.EncodeBase58Strict(tokenCache, event.QuoteToken),
		TokenAccount: utils.EncodeBase58Strict(tokenCache, event.TokenAccount),
		QuoteAccount: utils.EncodeBase58Strict(tokenCache, event.QuoteTokenAccount),
		CreateAt:     ifThen(event.Type == pb.EventType_CREATE_POOL, int32(event.BlockTime)),
		UpdateAt:     int32(event.BlockTime),
	}
}

func ifThen(cond bool, val int32) int32 {
	if cond {
		return val
	}
	return 0
}
