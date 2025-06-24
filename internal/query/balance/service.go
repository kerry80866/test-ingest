package balance

import "database/sql"

type QueryBalanceService struct {
	DB *sql.DB
}

func NewQueryBalanceService(db *sql.DB) *QueryBalanceService {
	return &QueryBalanceService{DB: db}
}
