package authentication

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

const (
	registrationPasswordMinBytes = 8
	registrationPasswordMaxBytes = 1024
	registrationNameMaxRunes     = 255
)

type RegistrationProfileInput struct {
	FamilyName string
	GivenName  string
	MiddleName *string
}

type RegisterIdentityInput struct {
	AuthorizationTransactionID uuid.UUID
	Email                      *string
	PhoneNumber                *string
	Password                   string
	Profile                    *RegistrationProfileInput
	RequestedFromIP            *sessiondomain.IPAddress
}

type RegisterIdentityOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
}

type cmdRegisterIdentity struct {
	transactionID   transactiondomain.ID
	email           *identitydomain.EmailAddress
	phone           *identitydomain.PhoneNumber
	password        string
	profile         RegistrationProfileInput
	requestedFromIP *sessiondomain.IPAddress
}

func (s *Service) RegisterIdentity(ctx context.Context, input RegisterIdentityInput) (RegisterIdentityOutput, error) {
	command, err := prepareRegisterIdentity(input)
	if err != nil {
		return RegisterIdentityOutput{}, apperr.New(apperr.TypeInvalidArgument, "registration_invalid_input", "registration request is invalid", err)
	}

	now := s.clock.Now()
	identity, verificationStep, err := s.newRegistrationIdentity(command, now)
	if err != nil {
		return RegisterIdentityOutput{}, apperr.New(apperr.TypeInvalidArgument, "registration_invalid_input", "registration request is invalid", err)
	}

	profile, err := identityprofiledomain.New(
		identity.ID,
		command.profile.FamilyName,
		command.profile.GivenName,
		command.profile.MiddleName,
		now,
	)
	if err != nil {
		return RegisterIdentityOutput{}, apperr.New(apperr.TypeInvalidArgument, "registration_invalid_input", "registration profile is invalid", err)
	}

	encodedPassword, err := s.passwordHasher.Hash(command.password)
	if err != nil {
		return RegisterIdentityOutput{}, fmt.Errorf("hash registration password: %w", err)
	}
	passwordHash, err := passworddomain.NewHash(encodedPassword)
	if err != nil {
		return RegisterIdentityOutput{}, fmt.Errorf("create registration password hash: %w", err)
	}
	password, err := passworddomain.New(identity.ID, passwordHash, false, nil, now)
	if err != nil {
		return RegisterIdentityOutput{}, fmt.Errorf("create registration password: %w", err)
	}

	var output RegisterIdentityOutput
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, command.transactionID)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}
		if err := transaction.BeginPasswordRegistration(identity.ID, verificationStep, now); err != nil {
			return apperr.New(apperr.TypeConflict, "registration_invalid_state", "registration is not available for this authorization transaction", err)
		}

		if err := s.identities.Create(txCtx, identity); err != nil {
			if errors.Is(err, identitydomain.ErrEmailAlreadyExists) || errors.Is(err, identitydomain.ErrPhoneAlreadyExists) {
				return apperr.New(apperr.TypeConflict, "registration_contact_already_exists", "registration contact already exists", err)
			}
			return fmt.Errorf("create registration identity: %w", err)
		}
		if err := s.identityProfiles.Create(txCtx, profile); err != nil {
			return fmt.Errorf("create registration identity profile: %w", err)
		}
		profileEvent, err := s.events.Create(
			identityprofiledomain.EventTypeUpdated,
			identityprofiledomain.AggregateType,
			identityprofiledomain.AggregateID(profile.IdentityID.UUID()),
			&profile.Version,
			profile.UpdatedEvent(),
			profile.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("create registration identity profile event: %w", err)
		}
		if err := s.outbox.Create(txCtx, profileEvent); err != nil {
			return fmt.Errorf("save registration identity profile event: %w", err)
		}
		if err := s.passwords.Create(txCtx, password); err != nil {
			return fmt.Errorf("create registration password: %w", err)
		}

		switch verificationStep {
		case transactiondomain.NextStepEmailVerification:
			if err := s.contactVerifications.StartEmail(txCtx, transaction, identity.Emails[0], command.requestedFromIP, now); err != nil {
				return fmt.Errorf("start registration email verification: %w", err)
			}
		case transactiondomain.NextStepPhoneVerification:
			if err := s.contactVerifications.StartPhone(txCtx, transaction, identity.Phones[0], command.requestedFromIP, now); err != nil {
				return fmt.Errorf("start registration phone verification: %w", err)
			}
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction: %w", err)
		}

		output = RegisterIdentityOutput{AuthorizationTransaction: transaction}
		return nil
	})
	if err != nil {
		return RegisterIdentityOutput{}, err
	}

	return output, nil
}

func prepareRegisterIdentity(input RegisterIdentityInput) (cmdRegisterIdentity, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return cmdRegisterIdentity{}, errors.New("authorization transaction identifier is required")
	}
	if (input.Email == nil) == (input.PhoneNumber == nil) {
		return cmdRegisterIdentity{}, errors.New("exactly one registration contact is required")
	}
	if len(input.Password) < registrationPasswordMinBytes || len(input.Password) > registrationPasswordMaxBytes {
		return cmdRegisterIdentity{}, errors.New("password length is invalid")
	}
	if input.Profile == nil {
		return cmdRegisterIdentity{}, errors.New("registration profile is required")
	}
	if utf8.RuneCountInString(input.Profile.FamilyName) > registrationNameMaxRunes ||
		utf8.RuneCountInString(input.Profile.GivenName) > registrationNameMaxRunes ||
		(input.Profile.MiddleName != nil && utf8.RuneCountInString(*input.Profile.MiddleName) > registrationNameMaxRunes) {
		return cmdRegisterIdentity{}, errors.New("registration profile name is too long")
	}

	command := cmdRegisterIdentity{
		transactionID:   transactiondomain.ID(input.AuthorizationTransactionID),
		password:        input.Password,
		profile:         *input.Profile,
		requestedFromIP: input.RequestedFromIP,
	}
	if input.Email != nil {
		email, err := identitydomain.NewEmailAddress(*input.Email)
		if err != nil {
			return cmdRegisterIdentity{}, fmt.Errorf("create registration email: %w", err)
		}
		command.email = &email
		return command, nil
	}

	phone, err := identitydomain.NewPhoneNumber(*input.PhoneNumber)
	if err != nil {
		return cmdRegisterIdentity{}, fmt.Errorf("create registration phone: %w", err)
	}
	command.phone = &phone
	return command, nil
}

func (s *Service) newRegistrationIdentity(
	command cmdRegisterIdentity,
	now time.Time,
) (*identitydomain.Identity, transactiondomain.NextStep, error) {
	identityID := identitydomain.ID(s.ids.New())
	if command.email != nil {
		identity, err := identitydomain.NewWithEmail(
			identityID,
			identitydomain.EmailID(s.ids.New()),
			*command.email,
			now,
		)
		if err != nil {
			return nil, "", err
		}
		return identity, transactiondomain.NextStepEmailVerification, nil
	}
	if command.phone != nil {
		identity, err := identitydomain.NewWithPhone(
			identityID,
			identitydomain.PhoneID(s.ids.New()),
			*command.phone,
			now,
		)
		if err != nil {
			return nil, "", err
		}
		return identity, transactiondomain.NextStepPhoneVerification, nil
	}

	return nil, "", errors.New("registration contact is missing")
}
