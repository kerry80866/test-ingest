package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const solanaRPC = "https://api.mainnet-beta.solana.com" // 你也可以换成你自己的 RPC 节点

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("用法: %s <tx_hash> <ix_index> <inner_index>\n", os.Args[0])
		os.Exit(1)
	}

	txHash := os.Args[1]
	ixIndex := mustParseUint8(os.Args[2])
	innerIndex := mustParseUint8(os.Args[3])

	slot, err := getSlotByTxHash(txHash)
	if err != nil {
		log.Fatalf("获取 slot 失败: %v", err)
	}
	fmt.Printf("🔹 slot: %d\n", slot)

	txIndex, err := getTxIndexInBlock(slot, txHash)
	if err != nil {
		log.Fatalf("获取 tx_index 失败: %v", err)
	}
	fmt.Printf("🔹 tx_index: %d\n", txIndex)

	eventID := (slot << 32) | (uint64(txIndex) << 16) | (uint64(ixIndex) << 8) | uint64(innerIndex)
	fmt.Printf("✅ event_id = %d (hex: 0x%x)\n", eventID, eventID)
}

// ===== getTransaction =====

func getSlotByTxHash(txHash string) (uint64, error) {
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{
			txHash,
			map[string]interface{}{
				"encoding": "json",
				//"maxSupportedTransactionVersion": 0,
			},
		},
	}

	respBody, err := doPost(body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Result struct {
			Slot uint64 `json:"slot"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, err
	}
	if result.Error != nil {
		return 0, fmt.Errorf("RPC 错误: %s", result.Error.Message)
	}
	return result.Result.Slot, nil
}

// ===== getBlock =====

func getTxIndexInBlock(slot uint64, txHash string) (int, error) {
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getBlock",
		"params": []interface{}{
			slot,
			map[string]interface{}{
				"encoding":                       "json",
				"transactionDetails":             "full",
				"maxSupportedTransactionVersion": 0, // ✅ 加上这行
			},
		},
	}

	respBody, err := doPost(body)
	if err != nil {
		return -1, err
	}

	var result struct {
		Result struct {
			Transactions []struct {
				Transaction struct {
					Signatures []string `json:"signatures"`
				} `json:"transaction"`
			} `json:"transactions"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return -1, err
	}
	if result.Error != nil {
		return -1, fmt.Errorf("RPC 错误: %s", result.Error.Message)
	}

	for i, tx := range result.Result.Transactions {
		if len(tx.Transaction.Signatures) > 0 && tx.Transaction.Signatures[0] == txHash {
			return i, nil
		}
	}
	return -1, fmt.Errorf("交易 %s 未在 slot %d 中找到", txHash, slot)
}

// ===== 工具方法 =====

func doPost(payload map[string]interface{}) ([]byte, error) {
	reqBytes, _ := json.Marshal(payload)
	resp, err := http.Post(solanaRPC, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP 状态码 %d: %s", resp.StatusCode, buf.String())
	}
	return buf.Bytes(), nil
}

func mustParseUint8(s string) uint8 {
	var val uint8
	_, err := fmt.Sscanf(s, "%d", &val)
	if err != nil {
		log.Fatalf("参数解析失败 [%s]: %v", s, err)
	}
	return val
}
