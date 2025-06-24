package handler

import (
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"github.com/hashicorp/golang-lru"
)

func BuildBalanceModels(events *pb.Events, cache *lru.Cache) []*model.Balance {
	var result = make([]*model.Balance, 0, len(events.Events))
	for _, e := range events.Events {
		event, ok := e.Event.(*pb.Event_Balance)
		if !ok || event.Balance == nil {
			continue
		}
		result = append(result, buildBalanceModel(event.Balance, cache))
	}
	return result
}

func buildBalanceModel(event *pb.BalanceUpdateEvent, cache *lru.Cache) *model.Balance {
	return &model.Balance{
		AccountAddress: utils.EncodeBase58Strict(cache, event.Account),
		OwnerAddress:   utils.EncodeBase58Strict(cache, event.Owner),
		TokenAddress:   utils.EncodeBase58Strict(cache, event.Token),
		Balance:        utils.Uint64ToDecimal(event.PostBalance),
		LastEventID:    int64(event.EventId),
	}
}
