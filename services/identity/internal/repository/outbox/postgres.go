package outbox

import (
	"time"

	sharedoutbox "github.com/krackl1n/careeria/outbox"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	"github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

type PostgresRepository struct {
	client     sharedpostgres.ExecutorProvider
	retryDelay time.Duration
}

func NewPostgresRepository(client sharedpostgres.ExecutorProvider, retryDelay time.Duration) *PostgresRepository {
	return &PostgresRepository{client: client, retryDelay: retryDelay}
}

var _ sharedoutbox.Repository[event.Message] = (*PostgresRepository)(nil)
