package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	user := "user"
	password := "test"
	host := "ld-xxxxxx.lindorm.rds.aliyuncs.com"
	port := 33060
	timeout := "5s"

	// 注意：不指定 database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%s", user, password, host, port, timeout)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open failed: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("Available databases:")
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			log.Fatal(err)
		}
		fmt.Println(" -", dbName)
	}
}
