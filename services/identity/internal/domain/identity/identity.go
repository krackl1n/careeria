package identity

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Identity struct {
	ID ID

	Status Status

	Emails           []Email
	Phones           []Phone
	ExternalAccounts []ExternalAccount

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewWithEmail(id ID, emailID EmailID, address EmailAddress, now time.Time) (*Identity, error) {
	entity, err := newIdentity(id, now)
	if err != nil {
		return nil, fmt.Errorf("create identity with email: %w", err)
	}

	email, err := newEmail(emailID, address, true, now)
	if err != nil {
		return nil, fmt.Errorf("create initial identity email: %w", err)
	}
	entity.Emails = append(entity.Emails, email)

	return entity, nil
}

func NewWithPhone(id ID, phoneID PhoneID, number PhoneNumber, now time.Time) (*Identity, error) {
	entity, err := newIdentity(id, now)
	if err != nil {
		return nil, fmt.Errorf("create identity with phone: %w", err)
	}

	phone, err := newPhone(phoneID, number, true, now)
	if err != nil {
		return nil, fmt.Errorf("create initial identity phone: %w", err)
	}
	entity.Phones = append(entity.Phones, phone)

	return entity, nil
}

func NewWithExternalAccount(
	id ID,
	accountID ExternalAccountID,
	provider ExternalProvider,
	subject ExternalSubject,
	now time.Time,
) (*Identity, error) {
	entity, err := newIdentity(id, now)
	if err != nil {
		return nil, fmt.Errorf("create identity with external account: %w", err)
	}

	account, err := newExternalAccount(accountID, provider, subject, now)
	if err != nil {
		return nil, fmt.Errorf("create initial external identity account: %w", err)
	}
	entity.ExternalAccounts = append(entity.ExternalAccounts, account)

	return entity, nil
}

func newIdentity(id ID, now time.Time) (*Identity, error) {
	if !id.IsValid() || now.IsZero() {
		return nil, errors.New("identity constructor arguments are invalid")
	}

	return &Identity{
		ID:        id,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (i *Identity) Activate(now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("activate identity: %w", err)
	}
	if i.Status != StatusPending {
		return fmt.Errorf("activate identity: %w", ErrInvalidTransition)
	}
	if !i.hasUsableLoginMethod() {
		return fmt.Errorf("activate identity without usable login method: %w", ErrLoginMethodRequired)
	}

	i.Status = StatusActive
	i.touch(now)
	return nil
}

func (i *Identity) Disable(now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("disable identity: %w", err)
	}
	if i.Status != StatusPending && i.Status != StatusActive {
		return fmt.Errorf("disable identity: %w", ErrInvalidTransition)
	}

	i.Status = StatusDisabled
	i.touch(now)
	return nil
}

func (i *Identity) Enable(now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enable identity: %w", err)
	}
	if i.Status != StatusDisabled {
		return fmt.Errorf("enable identity: %w", ErrInvalidTransition)
	}
	if !i.hasUsableLoginMethod() {
		return fmt.Errorf("enable identity without usable login method: %w", ErrLoginMethodRequired)
	}

	i.Status = StatusActive
	i.touch(now)
	return nil
}

func (i *Identity) Delete(now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("delete identity: %w", err)
	}

	i.Status = StatusDeleted
	i.DeletedAt = domain.ClonePointer(&now)
	i.touch(now)
	return nil
}

func (i *Identity) AddEmail(id EmailID, address EmailAddress, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add identity email: %w", err)
	}
	if i.hasEmailAddress(address) {
		return fmt.Errorf("add identity email: %w", ErrEmailAlreadyExists)
	}

	email, err := newEmail(id, address, i.activeEmailCount() == 0, now)
	if err != nil {
		return fmt.Errorf("add identity email: %w", err)
	}
	i.Emails = append(i.Emails, email)
	i.touch(now)
	return nil
}

func (i *Identity) VerifyEmail(id EmailID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("verify identity email: %w", err)
	}
	index := i.emailIndex(id)
	if index < 0 {
		return fmt.Errorf("verify identity email: %w", ErrEmailNotFound)
	}
	if err := i.Emails[index].Verify(now); err != nil {
		return fmt.Errorf("verify identity email: %w", err)
	}

	i.touch(now)
	return nil
}

func (i *Identity) SetPrimaryEmail(id EmailID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("set primary identity email: %w", err)
	}
	index := i.emailIndex(id)
	if index < 0 {
		return fmt.Errorf("set primary identity email: %w", ErrEmailNotFound)
	}
	changed, err := i.Emails[index].Promote(now)
	if err != nil {
		return fmt.Errorf("set primary identity email: %w", err)
	}
	if !changed {
		return nil
	}

	for emailIndex := range i.Emails {
		if emailIndex != index {
			if err := i.Emails[emailIndex].Demote(now); err != nil {
				return fmt.Errorf("set primary identity email: %w", err)
			}
		}
	}
	i.touch(now)
	return nil
}

