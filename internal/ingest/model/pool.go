package model

type Pool struct {
	PoolAddress  string
	Dex          int16
	TokenAddress string
	QuoteAddress string
	TokenAccount string
	QuoteAccount string
	CreateAt     int32 // 区块时间（秒级）
	UpdateAt     int32 // 区块时间（秒级）
}
