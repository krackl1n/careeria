package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestAsPgError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantNil  bool
	}{
		{
			name:     "direct pg error",
			err:      &pgconn.PgError{Code: CodeUniqueViolation, ConstraintName: "user_emails_email_key"},
			wantCode: CodeUniqueViolation,
		},
		{
			name:     "wrapped pg error",
			err:      fmt.Errorf("wrap1: %w", fmt.Errorf("wrap2: %w", &pgconn.PgError{Code: CodeForeignKeyViolation, ConstraintName: "user_emails_user_id_fkey"})),
			wantCode: CodeForeignKeyViolation,
		},
		{
			name:    "non pg error",
			err:     errors.New("boom"),
			wantNil: true,
		},
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AsPgError(tt.err)
			if tt.wantNil {
				require.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			require.Equal(t, tt.wantCode, got.Code)
		})
	}
}

func TestIsCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
		want bool
	}{
		{
			name: "matches unique code",
			err:  &pgconn.PgError{Code: CodeUniqueViolation},
			code: CodeUniqueViolation,
			want: true,
		},
		{
			name: "wrapped and matches",
			err:  fmt.Errorf("wrap: %w", &pgconn.PgError{Code: CodeForeignKeyViolation}),
			code: CodeForeignKeyViolation,
			want: true,
		},
		{
			name: "different code",
			err:  &pgconn.PgError{Code: CodeUniqueViolation},
			code: CodeForeignKeyViolation,
			want: false,
		},
		{
			name: "not pg error",
			err:  errors.New("boom"),
			code: CodeUniqueViolation,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			code: CodeUniqueViolation,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCode(tt.err, tt.code)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsConstraint(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		code        string
		constraints []string
		want        bool
	}{
		{
			name: "match code and single constraint",
			err: &pgconn.PgError{
				Code:           CodeUniqueViolation,
				ConstraintName: "user_emails_email_key",
			},
			code:        CodeUniqueViolation,
			constraints: []string{"user_emails_email_key"},
			want:        true,
		},
		{
			name: "match code and one of many constraints",
			err: &pgconn.PgError{
				Code:           CodeUniqueViolation,
				ConstraintName: "user_phones_phone_key",
			},
			code:        CodeUniqueViolation,
			constraints: []string{"user_emails_email_key", "user_phones_phone_key"},
			want:        true,
		},
		{
			name: "match code without constraints filter",
			err: &pgconn.PgError{
				Code:           CodeForeignKeyViolation,
				ConstraintName: "whatever",
			},
			code: CodeForeignKeyViolation,
			want: true,
		},
		{
			name: "code matches but constraint differs",
			err: &pgconn.PgError{
				Code:           CodeUniqueViolation,
				ConstraintName: "service_accounts_key_key",
			},
			code:        CodeUniqueViolation,
			constraints: []string{"user_emails_email_key"},
			want:        false,
		},
		{
			name: "constraint matches but code differs",
			err: &pgconn.PgError{
				Code:           CodeForeignKeyViolation,
				ConstraintName: "user_emails_email_key",
			},
			code:        CodeUniqueViolation,
			constraints: []string{"user_emails_email_key"},
			want:        false,
		},
		{
			name: "wrapped pg error",
			err: fmt.Errorf("wrap: %w", &pgconn.PgError{
				Code:           CodeForeignKeyViolation,
				ConstraintName: "service_account_credentials_service_account_id_fkey",
			}),
			code:        CodeForeignKeyViolation,
			constraints: []string{"service_account_credentials_service_account_id_fkey"},
			want:        true,
		},
		{
			name: "not pg error",
			err:  errors.New("boom"),
			code: CodeUniqueViolation,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			code: CodeUniqueViolation,
			want: false,
		},
		{
			name: "empty constraint does not match specific non-empty constraint name",
			err: &pgconn.PgError{
				Code:           CodeUniqueViolation,
				ConstraintName: "user_emails_email_key",
			},
			code:        CodeUniqueViolation,
			constraints: []string{""},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConstraint(tt.err, tt.code, tt.constraints...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		constraints []string
		want        bool
	}{
		{
			name: "unique with matching constraint",
			err: &pgconn.PgError{
				Code:           CodeUniqueViolation,
				ConstraintName: "user_emails_email_key",
			},
			constraints: []string{"user_emails_email_key"},
			want:        true,
		},
		{
			name: "unique without constraints",
			err: &pgconn.PgError{
				Code:           CodeUniqueViolation,
				ConstraintName: "x",
			},
			want: true,
		},
		{
			name: "foreign key is not unique",
			err: &pgconn.PgError{
				Code: CodeForeignKeyViolation,
			},
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non pg error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUniqueViolation(tt.err, tt.constraints...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsForeignKeyViolation(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		constraints []string
		want        bool
	}{
		{
			name: "foreign key with matching constraint",
			err: &pgconn.PgError{
				Code:           CodeForeignKeyViolation,
				ConstraintName: "user_password_credentials_user_id_fkey",
			},
			constraints: []string{"user_password_credentials_user_id_fkey"},
			want:        true,
		},
		{
			name: "foreign key without constraints",
			err: &pgconn.PgError{
				Code:           CodeForeignKeyViolation,
				ConstraintName: "x",
			},
			want: true,
		},
		{
			name: "unique is not foreign key",
			err: &pgconn.PgError{
				Code: CodeUniqueViolation,
			},
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "non pg error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsForeignKeyViolation(tt.err, tt.constraints...)
			require.Equal(t, tt.want, got)
		})
	}
}
