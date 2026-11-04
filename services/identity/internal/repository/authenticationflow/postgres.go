package authenticationflow

import sharedpostgres "github.com/krackl1n/careeria/postgres"

type PostgresReader struct {
	client sharedpostgres.ExecutorProvider
}

func NewPostgresReader(client sharedpostgres.ExecutorProvider) *PostgresReader {
	return &PostgresReader{client: client}
}
