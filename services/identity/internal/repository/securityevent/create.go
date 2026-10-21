package securityevent

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
)

func (r *PostgresRepository) Create(ctx context.Context, e *eventdomain.SecurityEvent) error {
	if e == nil {
		return errors.New("create security event: entity is nil")
	}

	return r.createRow(ctx, toRow(e))
}

func (r *PostgresRepository) CreateBulk(
	ctx context.Context,
	events []*eventdomain.SecurityEvent,
) error {
	b := &pgx.Batch{}

	for _, e := range events {
		if e == nil {
			return errors.New("create security events: entity is nil")
		}

		m := toRow(e)

		b.Queue(`
			INSERT INTO public.security_events (
				id,
				event_type,
				result,
				actor_type,
				identity_id,
				service_account_id,
				session_id,
				oauth_client_id,
				identifier_hash,
				ip_address,
				user_agent,
				correlation_id,
				occurred_at,
				metadata
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			m.ID,
			m.EventType,
			m.Result,
			m.ActorType,
			m.IdentityID,
			m.ServiceAccountID,
			m.SessionID,
			m.OAuthClientID,
			m.IdentifierHash,
			m.IPAddress,
			m.UserAgent,
			m.CorrelationID,
			m.OccurredAt,
			m.Metadata,
		)
	}

	if b.Len() == 0 {
		return nil
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		return fmt.Errorf("create security events: %w", err)
	}

	return nil
}

func (r *PostgresRepository) createRow(ctx context.Context, m row) error {
	_, err := r.client.Executor(ctx).Exec(ctx, `
		INSERT INTO public.security_events (
			id,
			event_type,
			result,
			actor_type,
			identity_id,
			service_account_id,
			session_id,
			oauth_client_id,
			identifier_hash,
			ip_address,
			user_agent,
			correlation_id,
			occurred_at,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		m.ID,
		m.EventType,
		m.Result,
		m.ActorType,
		m.IdentityID,
		m.ServiceAccountID,
		m.SessionID,
		m.OAuthClientID,
		m.IdentifierHash,
		m.IPAddress,
		m.UserAgent,
		m.CorrelationID,
		m.OccurredAt,
		m.Metadata,
	)

	if err != nil {
		return fmt.Errorf("create security event: %w", err)
	}

	return nil
}
