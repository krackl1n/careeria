package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	CodeUniqueViolation     = "23505"
	CodeForeignKeyViolation = "23503"
)

func IsCode(err error, code string) bool {
	pgErr := AsPgError(err)
	return pgErr != nil && pgErr.Code == code
}

func IsConstraint(err error, code string, constraints ...string) bool {
	pgErr := AsPgError(err)
	if pgErr == nil || pgErr.Code != code {
		return false
	}

	if len(constraints) == 0 {
		return true
	}

	for _, constraint := range constraints {
		if pgErr.ConstraintName == constraint {
			return true
		}
	}

	return false
}

func IsUniqueViolation(err error, constraints ...string) bool {
	return IsConstraint(err, CodeUniqueViolation, constraints...)
}

func IsForeignKeyViolation(err error, constraints ...string) bool {
	return IsConstraint(err, CodeForeignKeyViolation, constraints...)
}

func AsPgError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}

	return nil
}
