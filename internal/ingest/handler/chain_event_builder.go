package handler

import (
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"github.com/hashicorp/golang-lru"
)

// BuildChainEventModels 从 pb.Events 中提取标准 ChainEvent 模型（支持 Trade / Liquidity / Mint / Burn）
func BuildChainEventModels(events *pb.Events, cache *lru.Cache) []*model.ChainEvent {
	var result = make([]*model.ChainEvent, 0, len(events.Events))
	for _, e := range events.Events {
		var ev *model.ChainEvent
		switch inner := e.Event.(type) {
		case *pb.Event_Trade:
			ev = buildTradeEvent(inner.Trade, cache)
		case *pb.Event_Liquidity:
			ev = buildLiquidityEvent(inner.Liquidity, cache)
		case *pb.Event_Mint:
			ev = buildMintEvent(inner.Mint, cache)
		case *pb.Event_Burn:
			ev = buildBurnEvent(inner.Burn, cache)
		default:
			continue
		}
		if ev != nil {
			result = append(result, ev)
		}
	}
	return result
}

func buildTradeEvent(event *pb.TradeEvent, cache *lru.Cache) *model.ChainEvent {
	return &model.ChainEvent{
		EventIDHash: utils.EventIdHash(event.EventId),
		EventID:     int64(event.EventId),
		EventType:   int16(event.Type),
		Dex:         int16(event.Dex),

		UserWallet: utils.EncodeBase58Strict(cache, event.UserWallet),
		ToWallet:   "",

		PoolAddress: utils.EncodeBase58Strict(cache, event.PairAddress),
		Token:       utils.EncodeBase58Strict(cache, event.Token),
		QuoteToken:  utils.EncodeBase58Strict(cache, event.QuoteToken),

		TokenAmount: utils.Uint64ToString(event.TokenAmount),
		QuoteAmount: utils.Uint64ToString(event.QuoteTokenAmount),
		VolumeUsd:   event.AmountUsd,
		PriceUsd:    event.PriceUsd,

		TxHash: utils.TxHashToString(event.TxHash),
		Signer: utils.EncodeBase58Optional(cache, utils.SelectSigner(event.Signers, event.UserWallet)),

		BlockTime: int32(event.BlockTime),
		CreateAt:  0,
	}
}

func buildLiquidityEvent(event *pb.LiquidityEvent, cache *lru.Cache) *model.ChainEvent {
	return &model.ChainEvent{
		EventIDHash: utils.EventIdHash(event.EventId),
		EventID:     int64(event.EventId),
		EventType:   int16(event.Type),
		Dex:         int16(event.Dex),

		UserWallet: utils.EncodeBase58Strict(cache, event.UserWallet),
		ToWallet:   "",

		PoolAddress: utils.EncodeBase58Strict(cache, event.PairAddress),
		Token:       utils.EncodeBase58Strict(cache, event.Token),
		QuoteToken:  utils.EncodeBase58Strict(cache, event.QuoteToken),

		TokenAmount: utils.Uint64ToString(event.TokenAmount),
		QuoteAmount: utils.Uint64ToString(event.QuoteTokenAmount),
		VolumeUsd:   0,
		PriceUsd:    0,

		TxHash: utils.TxHashToString(event.TxHash),
		Signer: utils.EncodeBase58Optional(cache, utils.SelectSigner(event.Signers, event.UserWallet)),

		BlockTime: int32(event.BlockTime),
		CreateAt:  0,
	}
}

func buildMintEvent(event *pb.MintToEvent, cache *lru.Cache) *model.ChainEvent {
	return &model.ChainEvent{
		EventIDHash: utils.EventIdHash(event.EventId),
		EventID:     int64(event.EventId),
		EventType:   int16(event.Type),
		Dex:         int16(0),

		UserWallet: utils.EncodeBase58Strict(cache, event.ToAddress),
		ToWallet:   "",

		PoolAddress: "",
		Token:       utils.EncodeBase58Strict(cache, event.Token),
		QuoteToken:  "",

		TokenAmount: utils.Uint64ToString(event.Amount),
		QuoteAmount: "0",
		VolumeUsd:   0,
		PriceUsd:    0,

		TxHash: utils.TxHashToString(event.TxHash),
		Signer: utils.EncodeBase58Optional(cache, utils.SelectSigner(event.Signers, event.ToAddress)),

		BlockTime: int32(event.BlockTime),
		CreateAt:  0,
	}
}

func buildBurnEvent(event *pb.BurnEvent, cache *lru.Cache) *model.ChainEvent {
	return &model.ChainEvent{
		EventIDHash: utils.EventIdHash(event.EventId),
		EventID:     int64(event.EventId),
		EventType:   int16(event.Type),
		Dex:         int16(0),

		UserWallet: utils.EncodeBase58Strict(cache, event.FromAddress),
		ToWallet:   "",

		PoolAddress: "",
		Token:       utils.EncodeBase58Strict(cache, event.Token),
		QuoteToken:  "",

		TokenAmount: utils.Uint64ToString(event.Amount),
		QuoteAmount: "0",
		VolumeUsd:   0,
		PriceUsd:    0,

		TxHash: utils.TxHashToString(event.TxHash),
		Signer: utils.EncodeBase58Optional(cache, utils.SelectSigner(event.Signers, event.FromAddress)),

		BlockTime: int32(event.BlockTime),
		CreateAt:  0,
	}
}
