package mfa

import (
	"slices"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	mfadomain "github.com/krackl1n/careeria/services/identity/internal/domain/mfa"
)

func toDomain(identityID identitydomain.ID, rows []factorRow) *mfadomain.MFA {
	e := &mfadomain.MFA{
		IdentityID: identityID,
		Factors:    make([]mfadomain.Factor, 0, len(rows)),
	}

	for _, v := range rows {
		f := mfadomain.Factor{
			ID:          mfadomain.FactorID(v.ID),
			Type:        mfadomain.FactorType(v.FactorType),
			Name:        domain.ClonePointer(v.Name),
			IsPrimary:   v.IsPrimary,
			ConfirmedAt: domain.ClonePointer(v.ConfirmedAt),
			CreatedAt:   v.CreatedAt,
			UpdatedAt:   v.UpdatedAt,
			LastUsedAt:  domain.ClonePointer(v.LastUsedAt),
			DisabledAt:  domain.ClonePointer(v.DisabledAt),
		}

		if v.TOTPEncryptionKeyRef != nil {
			f.TOTP = &mfadomain.TOTP{
				SecretEncrypted:  mfadomain.EncryptedSecret(slices.Clone(v.TOTPSecretEncrypted)),
				EncryptionKeyRef: mfadomain.EncryptionKeyRef(*v.TOTPEncryptionKeyRef),
				Algorithm:        mfadomain.TOTPAlgorithm(*v.TOTPAlgorithm),
				Digits:           int(*v.TOTPDigits),
				PeriodSeconds:    int(*v.TOTPPeriodSeconds),
				LastUsedTimeStep: domain.ClonePointer(v.TOTPLastUsedTimeStep),
				UpdatedAt:        *v.TOTPUpdatedAt,
			}
		}

		if v.WebAuthnSignCount != nil {
			ts := make([]mfadomain.Transport, 0, len(v.WebAuthnTransports))

			for _, x := range v.WebAuthnTransports {
				ts = append(ts, mfadomain.Transport(x))
			}

			f.WebAuthn = &mfadomain.WebAuthnCredential{
				CredentialID:   mfadomain.CredentialID(slices.Clone(v.WebAuthnCredentialID)),
				PublicKey:      mfadomain.PublicKey(slices.Clone(v.WebAuthnPublicKey)),
				SignCount:      *v.WebAuthnSignCount,
				AAGUID:         domain.ClonePointer(v.WebAuthnAAGUID),
				Transports:     ts,
				BackupEligible: *v.WebAuthnBackupEligible,
				BackupState:    *v.WebAuthnBackupState,
				CreatedAt:      *v.WebAuthnCreatedAt,
				LastUsedAt:     domain.ClonePointer(v.WebAuthnLastUsedAt),
			}
		}

		e.Factors = append(e.Factors, f)
	}

	return e
}

func transportsToRow(values []mfadomain.Transport) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}

	return result
}
