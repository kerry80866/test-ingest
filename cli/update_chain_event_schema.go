package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // 替换为你实际使用的 driver
)

func main() {
	// 替换为你的 DSN
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	steps := []struct {
		name  string
		query string
	}{
		{
			name:  "Drop idx_user_token_type_id_desc",
			query: "DROP INDEX IF EXISTS idx_user_token_type_id_desc ON chain_event",
		},
		{
			name:  "Drop idx_pool_type_id",
			query: "DROP INDEX IF EXISTS idx_pool_type_id ON chain_event",
		},
		{
			name:  "Alter MUTABILITY to MUTABLE_LATEST",
			query: "ALTER TABLE chain_event SET TBLPROPERTIES ('MUTABILITY' = 'MUTABLE_LATEST')",
		},
		{
			name: "Create idx_user_token_type_id_desc",
			query: `
				CREATE INDEX IF NOT EXISTS idx_user_token_type_id_desc
				ON chain_event(user_wallet, token, event_type, event_id DESC)
				WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA')`,
		},
		{
			name: "Create idx_pool_type_id",
			query: `
				CREATE INDEX IF NOT EXISTS idx_pool_type_id
				ON chain_event(pool_address, event_type, event_id DESC)
				WITH (INDEX_COVERED_TYPE = 'COVERED_ALL_COLUMNS_IN_SCHEMA')`,
		},
	}

	for _, step := range steps {
		log.Printf("Running step: %s", step.name)
		if _, err := db.ExecContext(ctx, step.query); err != nil {
			log.Fatalf("Step failed [%s]: %v", step.name, err)
		}
		log.Printf("Step succeeded: %s", step.name)
	}

	log.Println("All steps completed successfully.")
}
