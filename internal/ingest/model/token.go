package model

type Token struct {
	TokenAddress string // VARCHAR(44)，代币地址
	Decimals     int16  // SMALLINT，精度
	Source       int16
	TotalSupply  string // DECIMAL(20, 0)，使用 string 防止精度丢失

	Name    string // VARCHAR(128)，代币名（可选）
	Symbol  string // VARCHAR(64)，符号（可选）
	URI     string // VARCHAR(256)，链接 URI（可选）
	Creator string // VARCHAR(44)，图标（可选）

	CreateAt   int32 // 创建时间（blockTime 或初次发现时间）
	UpdateAt   int32 // 最后更新时间（落库时间）
	IsCreating bool
}
