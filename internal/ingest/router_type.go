package ingest

import (
	"fmt"
	"strings"
)

type RouterType int

const (
	RouterEvent RouterType = iota
	RouterBalance
)

func ParseRouterType(s string) (RouterType, error) {
	switch strings.ToLower(s) {
	case "event":
		return RouterEvent, nil
	case "balance":
		return RouterBalance, nil
	default:
		return 0, fmt.Errorf("invalid ingest_type: %s (must be 'event' or 'balance')", s)
	}
}
