package utils

import (
	"math"
)

func Pow10(n uint32) float64 {
	switch n {
	case 6:
		return 1e6
	case 9:
		return 1e9
	case 0:
		return 1
	case 18:
		return 1e18
	default:
		return math.Pow10(int(n)) // math.Pow10 接收 int，已由 uint32 显式转换
	}
}
