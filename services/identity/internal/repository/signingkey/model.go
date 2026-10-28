package signingkey

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID            uuid.UUID
	KeyID         string
	Algorithm     string
	PublicJWK     []byte
	PrivateKeyRef string

	Status string

	CreatedAt     time.Time
	NotBefore     time.Time
	ActivatedAt   *time.Time
	RetiringAt    *time.Time
	RetiredAt     *time.Time
	ExpiresAt     *time.Time
	CompromisedAt *time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var model row

	if err := source.Scan(
		&model.ID,
		&model.KeyID,
		&model.Algorithm,
		&model.PublicJWK,
		&model.PrivateKeyRef,
		&model.Status,
		&model.CreatedAt,
		&model.NotBefore,
		&model.ActivatedAt,
		&model.RetiringAt,
		&model.RetiredAt,
		&model.ExpiresAt,
		&model.CompromisedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan signing key row: %w", err)
	}

	return model, nil
}

func scanRows(rows pgx.Rows) ([]row, error) {
	models := make([]row, 0)

	for rows.Next() {
		model, err := scanRow(rows)
		if err != nil {
			return nil, err
		}

		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signing keys: %w", err)
	}

	return models, nil
}
