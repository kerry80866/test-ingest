package utils

import "strconv"

// Uint64ToString 将 uint64 转为数据库可用的 DECIMAL(20, 0) 格式字符串
func Uint64ToString(val uint64) string {
	return strconv.FormatUint(val, 10)
}

func ParseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
