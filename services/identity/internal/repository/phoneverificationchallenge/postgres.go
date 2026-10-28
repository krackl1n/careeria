package phoneverificationchallenge

import sharedpostgres "github.com/krackl1n/careeria/postgres"

type PostgresRepository struct {
	client sharedpostgres.ExecutorProvider
}

func NewPostgresRepository(client sharedpostgres.ExecutorProvider) *PostgresRepository {
	return &PostgresRepository{client: client}
}
