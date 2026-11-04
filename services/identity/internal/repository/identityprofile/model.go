package identityprofile

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type row struct {
	IdentityID uuid.UUID
	FamilyName string
	GivenName  string
	MiddleName *string

	Source         string
	AssuranceLevel string
	Version        int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var model row

	if err := source.Scan(
		&model.IdentityID,
		&model.FamilyName,
		&model.GivenName,
		&model.MiddleName,
		&model.Source,
		&model.AssuranceLevel,
		&model.Version,
		&model.CreatedAt,
		&model.UpdatedAt,
	); err != nil {
		return row{}, fmt.Errorf("scan identity profile row: %w", err)
	}

	return model, nil
}
