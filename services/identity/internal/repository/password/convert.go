package password

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
)

func toDomain(
	credential credentialRow,
	authState authStateRow,
) *passworddomain.Password {
	var lockReason *passworddomain.LockReason
	if authState.LockReason != nil {
		value := passworddomain.LockReason(*authState.LockReason)
		lockReason = &value
	}

	return &passworddomain.Password{
		IdentityID: identitydomain.ID(credential.IdentityID),
		Credential: passworddomain.Credential{
			Hash:              passworddomain.Hash(credential.PasswordHash),
			MustChange:        credential.MustChange,
			PasswordUpdatedAt: credential.PasswordUpdatedAt,
			ExpiresAt:         domain.ClonePointer(credential.ExpiresAt),
			CompromisedAt:     domain.ClonePointer(credential.CompromisedAt),
			CreatedAt:         credential.CreatedAt,
			UpdatedAt:         credential.UpdatedAt,
		},
		AuthState: passworddomain.AuthState{
			FailedAttempts: authState.FailedAttempts,
			LastFailedAt:   domain.ClonePointer(authState.LastFailedAt),
			LockedUntil:    domain.ClonePointer(authState.LockedUntil),
			LockReason:     lockReason,
			UpdatedAt:      authState.UpdatedAt,
		},
	}
}

func lockReasonToRow(value *passworddomain.LockReason) *string {
	if value == nil {
		return nil
	}

	result := string(*value)
	return &result
}
