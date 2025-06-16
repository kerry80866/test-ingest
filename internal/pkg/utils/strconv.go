package utils

import "strconv"

// Uint64ToDecimal 将 uint64 转为数据库可用的 DECIMAL(20, 0) 格式字符串
func Uint64ToDecimal(val uint64) string {
	return strconv.FormatUint(val, 10)
}

func ParseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
