package identity

import (
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func conflictError(err error) error {
	switch {
	case sharedpostgres.IsUniqueViolation(err, "identity_emails_email_uidx"):
		return identitydomain.ErrEmailAlreadyExists
	case sharedpostgres.IsUniqueViolation(err, "identity_phones_phone_uidx"):
		return identitydomain.ErrPhoneAlreadyExists
	case sharedpostgres.IsUniqueViolation(err, "external_identity_accounts_provider_subject_uidx"):
		return identitydomain.ErrExternalAccountExists
	default:
		return nil
	}
}
