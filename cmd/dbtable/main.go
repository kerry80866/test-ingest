package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql" // 替换为你用的 driver
)

func main() {
	// 替换为你自己的 Lindorm/MySQL 数据源
	dsn := "dex:abc@tcp(ld-0iwr13uihv3zv8i65-proxy-lindorm-vpc.lindorm.aliyuncs.com:33060)/default?charset=utf8mb4&parseTime=true"
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
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');
`,
		`CREATE INDEX IF NOT EXISTS idx_balance_token
     ON balance(token_address)
     INCLUDE (owner_address, balance);`,

		`CREATE INDEX IF NOT EXISTS idx_balance_owner_token
     ON balance(owner_address, token_address)
     INCLUDE (balance);`,

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
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'IMMUTABLE_ROWS');
`,

		`CREATE INDEX IF NOT EXISTS idx_user_token_type_id_desc
     ON chain_event(user_wallet, token, event_type, event_id DESC)
     INCLUDE (
         dex, to_wallet, pool_address, quote_token,
         token_amount, quote_amount, volume_usd, price_usd,
         tx_hash, block_time
     );`,

		`CREATE INDEX IF NOT EXISTS idx_pool_type_id
     ON chain_event(pool_address, event_type, event_id DESC)
     INCLUDE (
         user_wallet,
         token_amount, quote_amount, volume_usd, price_usd,
         tx_hash, block_time
     );`,

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
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');
`,

		`CREATE INDEX IF NOT EXISTS idx_pool_token_quote
     ON pool(token_address, quote_address)
     WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA');`,

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
) WITH (CONSISTENCY = 'strong', MUTABILITY = 'MUTABLE_LATEST');
`,
	}

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err := db.Exec(stmt)
		if err != nil {
			log.Fatalf("执行第 %d 条建表语句失败: %v\nSQL:\n%s", i+1, err, stmt)
		}
		fmt.Printf("成功执行建表语句 #%d\n", i+1)
	}
}
