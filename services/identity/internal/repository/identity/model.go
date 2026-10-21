package identity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	ID     uuid.UUID
	Status string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type emailRow struct {
	ID         uuid.UUID
	IdentityID uuid.UUID
	Email      string
	IsPrimary  bool
	VerifiedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type phoneRow struct {
	ID         uuid.UUID
	IdentityID uuid.UUID
	Phone      string
	IsPrimary  bool
	VerifiedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type externalAccountRow struct {
	ID         uuid.UUID
	IdentityID uuid.UUID
	Provider   string
	Subject    string

	LinkedAt            time.Time
	LastAuthenticatedAt *time.Time
	DisabledAt          *time.Time
	DeletedAt           *time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var model row

	if err := source.Scan(
		&model.ID,
		&model.Status,
		&model.CreatedAt,
		&model.UpdatedAt,
		&model.DeletedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan identity row: %w", err)
	}

	return model, nil
}

func scanEmailRows(rows pgx.Rows) ([]emailRow, error) {
	models := make([]emailRow, 0)

	for rows.Next() {
		var model emailRow

		if err := rows.Scan(
			&model.ID,
			&model.IdentityID,
			&model.Email,
			&model.IsPrimary,
			&model.VerifiedAt,
			&model.CreatedAt,
			&model.UpdatedAt,
			&model.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan identity email row: %w", err)
		}

		models = append(models, model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate identity emails: %w", err)
	}

	return models, nil
}

func scanPhoneRows(rows pgx.Rows) ([]phoneRow, error) {
	models := make([]phoneRow, 0)

	for rows.Next() {
		var model phoneRow

		if err := rows.Scan(
			&model.ID,
			&model.IdentityID,
			&model.Phone,
			&model.IsPrimary,
			&model.VerifiedAt,
			&model.CreatedAt,
			&model.UpdatedAt,
			&model.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan identity phone row: %w", err)
		}

		models = append(models, model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate identity phones: %w", err)
	}

	return models, nil
}

func scanExternalAccountRows(rows pgx.Rows) ([]externalAccountRow, error) {
	models := make([]externalAccountRow, 0)

	for rows.Next() {
		var model externalAccountRow

		if err := rows.Scan(
			&model.ID,
			&model.IdentityID,
			&model.Provider,
			&model.Subject,
			&model.LinkedAt,
			&model.LastAuthenticatedAt,
			&model.DisabledAt,
			&model.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan external identity account row: %w", err)
		}

		models = append(models, model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate external identity accounts: %w", err)
	}

	return models, nil
}
