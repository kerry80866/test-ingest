package main

import (
	"context"
	"dex-ingest-sol/pb"
	"fmt"
	"github.com/zeromicro/go-zero/zrpc"
	"log"
	"time"
)

var (
	limit = uint32(20)

	testToken = "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN"
	testPair  = "9d9mb8kooFfaD3SctgZtkxQypkshx6ezhbKio89ixyy2"
	testUser  = "5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9"

	testOwner = "5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9"

	eventIds = []uint64{
		1490979505759323392,
		1490980613911610624,
		1490978122837067008,
		1490980682643866880,
		1490980682643866881,
	}
	tokenAccounts = []string{
		//"12DPMg8fwA2ssxA1Zvqo4HTibhxUsofszXBnZxjkJHnU",
		//"12j4FzjWc8UiwWAkqzAQDk7Jw57yA6o164r4EKVXvhMa",
		//"5tzFkiKscXHK5ZXCGbXZxdw7gTjjD1mBwuoFbhUvuAi9",
		"YN4U8xySzuyARUMTNCpMgkkek7nnh2VAkUMdMygpump",
	}
)

func main() {
	//conn, err := grpc.Dial("172.19.32.47:50051", grpc.WithInsecure(), grpc.WithBlock())
	//if err != nil {
	//	log.Fatalf("failed to connect to gRPC server: %v", err)
	//}
	//defer conn.Close()
	conn := zrpc.MustNewClient(zrpc.RpcClientConf{Target: "localhost:50051"}).Conn()

	client := pb.NewIngestQueryServiceClient(conn)

	//testQueryHolderCountByToken(client)
	//testQueryTopHoldersByToken(client)
	//testQueryBalancesByOwner(client)
	//testQueryBalancesByAccounts(client)
	//testQueryEventsByIDs(client)
	//testQueryEventsByUser(client)
	//testQueryEventsByPool(client)
	//testQueryPoolsByAddresses(client)
	testQueryPoolsByToken(client)
	testQueryTransferEvents(client)
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
		for _, p := range pr.Pools {
			log.Printf("token=%s quote=%s tokenAccount=%s quoteAccount=%s",
				p.TokenAddress, p.QuoteAddress, p.TokenAccount, p.QuoteAccount)
		}
	}
}

//
//func testQueryPoolsByToken(client pb.IngestQueryServiceClient) {
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	resp, err := client.QueryPoolsByToken(ctx, &pb.PoolTokenReq{
//		BaseToken: testToken,
//	})
//	if err != nil {
//		log.Printf("QueryPoolsByToken error: %v", err)
//		return
//	}
//	log.Printf("PoolsByToken result:")
//	for _, pool := range resp.Pools {
//		log.Printf("pool_address=%s quote=%s tokenAccount=%s quoteAccount=%s", pool.PoolAddress, pool.QuoteAddress, pool.TokenAccount, pool.QuoteAccount)
//	}
//}

func testQueryPoolsByToken(client pb.IngestQueryServiceClient) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.QueryPoolsByToken(ctx, &pb.PoolTokenReq{
		BaseToken: testToken,
	})
	if err != nil {
		log.Printf("QueryPoolsByToken error: %v", err)
		return
	}
	duration := time.Since(start)
	log.Printf("QueryPoolsByToken took: %s", duration)

	// 收集所有tokenAccount和quoteAccount，准备查询余额
	accountSet := make(map[string]struct{})
	for _, pool := range resp.Pools {
		if pool.TokenAccount != "" {
			accountSet[pool.TokenAccount] = struct{}{}
		}
		if pool.QuoteAccount != "" {
			accountSet[pool.QuoteAccount] = struct{}{}
		}
	}

	accounts := make([]string, 0, len(accountSet))
	for acc := range accountSet {
		accounts = append(accounts, acc)
	}

	if len(accounts) == 0 {
		log.Println("No token accounts found for balance query")
		return
	}

	// 调用 QueryBalancesByAccounts 查询余额
	start1 := time.Now()
	balanceResp, err := client.QueryBalancesByAccounts(ctx, &pb.AccountsReq{
		Accounts: accounts,
	})
	duration1 := time.Since(start1)
	log.Printf("QueryBalancesByAccounts took: %s", duration1)
	if err != nil {
		log.Printf("QueryBalancesByAccounts error: %v", err)
		return
	}

	log.Printf("Balances for token and quote accounts:")
	for _, result := range balanceResp.Results {
		balStr := "nil"
		if result.Balance != nil {
			balStr = fmt.Sprintf("%d", result.Balance.Balance)
		}
		log.Printf("account=%s, balance=%s", result.AccountAddress, balStr)
	}
}

func testQueryTransferEvents(client pb.IngestQueryServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.TransferEventQueryReq{
		UserWallet: testUser,                 // 替换为实际地址
		QueryType:  pb.TransferQueryType_ALL, // 支持 QUERY_FROM_WALLET / QUERY_TO_WALLET / QUERY_ALL
	}

	resp, err := client.QueryTransferEvents(ctx, req)
	if err != nil {
		log.Printf("QueryTransferEvents error: %v", err)
		return
	}

	log.Println("TransferEvents result:")
	for _, ev := range resp.Events {
		log.Printf("from=%s to=%s token=%s amount=%s block_time=%d",
			ev.UserWallet, ev.ToWallet, ev.Token, ev.TokenAmount, ev.BlockTime)
	}
}
