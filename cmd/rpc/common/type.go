package common

type LindormResult struct {
	PoolAddress   string
	AccountKey    int64
	TokenAddress  string
	QuoteAddress  string
	TokenDecimals int16
	QuoteDecimals int16
}

type Pool struct {
	PoolAddress   string
	AccountKey    int64
	Dex           int16
	TokenDecimals int16
	QuoteDecimals int16
	TokenAddress  string
	QuoteAddress  string
	TokenAccount  string
	QuoteAccount  string
	CreateAt      int32 // 区块时间（秒级）
	UpdateAt      int32 // 区块时间（秒级）
}
