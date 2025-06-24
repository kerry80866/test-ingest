package model

type ChainEvent struct {
	EventIDHash int32 // 事件哈希
	EventID     int64 // 原始事件 ID
	EventType   int16 // 事件类型（如 TRADE / LP / MINT 等）
	Dex         int16 // DEX 标识

	UserWallet string // 交易发起人钱包地址
	ToWallet   string // 接收人钱包地址（可能为空）

	PoolAddress string // 池子地址（可能为空）
	Token       string // 主 Token 地址（base）
	QuoteToken  string // Quote 地址（可选）

	TokenAmount string  // 精度高的金额，用 string 表示 DECIMAL(20, 0)
	QuoteAmount string  // 同上
	VolumeUsd   float64 // 估算总价值（单位 USD）
	PriceUsd    float64 // 单价（单位 USD）

	TxHash string // 交易哈希
	Signer string // 签名人地址（一般等于 UserWallet）

	BlockTime int32 // 区块时间戳（秒级）
	CreateAt  int32 // 写入数据库的时间（秒级）
}
