package handler

import (
	"dex-ingest-sol/internal/model"
	"dex-ingest-sol/pb"
	"dex-ingest-sol/pkg/utils"
	"github.com/hashicorp/golang-lru"
)

func BuildPoolModels(events *pb.Events, cache *lru.Cache) []*model.Pool {
	initialCap := utils.Max(10, len(events.Events)/3)
	result := make([]*model.Pool, 0, initialCap)
	for _, e := range events.Events {
		event, ok := e.Event.(*pb.Event_Liquidity)
		if !ok || event.Liquidity == nil {
			continue
		}
		result = append(result, buildPoolModel(event.Liquidity, cache))
	}
	return result
}

func buildPoolModel(event *pb.LiquidityEvent, cache *lru.Cache) *model.Pool {
	return &model.Pool{
		PoolAddress:  utils.EncodeBase58Strict(cache, event.PairAddress),
		Dex:          int16(event.Dex),
		TokenAddress: utils.EncodeBase58Strict(cache, event.Token),
		QuoteAddress: utils.EncodeBase58Strict(cache, event.QuoteToken),
		TokenAccount: utils.EncodeBase58Strict(cache, event.TokenAccount),
		QuoteAccount: utils.EncodeBase58Strict(cache, event.QuoteTokenAccount),
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
