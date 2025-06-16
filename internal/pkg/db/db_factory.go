package db

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
	"time"
)

type DBType string

const (
	PG      DBType = "pg"
	Lindorm DBType = "lindorm"
)

// DBClient 封装了 gorm.DB 和数据库类型
type DBClient struct {
	DB      *gorm.DB
	Dialect DBType
}

// NewDBClient 创建一个数据库连接客户端
func NewDBClient(dsn string, dialect DBType) (*DBClient, error) {
	var dial gorm.Dialector
	switch dialect {
	case PG:
		dial = postgres.Open(dsn)
	case Lindorm:
		dial = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported db type: %s", dialect)
	}

	db, err := gorm.Open(dial, &gorm.Config{
		Logger: NewZapGormLogger(gormlog.Info),
	})
	if err != nil {
		return nil, err
	}

	// Lindorm 不支持事务，需要关闭默认事务包装
	if dialect == Lindorm {
		db = db.Session(&gorm.Session{SkipDefaultTransaction: true})
	}

	return &DBClient{
		DB:      db,
		Dialect: dialect,
	}, nil
}

// ZapGormLogger 实现 GORM 的 logger.Interface，输出到自定义 zap logger
type ZapGormLogger struct {
	level gormlog.LogLevel
}

func NewZapGormLogger(level gormlog.LogLevel) *ZapGormLogger {
	return &ZapGormLogger{level: level}
}

func (l *ZapGormLogger) LogMode(level gormlog.LogLevel) gormlog.Interface {
	return &ZapGormLogger{level: level}
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlog.Info {
		logger.Infof(msg, data...)
	}
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlog.Warn {
		logger.Warnf(msg, data...)
	}
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlog.Error {
		logger.Errorf(msg, data...)
	}
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlog.Silent {
		return
	}

	sql, rows := fc()
	elapsed := time.Since(begin)

	switch {
	case err != nil && l.level >= gormlog.Error:
		logger.Errorf("[GORM] %s | %d rows | %v | err=%v", sql, rows, elapsed, err)
	case l.level >= gormlog.Info:
		logger.Infof("[GORM] %s | %d rows | %v", sql, rows, elapsed)
	}
}
