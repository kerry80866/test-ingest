package handler

import (
	"dex-ingest-sol/internal/ingest/model"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"github.com/hashicorp/golang-lru"
)

func BuildTransferEventModels(events *pb.Events, cache *lru.Cache) []*model.TransferEvent {
	var result = make([]*model.TransferEvent, 0, len(events.Events))

	for _, e := range events.Events {
		event, ok := e.Event.(*pb.Event_Transfer)
		if !ok || event.Transfer == nil {
			continue
		}
		result = append(result, buildTransferEvent(event.Transfer, cache))
	}
	return result
}

func buildTransferEvent(event *pb.TransferEvent, cache *lru.Cache) *model.TransferEvent {
	return &model.TransferEvent{
		EventIDHash: utils.EventIdHash(event.EventId),
		EventID:     int64(event.EventId),

		FromWallet: utils.EncodeBase58Strict(cache, event.SrcWallet),
		ToWallet:   utils.EncodeBase58Strict(cache, event.DestWallet),

		Token:    utils.EncodeBase58Strict(cache, event.Token),
		Amount:   utils.Uint64ToString(event.Amount),
		Decimals: int16(event.Decimals),

		TxHash: utils.TxHashToString(event.TxHash),
		Signer: utils.EncodeBase58Optional(cache, utils.SelectSigner(event.Signers, event.SrcWallet)),

		BlockTime: int32(event.BlockTime),
		CreateAt:  0,
	}
}
