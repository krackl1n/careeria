package vacancy

import (
	"github.com/krackl1n/careeria/postgres"
)

type PostgresRepository struct {
	client postgres.ExecutorProvider
}

func NewPostgresRepository(client postgres.ExecutorProvider) *PostgresRepository {
	return &PostgresRepository{client: client}
}
