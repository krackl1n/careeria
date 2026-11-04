package authentication

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	flowdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authenticationflow"
	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
)

type AuthenticateWithPasswordInput struct {
	AuthorizationTransactionID uuid.UUID
	Email                      *string
	PhoneNumber                *string
	Password                   string
}

type AuthenticateWithPasswordOutput struct {
	AuthorizationTransaction *transactiondomain.AuthorizationTransaction
}

type cmdAuthenticateWithPassword struct {
	transactionID transactiondomain.ID
	email         *identitydomain.EmailAddress
	phone         *identitydomain.PhoneNumber
	password      string
}

type contactVerificationTarget struct {
	email *identitydomain.Email
	phone *identitydomain.Phone
}

func (s *Service) AuthenticateWithPassword(ctx context.Context, input AuthenticateWithPasswordInput) (AuthenticateWithPasswordOutput, error) {
	command, err := prepareAuthenticateWithPassword(input)
	if err != nil {
		return AuthenticateWithPasswordOutput{}, apperr.New(apperr.TypeInvalidArgument, "authentication_invalid_input", "password authentication request is invalid", err)
	}

	now := s.clock.Now()
	var output AuthenticateWithPasswordOutput
	var authenticationErr error
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		transaction, err := s.authorizationTransactions.Get(txCtx, command.transactionID)
		if err != nil {
			if errors.Is(err, transactiondomain.ErrNotFound) {
				return apperr.New(apperr.TypeNotFound, "authorization_transaction_not_found", "authorization transaction not found", err)
			}
			return fmt.Errorf("get authorization transaction: %w", err)
		}

		identity, err := s.findIdentity(txCtx, command)
		if err != nil {
			if errors.Is(err, identitydomain.ErrNotFound) {
				authenticationErr = invalidCredentials(err)
				return nil
			}
			return err
		}
		if identity.Status != identitydomain.StatusActive {
			authenticationErr = invalidCredentials(err)
			return nil
		}

		password, err := s.passwords.Get(txCtx, identity.ID)
		if err != nil {
			if errors.Is(err, passworddomain.ErrNotFound) {
				authenticationErr = invalidCredentials(err)
				return nil
			}
			return fmt.Errorf("get password: %w", err)
		}
		if err := s.passwordVerifier.Verify(command.password, password.Credential.Hash); err != nil {
			if _, recordErr := password.RecordFailedAttempt(s.passwordFailurePolicy, now); recordErr == nil {
				if updateErr := s.passwords.Update(txCtx, password); updateErr != nil {
					return fmt.Errorf("record failed password authentication: %w", updateErr)
				}
			}
			authenticationErr = invalidCredentials(err)
			return nil
		}
		if err := password.RecordSuccessfulAuthentication(now); err != nil {
			authenticationErr = invalidCredentials(err)
			return nil
		}

		contactVerification, verificationTarget, err := resolveContactVerification(identity, command)
		if err != nil {
			return fmt.Errorf("resolve authenticated contact verification: %w", err)
		}
		decision, err := s.resolveNextStep(
			txCtx,
			transaction,
			identity.ID,
			flowdomain.Requirements{
				PasswordChangeRequired: password.Credential.MustChange,
				ContactVerification:    contactVerification,
			},
			now,
		)
		if err != nil {
			return fmt.Errorf("resolve authentication flow next step: %w", err)
		}

		if err := transaction.CompletePrimaryAuthentication(
			identity.ID,
			authenticationdomain.MethodPassword,
			decision.NextStep,
			now,
		); err != nil {
			return apperr.New(apperr.TypeConflict, "authentication_invalid_state", "password authentication is not available for this authorization transaction", err)
		}

		switch decision.NextStep {
		case transactiondomain.NextStepEmailVerification:
			if verificationTarget.email == nil {
				return errors.New("start authenticated email verification: target is missing")
			}
			if err := s.contactVerifications.StartEmail(txCtx, transaction, *verificationTarget.email, nil, now); err != nil {
				return fmt.Errorf("start authenticated email verification: %w", err)
			}
		case transactiondomain.NextStepPhoneVerification:
			if verificationTarget.phone == nil {
				return errors.New("start authenticated phone verification: target is missing")
			}
			if err := s.contactVerifications.StartPhone(txCtx, transaction, *verificationTarget.phone, nil, now); err != nil {
				return fmt.Errorf("start authenticated phone verification: %w", err)
			}
		}
		if err := s.passwords.Update(txCtx, password); err != nil {
			return fmt.Errorf("record successful password authentication: %w", err)
		}
		if err := s.authorizationTransactions.Update(txCtx, transaction); err != nil {
			return fmt.Errorf("update authorization transaction: %w", err)
		}

		output = AuthenticateWithPasswordOutput{AuthorizationTransaction: transaction}
		return nil
	})
	if err != nil {
		return AuthenticateWithPasswordOutput{}, err
	}
	if authenticationErr != nil {
		return AuthenticateWithPasswordOutput{}, authenticationErr
	}
	return output, nil
}