func (i *Identity) RemoveEmail(id EmailID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("remove identity email: %w", err)
	}
	index := i.emailIndex(id)
	if index < 0 {
		return fmt.Errorf("remove identity email: %w", ErrEmailNotFound)
	}
	if i.Emails[index].IsPrimaryEmail() && i.activeEmailCount() > 1 {
		return errors.New("remove identity email: select another primary email first")
	}
	if err := i.ensureLoginMethodRemovable(index, -1, -1); err != nil {
		return fmt.Errorf("remove identity email: %w", err)
	}

	if err := i.Emails[index].Remove(now); err != nil {
		return fmt.Errorf("remove identity email: %w", err)
	}
	i.touch(now)
	return nil
}

func (i *Identity) AddPhone(id PhoneID, number PhoneNumber, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add identity phone: %w", err)
	}
	if i.hasPhoneNumber(number) {
		return fmt.Errorf("add identity phone: %w", ErrPhoneAlreadyExists)
	}

	phone, err := newPhone(id, number, i.activePhoneCount() == 0, now)
	if err != nil {
		return fmt.Errorf("add identity phone: %w", err)
	}
	i.Phones = append(i.Phones, phone)
	i.touch(now)
	return nil
}

func (i *Identity) VerifyPhone(id PhoneID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("verify identity phone: %w", err)
	}
	index := i.phoneIndex(id)
	if index < 0 {
		return fmt.Errorf("verify identity phone: %w", ErrPhoneNotFound)
	}
	if err := i.Phones[index].Verify(now); err != nil {
		return fmt.Errorf("verify identity phone: %w", err)
	}

	i.touch(now)
	return nil
}

func (i *Identity) SetPrimaryPhone(id PhoneID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("set primary identity phone: %w", err)
	}
	index := i.phoneIndex(id)
	if index < 0 {
		return fmt.Errorf("set primary identity phone: %w", ErrPhoneNotFound)
	}
	changed, err := i.Phones[index].Promote(now)
	if err != nil {
		return fmt.Errorf("set primary identity phone: %w", err)
	}
	if !changed {
		return nil
	}

	for phoneIndex := range i.Phones {
		if phoneIndex != index {
			if err := i.Phones[phoneIndex].Demote(now); err != nil {
				return fmt.Errorf("set primary identity phone: %w", err)
			}
		}
	}
	i.touch(now)
	return nil
}

func (i *Identity) RemovePhone(id PhoneID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("remove identity phone: %w", err)
	}
	index := i.phoneIndex(id)
	if index < 0 {
		return fmt.Errorf("remove identity phone: %w", ErrPhoneNotFound)
	}
	if i.Phones[index].IsPrimaryPhone() && i.activePhoneCount() > 1 {
		return errors.New("remove identity phone: select another primary phone first")
	}
	if err := i.ensureLoginMethodRemovable(-1, index, -1); err != nil {
		return fmt.Errorf("remove identity phone: %w", err)
	}

	if err := i.Phones[index].Remove(now); err != nil {
		return fmt.Errorf("remove identity phone: %w", err)
	}
	i.touch(now)
	return nil
}

func (i *Identity) LinkExternalAccount(
	id ExternalAccountID,
	provider ExternalProvider,
	subject ExternalSubject,
	now time.Time,
) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("link external identity account: %w", err)
	}
	if i.hasExternalAccount(provider, subject) {
		return fmt.Errorf("link external identity account: %w", ErrExternalAccountExists)
	}

	account, err := newExternalAccount(id, provider, subject, now)
	if err != nil {
		return fmt.Errorf("link external identity account: %w", err)
	}
	i.ExternalAccounts = append(i.ExternalAccounts, account)
	i.touch(now)
	return nil
}

func (i *Identity) RecordExternalAuthentication(id ExternalAccountID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("record external identity authentication: %w", err)
	}
	if i.Status != StatusActive {
		return fmt.Errorf("record external identity authentication: %w", ErrInvalidStatus)
	}
	index := i.externalAccountIndex(id)
	if index < 0 {
		return fmt.Errorf("record external identity authentication: %w", ErrExternalAccountNotFound)
	}
	if err := i.ExternalAccounts[index].RecordAuthentication(now); err != nil {
		return fmt.Errorf("record external identity authentication: %w", err)
	}

	i.touch(now)
	return nil
}

func (i *Identity) DisableExternalAccount(id ExternalAccountID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("disable external identity account: %w", err)
	}
	index := i.externalAccountIndex(id)
	if index < 0 {
		return fmt.Errorf("disable external identity account: %w", ErrExternalAccountNotFound)
	}
	if !i.ExternalAccounts[index].IsUsable() {
		return fmt.Errorf("disable external identity account: %w", ErrExternalAccountDisabled)
	}
	if i.Status == StatusActive && !i.hasUsableLoginMethodExcept(-1, -1, index) {
		return fmt.Errorf("disable external identity account: %w", ErrLoginMethodRequired)
	}

	if err := i.ExternalAccounts[index].Disable(now); err != nil {
		return fmt.Errorf("disable external identity account: %w", err)
	}
	i.touch(now)
	return nil
}

