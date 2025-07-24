package chain

import (
	"context"
	"fmt"
	"github.com/blocto/solana-go-sdk/client"
	"time"
)

func QueryTokenDecimalsFromChain(cl *client.Client, tokens []string, tokenMap map[string]int16) {
	for {
		if queryTokenDecimals(cl, tokens, tokenMap) == nil {
			return
		}
		time.Sleep(3 * time.Second)
	}
}

func queryTokenDecimals(cl *client.Client, tokens []string, tokenMap map[string]int16) (e error) {
	defer func() {
		if r := recover(); r != nil {
			e = fmt.Errorf("queryTokenDecimals panic: %v", r)
		}
	}()

	const batchSize = 100

	// 去重 + 过滤缓存
	unique := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if _, ok := tokenMap[t]; !ok {
			unique[t] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return nil
	}

	// 转成切片方便分批
	list := make([]string, 0, len(unique))
	for t := range unique {
		list = append(list, t)
	}

	// 分批调用
	for start := 0; start < len(list); start += batchSize {
		end := start + batchSize
		if end > len(list) {
			end = len(list)
		}
		batch := list[start:end]

		infos, err := cl.GetMultipleAccounts(context.Background(), batch)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		// 你这里补充处理 infos 逻辑，比如解析 decimals 并写入 tokenMap
		for i, info := range infos {
			if len(info.Data) < 45 {
				continue
			}

			decimals := info.Data[44]
			tokenAddr := batch[i]
			tokenMap[tokenAddr] = int16(decimals)
		}
		//time.Sleep(1 * time.Second)
	}
	return nil
}
