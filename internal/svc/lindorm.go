package svc

import (
	"database/sql"
	"dex-ingest-sol/internal/config"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MustInitLindorm 初始化 Lindorm/MySQL 连接，失败时 panic
func MustInitLindorm(conf config.LindormConf) *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Database,
		conf.Timeout,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Lindorm: %v", err))
	}

	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetMaxIdleConns(conf.MaxIdleConns)

	if dur, err := time.ParseDuration(conf.ConnMaxIdleTime); err == nil {
		db.SetConnMaxIdleTime(dur)
	} else {
		panic(fmt.Sprintf("invalid conn_max_idle_time: %v", err))
	}

	return db
}
