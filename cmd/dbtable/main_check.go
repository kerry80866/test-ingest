package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// === 检查 MySQL chain_event 表 ===
	mysqlDSN := "user:password@tcp(your-host:3306)/your_database?charset=utf8mb4&parseTime=true&loc=Local"
	mysqlDB, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("连接 MySQL/Lindorm 失败: %v", err)
	}
	defer mysqlDB.Close()

	checkTable(mysqlDB, "mysql", "chain_event")

	// === 检查 PostgreSQL 表 ===
	pgDSN := "postgres://user:password@localhost:5432/dbname?sslmode=disable"
	pgDB, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: %v", err)
	}
	defer pgDB.Close()

	checkTable(pgDB, "postgres", "pool")
	checkTable(pgDB, "postgres", "token")
	checkTable(pgDB, "postgres", "balance_p0")
	checkTable(pgDB, "postgres", "balance_p63")
	checkTable(pgDB, "postgres", "balance_p127")
}

// checkTable 封装日志打印
func checkTable(db *sql.DB, dbType, table string) {
	exists, err := TableExists(db, dbType, table)
	if err != nil {
		log.Printf("检查表 [%s] 失败: %v", table, err)
		return
	}
	if exists {
		log.Printf("✅ [%s] 表存在 (%s)", table, dbType)
	} else {
		log.Printf("❌ [%s] 表不存在 (%s)", table, dbType)
	}
}

// TableExists 检查表是否存在，支持 MySQL 和 PostgreSQL
func TableExists(db *sql.DB, dbType string, tableName string) (bool, error) {
	var query string
	switch strings.ToLower(dbType) {
	case "mysql":
		query = fmt.Sprintf(`
SELECT COUNT(*) > 0 FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = '%s'`, tableName)

	case "postgres":
		query = fmt.Sprintf(`
SELECT EXISTS (
	SELECT 1 FROM information_schema.tables 
	WHERE table_schema = 'public' AND table_name = '%s'
)`, tableName)

	default:
		return false, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	var exists bool
	err := db.QueryRow(query).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("执行查询失败: %w", err)
	}
	return exists, nil
}
