package main

import (
	"database/sql"
	"dex-ingest-sol/cmd/rpc/chain"
	"dex-ingest-sol/cmd/rpc/common"
	"dex-ingest-sol/cmd/rpc/db"
	"github.com/blocto/solana-go-sdk/client"
	_ "github.com/go-sql-driver/mysql" // 替换为你用的 driver
	_ "github.com/lib/pq"
	"log"
)

const lindormDSN = ""

const endpoint = ""

var tokenMap = make(map[string]int16)

func main() {
	tokenMap["0"] = 9
	tokenMap["1"] = 9
	tokenMap["2"] = 6
	tokenMap["3"] = 6

	lindorm, err := sql.Open("mysql", lindormDSN)
	if err != nil {
		log.Fatalf("连接lindorm失败: %v", err)
	}
	defer lindorm.Close()

	cl := client.NewClient(endpoint)

	for {
		pools := db.Query(lindorm)
		if len(pools) == 0 {
			log.Printf("查询到空池，结束")
			return
		}

		tokens := getTokens(pools)
		log.Printf("待补全的 token 数量: %d", len(tokens))

		chain.QueryTokenDecimalsFromChain(cl, tokens, tokenMap)
		log.Printf("链上 decimals 查询完成")

		list := make([]*common.LindormResult, 0, len(pools))

		missCount := 0
		for _, pool := range pools {
			baseDecimals, baseOk := tokenMap[pool.TokenAddress]
			quoteDecimals, quoteOk := tokenMap[pool.QuoteAddress]

			if baseOk {
				pool.TokenDecimals = baseDecimals
			}
			if quoteOk {
				pool.QuoteDecimals = quoteDecimals
			}

			if baseOk && quoteOk {
				list = append(list, pool)
			} else {
				missCount++
			}
		}

		log.Printf("可写入 pool 数量: %d，跳过数量: %d", len(list), missCount)

		if len(list) > 0 {
			err := db.InsertPools(lindorm, list)
			if err != nil {
				log.Printf("写入 Lindorm 失败: %v", err)
			} else {
				log.Printf("写入 Lindorm 成功，写入数量: %d", len(list))
			}
		}
	}
}

func getTokens(pools []*common.LindormResult) []string {
	unique := make(map[string]struct{}, len(pools))
	for _, pool := range pools {
		if decimals, ok := tokenMap[pool.TokenAddress]; ok {
			pool.TokenDecimals = decimals
		} else {
			unique[pool.TokenAddress] = struct{}{}
		}
		if decimals, ok := tokenMap[pool.QuoteAddress]; ok {
			pool.QuoteDecimals = decimals
		} else {
			unique[pool.QuoteAddress] = struct{}{}
		}
	}

	list := make([]string, 0, len(unique))
	for t := range unique {
		list = append(list, t)
	}
	return list
}
