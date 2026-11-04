package serviceaccount

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type ServiceAccount struct {
	ID             ID
	OrganizationID OrganizationID

	AccountKey  AccountKey
	Name        string
	Description *string

	Status Status

	CreatedByIdentityID *identitydomain.ID
	Credentials         []Credential

	CreatedAt  time.Time
	UpdatedAt  time.Time
	DisabledAt *time.Time
	DeletedAt  *time.Time
}

func New(
	id ID,
	organizationID OrganizationID,
	accountKey AccountKey,
	name string,
	description *string,
	createdByIdentityID *identitydomain.ID,
	credentialID CredentialID,
	secretHash SecretHash,
	secretPrefix *SecretPrefix,
	credentialName *string,
	credentialExpiresAt *time.Time,
	now time.Time,
) (*ServiceAccount, error) {
	if !id.IsValid() {
		return nil, errors.New("create service account: identifier is required")
	}
	if !organizationID.IsValid() {
		return nil, errors.New("create service account: organization identifier is required")
	}
	if createdByIdentityID != nil && !createdByIdentityID.IsValid() {
		return nil, errors.New("create service account: creator identifier is invalid")
	}
	if now.IsZero() {
		return nil, errors.New("create service account: time is invalid")
	}
	normalizedName, ok := domain.NormalizeRequiredText(name)
	if !ok {
		return nil, errors.New("create service account: name is required")
	}
	credential, err := newCredential(
		credentialID,
		secretHash,
		secretPrefix,
		credentialName,
		createdByIdentityID,
		credentialExpiresAt,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create service account: %w", err)
	}

	account := &ServiceAccount{
		ID:             id,
		OrganizationID: organizationID,

		AccountKey:  accountKey,
		Name:        normalizedName,
		Description: domain.NormalizeOptionalText(description),

		Status: StatusActive,

		CreatedByIdentityID: domain.ClonePointer(createdByIdentityID),
		Credentials:         []Credential{credential},

		CreatedAt: now,
		UpdatedAt: now,
	}
	return account, nil
}

func (a *ServiceAccount) UpdateMetadata(name string, description *string, now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("update service account metadata: %w", err)
	}
	normalizedName, ok := domain.NormalizeRequiredText(name)
	if !ok {
		return errors.New("update service account metadata: name is required")
	}
	normalizedDescription := domain.NormalizeOptionalText(description)
	if a.Name == normalizedName && domain.EqualOptionalText(a.Description, normalizedDescription) {
		return nil
	}

	a.Name = normalizedName
	a.Description = normalizedDescription
	a.touch(now)
	return nil
}

func (a *ServiceAccount) AddCredential(
	id CredentialID,
	secretHash SecretHash,
	secretPrefix *SecretPrefix,
	name *string,
	createdByIdentityID *identitydomain.ID,
	expiresAt *time.Time,
	now time.Time,
) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add service account credential: %w", err)
	}
	credential, err := newCredential(
		id,
		secretHash,
		secretPrefix,
		name,
		createdByIdentityID,
		expiresAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("add service account credential: %w", err)
	}
	if a.credentialIndex(credential.ID) >= 0 {
		return errors.New("add service account credential: identifier already exists")
	}

	a.Credentials = append(a.Credentials, credential)
	a.touch(now)

	return nil
}

func (a *ServiceAccount) RenameCredential(id CredentialID, name *string, now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("rename service account credential: %w", err)
	}
	index := a.credentialIndex(id)
	if index < 0 {
		return fmt.Errorf("rename service account credential: %w", ErrCredentialNotFound)
	}
	changed, err := a.Credentials[index].Rename(name, now)
	if err != nil {
		return fmt.Errorf("rename service account credential: %w", err)
	}
	if !changed {
		return nil
	}

	a.touch(now)

	return nil
}

func (a *ServiceAccount) RevokeCredential(id CredentialID, reason string, now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("revoke service account credential: %w", err)
	}
	index := a.credentialIndex(id)
	if index < 0 {
		return fmt.Errorf("revoke service account credential: %w", ErrCredentialNotFound)
	}
	if err := a.Credentials[index].Revoke(now, reason); err != nil {
		return fmt.Errorf("revoke service account credential: %w", err)
	}

	a.touch(now)

	return nil
}

func (a *ServiceAccount) ValidateCredential(id CredentialID, now time.Time) error {
	if a == nil {
		return errors.New("validate service account credential: service account is nil")
	}
	if now.IsZero() || now.Before(a.UpdatedAt) {
		return errors.New("validate service account credential: time is invalid")
	}
	if a.Status != StatusActive {
		return fmt.Errorf("validate service account credential: %w", ErrInvalidStatus)
	}
	index := a.credentialIndex(id)
	if index < 0 {
		return fmt.Errorf("validate service account credential: %w", ErrCredentialNotFound)
	}
	if err := a.Credentials[index].Validate(now); err != nil {
		return fmt.Errorf("validate service account credential: %w", err)
	}

	return nil
}

func (a *ServiceAccount) RecordCredentialUsed(id CredentialID, now time.Time) error {
	if err := a.ValidateCredential(id, now); err != nil {
		return fmt.Errorf("record service account credential use: %w", err)
	}

	index := a.credentialIndex(id)
	if err := a.Credentials[index].RecordUsed(now); err != nil {
		return fmt.Errorf("record service account credential use: %w", err)
	}

	a.touch(now)

	return nil
}

func (a *ServiceAccount) Disable(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("disable service account: %w", err)
	}
	if a.Status != StatusActive {
		return fmt.Errorf("disable service account: %w", ErrInvalidStatus)
	}

	a.Status = StatusDisabled
	a.DisabledAt = domain.ClonePointer(&now)
	a.touch(now)
	return nil
}

func (a *ServiceAccount) Enable(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enable service account: %w", err)
	}
	if a.Status != StatusDisabled {
		return fmt.Errorf("enable service account: %w", ErrInvalidStatus)
	}
	if !a.hasUsableCredential(now) {
		return fmt.Errorf("enable service account: %w", ErrUsableCredentialRequired)
	}

	a.Status = StatusActive
	a.DisabledAt = nil
	a.touch(now)
	return nil
}

func (a *ServiceAccount) Delete(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("delete service account: %w", err)
	}

	for index := range a.Credentials {
		if err := a.Credentials[index].Revoke(now, "service_account_deleted"); err != nil && !errors.Is(err, ErrCredentialRevoked) {
			return fmt.Errorf("delete service account: revoke credential: %w", err)
		}
	}
	a.Status = StatusDeleted
	a.DeletedAt = domain.ClonePointer(&now)
	a.touch(now)
	return nil
}

func (a *ServiceAccount) ensureChangeAllowed(now time.Time) error {
	if a == nil {
		return errors.New("service account is nil")
	}
	if a.Status == StatusDeleted {
		return ErrInvalidStatus
	}
	if now.IsZero() || now.Before(a.UpdatedAt) {
		return errors.New("service account change time is invalid")
	}

	return nil
}

func (a *ServiceAccount) touch(now time.Time) {
	a.UpdatedAt = now
}

func (a *ServiceAccount) hasUsableCredential(now time.Time) bool {
	for index := range a.Credentials {
		if a.Credentials[index].IsUsable(now) {
			return true
		}
	}

	return false
}

func (a *ServiceAccount) credentialIndex(id CredentialID) int {
	for index := range a.Credentials {
		if a.Credentials[index].ID == id {
			return index
		}
	}

	return -1
}
