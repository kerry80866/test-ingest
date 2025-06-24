package utils

import "github.com/mr-tron/base58"

const (
	solMintStr  = "So11111111111111111111111111111111111111111"
	wsolMintStr = "So11111111111111111111111111111111111111112"
	usdcMintStr = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	usdtMintStr = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
)

var (
	nativeSolMint = make([]byte, 32)
	nativeSolStr  = base58.Encode(nativeSolMint)

	solMint, _  = base58.Decode(solMintStr)
	wsolMint, _ = base58.Decode(wsolMintStr)
	usdcMint, _ = base58.Decode(usdcMintStr)
	usdtMint, _ = base58.Decode(usdtMintStr)
)

var knownTokenMintToID = map[string]string{
	string(nativeSolMint): "0",
	string(solMint):       "0",
	string(wsolMint):      "1",
	string(usdcMint):      "2",
	string(usdtMint):      "3",
}

var knownTokenToID = map[string]string{
	nativeSolStr: "0",
	solMintStr:   "0",
	wsolMintStr:  "1",
	usdcMintStr:  "2",
	usdtMintStr:  "3",
}

var knownIDToToken = map[string]string{
	"0": solMintStr,
	"1": wsolMintStr,
	"2": usdcMintStr,
	"3": usdtMintStr,
}

func EncodeTokenAddress(addr string) string {
	if id, ok := knownTokenToID[addr]; ok {
		return id
	}
	return addr
}

func DecodeTokenAddress(id string) string {
	if addr, ok := knownIDToToken[id]; ok {
		return addr
	}
	return id
}
