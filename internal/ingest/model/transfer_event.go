package model

type TransferEvent struct {
	EventIDHash int32
	EventID     int64

	FromWallet string
	ToWallet   string

	Token    string
	Amount   string // 精度高的金额，用 string 表示 DECIMAL(20, 0)
	Decimals int16  // SMALLINT，精度

	TxHash string
	Signer string

	BlockTime int32 // 区块时间戳（秒级）
	CreateAt  int32 // 写入数据库的时间（秒级）
}
