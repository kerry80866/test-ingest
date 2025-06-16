package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 替换为你自己的 MySQL 数据源
	dsn := "my_dev:123456@tcp(localhost:3306)/ddex?charset=utf8mb4&parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	statements := []string{
		`
CREATE TABLE IF NOT EXISTS balance (
    account_address VARCHAR(44) NOT NULL,
    owner_address   VARCHAR(44) NOT NULL,
    token_address   VARCHAR(44) NOT NULL,
    balance         DECIMAL(20, 0) NOT NULL,
    last_event_id   BIGINT NOT NULL,
    PRIMARY KEY (account_address)
);
`,
		// 注意：MySQL 不支持 IF NOT EXISTS for index
		`CREATE INDEX idx_balance_token ON balance(token_address, owner_address, balance);`,
		`CREATE INDEX idx_balance_owner_token ON balance(owner_address, token_address);`,

		`
CREATE TABLE IF NOT EXISTS chain_event (
     event_id_hash INT NOT NULL,
     event_id BIGINT NOT NULL,
     event_type SMALLINT NOT NULL,
     dex SMALLINT NOT NULL,

     user_wallet VARCHAR(44) NOT NULL,
     to_wallet VARCHAR(44) NOT NULL,

     pool_address VARCHAR(44) NOT NULL,
     token VARCHAR(44) NOT NULL,
     quote_token VARCHAR(44) NOT NULL,

     token_amount DECIMAL(20, 0) NOT NULL,
     quote_amount DECIMAL(20, 0) NOT NULL,
     volume_usd DOUBLE NOT NULL,
     price_usd DOUBLE NOT NULL,

     tx_hash VARCHAR(88) NOT NULL,
     signer VARCHAR(44) NOT NULL,

     block_time INT NOT NULL,
     create_at INT NOT NULL,

     PRIMARY KEY (event_id_hash, event_id)
);
`,

		`CREATE INDEX idx_user_token_type_id_desc 
		 ON chain_event(user_wallet, token, event_type, event_id DESC);`,

		`CREATE INDEX idx_pool_type_id 
		 ON chain_event(pool_address, event_type, event_id DESC);`,

		`
CREATE TABLE IF NOT EXISTS pool (
    pool_address VARCHAR(44) NOT NULL,
    dex SMALLINT NOT NULL,
    token_address VARCHAR(44) NOT NULL,
    quote_address VARCHAR(44) NOT NULL,
    token_account VARCHAR(44) NOT NULL,
    quote_account VARCHAR(44) NOT NULL,
    create_at INT NOT NULL,
    update_at INT NOT NULL,
	PRIMARY KEY (pool_address, token_account, quote_account)
);
`,

		`CREATE INDEX idx_pool_token_quote ON pool(token_address, quote_address);`,

		`
CREATE TABLE IF NOT EXISTS token (
    token_address VARCHAR(44) NOT NULL,
    decimals SMALLINT NOT NULL,
    source SMALLINT NOT NULL,
    total_supply DECIMAL(20, 0) NOT NULL,
    name VARCHAR(128) NOT NULL,
    symbol VARCHAR(64) NOT NULL,
    creator VARCHAR(44) NOT NULL,
    uri VARCHAR(256) NOT NULL,
    create_at INT NOT NULL,
    update_at INT NOT NULL,
    PRIMARY KEY (token_address)
);
`,
	}

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err := db.Exec(stmt)
		if err != nil {
			// 忽略索引已存在的错误（MySQL error code 1061）
			if strings.Contains(err.Error(), "1061") || strings.Contains(err.Error(), "Duplicate key name") {
				log.Printf("⚠️ 索引已存在（跳过）：%s", firstLine(stmt))
				continue
			}
			log.Fatalf("执行第 %d 条建表语句失败: %v\nSQL:\n%s", i+1, err, stmt)
		}
		fmt.Printf("✅ 成功执行建表语句 #%d\n", i+1)
	}
}

// 获取首行注释，方便日志中输出 SQL 片段
func firstLine(s string) string {
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			return l
		}
	}
	return ""
}