func prepareAuthenticateWithPassword(input AuthenticateWithPasswordInput) (cmdAuthenticateWithPassword, error) {
	if input.AuthorizationTransactionID == uuid.Nil {
		return cmdAuthenticateWithPassword{}, errors.New("authorization transaction identifier is required")
	}
	if input.Password == "" {
		return cmdAuthenticateWithPassword{}, errors.New("password is required")
	}
	if (input.Email == nil) == (input.PhoneNumber == nil) {
		return cmdAuthenticateWithPassword{}, errors.New("exactly one login identifier is required")
	}

	command := cmdAuthenticateWithPassword{transactionID: transactiondomain.ID(input.AuthorizationTransactionID), password: input.Password}
	if input.Email != nil {
		email, err := identitydomain.NewEmailAddress(*input.Email)
		if err != nil {
			return cmdAuthenticateWithPassword{}, fmt.Errorf("create email login identifier: %w", err)
		}
		command.email = &email
		return command, nil
	}

	phone, err := identitydomain.NewPhoneNumber(*input.PhoneNumber)
	if err != nil {
		return cmdAuthenticateWithPassword{}, fmt.Errorf("create phone login identifier: %w", err)
	}
	command.phone = &phone
	return command, nil
}

func (s *Service) findIdentity(ctx context.Context, command cmdAuthenticateWithPassword) (*identitydomain.Identity, error) {
	if command.email != nil {
		identity, err := s.identities.GetByEmail(ctx, *command.email)
		if err != nil {
			if errors.Is(err, identitydomain.ErrNotFound) {
				return nil, err
			}
			return nil, fmt.Errorf("get identity by email: %w", err)
		}
		return identity, nil
	}

	identity, err := s.identities.GetByPhone(ctx, *command.phone)
	if err != nil {
		if errors.Is(err, identitydomain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get identity by phone: %w", err)
	}
	return identity, nil
}

func invalidCredentials(cause error) error {
	return apperr.New(apperr.TypeUnauthorized, "authentication_invalid_credentials", "invalid credentials", cause)
}

func resolveContactVerification(
	identity *identitydomain.Identity,
	command cmdAuthenticateWithPassword,
) (flowdomain.ContactVerificationRequirement, contactVerificationTarget, error) {
	if identity == nil {
		return flowdomain.ContactVerificationNone, contactVerificationTarget{}, errors.New("identity is nil")
	}
	if command.email != nil {
		for index := range identity.Emails {
			email := &identity.Emails[index]
			if email.IsActive() && email.HasAddress(*command.email) {
				if email.VerifiedAt == nil {
					return flowdomain.ContactVerificationEmail, contactVerificationTarget{email: email}, nil
				}
				return flowdomain.ContactVerificationNone, contactVerificationTarget{}, nil
			}
		}
		return flowdomain.ContactVerificationNone, contactVerificationTarget{}, errors.New("authenticated email is missing from identity")
	}
	if command.phone != nil {
		for index := range identity.Phones {
			phone := &identity.Phones[index]
			if phone.IsActive() && phone.HasNumber(*command.phone) {
				if phone.VerifiedAt == nil {
					return flowdomain.ContactVerificationPhone, contactVerificationTarget{phone: phone}, nil
				}
				return flowdomain.ContactVerificationNone, contactVerificationTarget{}, nil
			}
		}
		return flowdomain.ContactVerificationNone, contactVerificationTarget{}, errors.New("authenticated phone is missing from identity")
	}
	return flowdomain.ContactVerificationNone, contactVerificationTarget{}, errors.New("authenticated contact is missing")
}
