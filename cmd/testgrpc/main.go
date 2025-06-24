package main

import (
	"context"
	"dex-ingest-sol/pb"
	"log"
	"time"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewIngestQueryServiceClient(conn)

	testQueryHolderCountByToken(client)
	testQueryTopHoldersByToken(client)
	testQueryBalancesByOwner(client)
	testQueryBalancesByAccounts(client)
	testQueryEventsByIDs(client)
	testQueryEventsByUser(client)
	testQueryEventsByPool(client)
	testQueryPoolsByAddresses(client)
	testQueryPoolsByToken(client)
}

func testQueryHolderCountByToken(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryHolderCountByToken(ctx, &pb.TokenReq{
		TokenAddress: testToken,
	})
	if err != nil {
		log.Printf("QueryHolderCountByToken error: %v", err)
		return
	}
	log.Printf("HolderCountByToken result: %d", resp.Count)
}

func testQueryTopHoldersByToken(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryTopHoldersByToken(ctx, &pb.TokenTopReq{
		TokenAddress: testToken,
		Limit:        &limit,
	})
	if err != nil {
		log.Printf("QueryTopHoldersByToken error: %v", err)
		return
	}
	log.Printf("TopHoldersByToken result:")
	for i, holder := range resp.Holders {
		log.Printf("%d: owner=%s, balance=%d", i+1, holder.OwnerAddress, holder.Balance)
	}
}

func testQueryBalancesByOwner(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryBalancesByOwner(ctx, &pb.OwnerReq{
		OwnerAddress: testOwner,
	})
	if err != nil {
		log.Printf("QueryBalancesByOwner error: %v", err)
		return
	}
	log.Printf("BalancesByOwner result:")
	for _, bal := range resp.Balances {
		log.Printf("account=%s token=%s balance=%d", bal.AccountAddress, bal.TokenAddress, bal.Balance)
	}
}

func testQueryBalancesByAccounts(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accounts := tokenAccounts // 替换成真实账户地址
	resp, err := client.QueryBalancesByAccounts(ctx, &pb.AccountsReq{
		Accounts: accounts,
	})
	if err != nil {
		log.Printf("QueryBalancesByAccounts error: %v", err)
		return
	}
	log.Printf("BalancesByAccounts result:")
	for _, res := range resp.Results {
		if res.Balance != nil {
			log.Printf("account=%s token=%s balance=%d", res.AccountAddress, res.Balance.TokenAddress, res.Balance.Balance)
		} else {
			log.Printf("account=%s balance=nil", res.AccountAddress)
		}
	}
}

func testQueryEventsByIDs(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryEventsByIDs(ctx, &pb.EventIDsReq{
		EventIds: eventIds,
	})
	if err != nil {
		log.Printf("QueryEventsByIDs error: %v", err)
		return
	}
	log.Printf("EventsByIDs result:")
	for _, res := range resp.Results {
		if res.Event != nil {
			log.Printf("event_id=%d event_type=%d", res.EventId, res.Event.EventType)
		} else {
			log.Printf("event_id=%d event=nil", res.EventId)
		}
	}
}

func testQueryEventsByUser(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryEventsByUser(ctx, &pb.UserEventReq{
		UserWallet: testUser,
		Limit:      &limit,
	})
	if err != nil {
		log.Printf("QueryEventsByUser error: %v", err)
		return
	}
	log.Printf("EventsByUser result:")
	for _, ev := range resp.Events {
		log.Printf("event_id=%d event_type=%d pool=%s token=%s", ev.EventId, ev.EventType, ev.PoolAddress, ev.Token)
	}
}

func testQueryEventsByPool(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryEventsByPool(ctx, &pb.PoolEventReq{
		PoolAddress: testPair,
	})
	if err != nil {
		log.Printf("QueryEventsByPool error: %v", err)
		return
	}
	log.Printf("EventsByPool result:")
	for _, ev := range resp.Events {
		log.Printf("event_id=%d event_type=%d pool=%s token=%s", ev.EventId, ev.EventType, ev.PoolAddress, ev.Token)
	}
}

func testQueryPoolsByAddresses(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs := []string{testPair} // 替换成真实池子地址
	resp, err := client.QueryPoolsByAddresses(ctx, &pb.PoolAddressesReq{
		PoolAddresses: addrs,
	})
	if err != nil {
		log.Printf("QueryPoolsByAddresses error: %v", err)
		return
	}
	log.Printf("PoolsByAddresses result:")
	for _, pr := range resp.Results {
		if pr.Pool != nil {
			log.Printf("pool_address=%s token=%s quote=%s", pr.PoolAddress, pr.Pool.TokenAddress, pr.Pool.QuoteAddress)
		} else {
			log.Printf("pool_address=%s pool=nil", pr.PoolAddress)
		}
	}
}

func testQueryPoolsByToken(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryPoolsByToken(ctx, &pb.PoolTokenReq{
		BaseToken: testToken,
	})
	if err != nil {
		log.Printf("QueryPoolsByToken error: %v", err)
		return
	}
	log.Printf("PoolsByToken result:")
	for _, pool := range resp.Pools {
		log.Printf("pool_address=%s quote=%s tokenAccount=%s quoteAccount=%s", pool.PoolAddress, pool.QuoteAddress, pool.TokenAccount, pool.QuoteAccount)
	}
}
