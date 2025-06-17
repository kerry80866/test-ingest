package handler

import (
	"dex-ingest-sol/internal/model"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"github.com/hashicorp/golang-lru"
)

func BuildTokenModels(events *pb.Events, cache *lru.Cache) []*model.Token {
	const (
		maxNameLen   = 128
		maxSymbolLen = 64
		maxUriLen    = 256
	)

	truncate := func(fieldName, val string, max int, tokenAddr string) string {
		if len(val) > max {
			logger.Warnf("token [%s] field [%s] too long, truncated: len=%d, val=%q", tokenAddr, fieldName, len(val), val)
			return val[:max]
		}
		return val
	}

	initialCap := utils.Max(10, len(events.Events)/3)
	result := make([]*model.Token, 0, initialCap)

	for _, e := range events.Events {
		switch ev := e.Event.(type) {
		case *pb.Event_Liquidity:
			// 来自流动性事件，仅填 token address + decimals，用于补全缺失信息
			if ev.Liquidity == nil || len(ev.Liquidity.Token) == 0 {
				continue
			}
			result = append(result, &model.Token{
				TokenAddress: utils.EncodeBase58Strict(cache, ev.Liquidity.Token),
				Decimals:     int16(ev.Liquidity.TokenDecimals),
				Source:       int16(0),
				TotalSupply:  "0",
				Name:         "",
				Symbol:       "",
				URI:          "",
				Creator:      "",
				CreateAt:     0,
				IsCreating:   false,
			})

		case *pb.Event_Token:
			// 来自 LaunchpadTokenEvent，填充全部字段
			t := ev.Token
			if t == nil || len(t.Token) == 0 {
				continue
			}

			tokenStr := utils.EncodeBase58Strict(cache, t.Token)
			creator := ""
			if len(t.Creator) > 0 {
				creator = utils.EncodeBase58Strict(cache, t.Creator)
			}

			result = append(result, &model.Token{
				TokenAddress: tokenStr,
				Decimals:     int16(t.Decimals),
				Source:       int16(t.Dex),
				TotalSupply:  utils.Uint64ToDecimal(t.TotalSupply),
				Name:         truncate("name", t.Name, maxNameLen, tokenStr),
				Symbol:       truncate("symbol", t.Symbol, maxSymbolLen, tokenStr),
				URI:          truncate("uri", t.Uri, maxUriLen, tokenStr),
				Creator:      creator,
				CreateAt:     int32(t.BlockTime),
				IsCreating:   true,
				// UpdateAt: 由插入逻辑填充
			})
		}
	}

	return result
}
