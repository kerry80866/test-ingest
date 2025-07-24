package db

import (
	"database/sql"
	"fmt"
	"github.com/mr-tron/base58"
	"strings"
)

func QueryTokenDecimalsFromPG(db *sql.DB, tokens []string, tokenMap map[string]int16) (e error) {
	defer func() {
		if r := recover(); r != nil {
			e = fmt.Errorf("QueryTokenDecimalsFromPG panic: %v", r)
		}
	}()

	if len(tokens) == 0 {
		return nil
	}

	// 去重 + 排除已缓存
	unique := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if _, ok := tokenMap[t]; !ok {
			unique[t] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return nil
	}

	args := make([]interface{}, 0, len(unique))
	placeholders := make([]string, 0, len(unique))
	i := 1
	for t := range unique {
		decoded, err := base58.Decode(t)
		if err != nil {
			continue // 非法 token，跳过
		}
		args = append(args, decoded)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	if len(args) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		SELECT address, decimals
		FROM public.token
		WHERE address IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var addr []byte
		var dec int16
		if err := rows.Scan(&addr, &dec); err != nil {
			return err
		}
		if dec > 0 {
			base58Addr := base58.Encode(addr)
			tokenMap[base58Addr] = dec
		}
	}
	return rows.Err() // 避免遗漏扫描错误
}