func (i *Identity) EnableExternalAccount(id ExternalAccountID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enable external identity account: %w", err)
	}
	index := i.externalAccountIndex(id)
	if index < 0 {
		return fmt.Errorf("enable external identity account: %w", ErrExternalAccountNotFound)
	}
	if err := i.ExternalAccounts[index].Enable(now); err != nil {
		return fmt.Errorf("enable external identity account: %w", err)
	}

	i.touch(now)
	return nil
}

func (i *Identity) UnlinkExternalAccount(id ExternalAccountID, now time.Time) error {
	if err := i.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("unlink external identity account: %w", err)
	}
	index := i.externalAccountIndex(id)
	if index < 0 {
		return fmt.Errorf("unlink external identity account: %w", ErrExternalAccountNotFound)
	}
	if err := i.ensureLoginMethodRemovable(-1, -1, index); err != nil {
		return fmt.Errorf("unlink external identity account: %w", err)
	}

	if err := i.ExternalAccounts[index].Unlink(now); err != nil {
		return fmt.Errorf("unlink external identity account: %w", err)
	}
	i.touch(now)
	return nil
}

func (i *Identity) ensureChangeAllowed(now time.Time) error {
	if i == nil || i.Status == StatusDeleted || i.DeletedAt != nil {
		return fmt.Errorf("validate identity change: %w", ErrInvalidStatus)
	}
	if now.IsZero() || now.Before(i.UpdatedAt) {
		return errors.New("identity change time is invalid")
	}

	return nil
}

func (i *Identity) touch(now time.Time) {
	i.UpdatedAt = now
}

func (i *Identity) ensureLoginMethodRemovable(emailIndex, phoneIndex, accountIndex int) error {
	if !i.hasLoginMethodExcept(emailIndex, phoneIndex, accountIndex) {
		return ErrLoginMethodRequired
	}
	if i.Status == StatusActive && !i.hasUsableLoginMethodExcept(emailIndex, phoneIndex, accountIndex) {
		return ErrLoginMethodRequired
	}

	return nil
}

func (i *Identity) hasLoginMethodExcept(emailIndex, phoneIndex, accountIndex int) bool {
	for index := range i.Emails {
		if index != emailIndex && i.Emails[index].IsActive() {
			return true
		}
	}
	for index := range i.Phones {
		if index != phoneIndex && i.Phones[index].IsActive() {
			return true
		}
	}
	for index := range i.ExternalAccounts {
		if index != accountIndex && i.ExternalAccounts[index].IsActive() {
			return true
		}
	}

	return false
}

func (i *Identity) hasUsableLoginMethodExcept(emailIndex, phoneIndex, accountIndex int) bool {
	for index := range i.Emails {
		if index != emailIndex && i.Emails[index].IsUsable() {
			return true
		}
	}
	for index := range i.Phones {
		if index != phoneIndex && i.Phones[index].IsUsable() {
			return true
		}
	}
	for index := range i.ExternalAccounts {
		if index != accountIndex && i.ExternalAccounts[index].IsUsable() {
			return true
		}
	}

	return false
}

func (i *Identity) hasUsableLoginMethod() bool {
	for index := range i.Emails {
		if i.Emails[index].IsUsable() {
			return true
		}
	}
	for index := range i.Phones {
		if i.Phones[index].IsUsable() {
			return true
		}
	}
	for index := range i.ExternalAccounts {
		if i.ExternalAccounts[index].IsUsable() {
			return true
		}
	}
	return false
}

func (i *Identity) activeEmailCount() int {
	count := 0
	for index := range i.Emails {
		if i.Emails[index].IsActive() {
			count++
		}
	}
	return count
}

func (i *Identity) hasEmailAddress(address EmailAddress) bool {
	for index := range i.Emails {
		if i.Emails[index].HasAddress(address) {
			return true
		}
	}
	return false
}

func (i *Identity) emailIndex(id EmailID) int {
	for index := range i.Emails {
		if i.Emails[index].HasID(id) {
			return index
		}
	}
	return -1
}

func (i *Identity) activePhoneCount() int {
	count := 0
	for index := range i.Phones {
		if i.Phones[index].IsActive() {
			count++
		}
	}
	return count
}

func (i *Identity) hasPhoneNumber(number PhoneNumber) bool {
	for index := range i.Phones {
		if i.Phones[index].HasNumber(number) {
			return true
		}
	}
	return false
}

func (i *Identity) phoneIndex(id PhoneID) int {
	for index := range i.Phones {
		if i.Phones[index].HasID(id) {
			return index
		}
	}
	return -1
}

func (i *Identity) hasExternalAccount(provider ExternalProvider, subject ExternalSubject) bool {
	for index := range i.ExternalAccounts {
		if i.ExternalAccounts[index].Matches(provider, subject) {
			return true
		}
	}
	return false
}

func (i *Identity) externalAccountIndex(id ExternalAccountID) int {
	for index := range i.ExternalAccounts {
		if i.ExternalAccounts[index].HasID(id) {
			return index
		}
	}
	return -1
}
