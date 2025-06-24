package pool

import "database/sql"

type QueryPoolService struct {
	DB *sql.DB
}

func NewQueryPoolService(db *sql.DB) *QueryPoolService {
	return &QueryPoolService{DB: db}
}
