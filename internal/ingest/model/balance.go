package model

type Balance struct {
	AccountAddress string // 账户地址（TokenAccount）
	OwnerAddress   string // 所属钱包地址
	TokenAddress   string // Token 合约地址
	Balance        string // 用字符串避免 DECIMAL 精度误差
	LastEventID    int64  // 最后一次处理的事件 ID
}
