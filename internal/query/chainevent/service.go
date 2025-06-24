package chainevent

import "database/sql"

type QueryChainEventService struct {
	DB *sql.DB
}

func NewQueryChainEventService(db *sql.DB) *QueryChainEventService {
	return &QueryChainEventService{DB: db}
}
