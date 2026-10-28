package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) GetByEmail(
	ctx context.Context,
	email identitydomain.EmailAddress,
) (*identitydomain.Identity, error) {
	var id uuid.UUID
	err := r.client.Executor(ctx).QueryRow(ctx, `
		SELECT identity_id
		FROM public.identity_emails
		WHERE email = $1
			AND deleted_at IS NULL`,
		string(email),
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identitydomain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get identity by email: %w", err)
	}

	return r.Get(ctx, identitydomain.ID(id))
}

func (r *PostgresRepository) GetByPhone(
	ctx context.Context,
	phone identitydomain.PhoneNumber,
) (*identitydomain.Identity, error) {
	var id uuid.UUID
	err := r.client.Executor(ctx).QueryRow(ctx, `
		SELECT identity_id
		FROM public.identity_phones
		WHERE phone = $1
			AND deleted_at IS NULL`,
		string(phone),
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identitydomain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get identity by phone: %w", err)
	}

	return r.Get(ctx, identitydomain.ID(id))
}
