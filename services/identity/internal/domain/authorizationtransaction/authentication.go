package authorizationtransaction

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (t *AuthorizationTransaction) CompletePrimaryAuthentication(
	identityID identitydomain.ID,
	method authenticationdomain.Method,
	nextStep NextStep,
	now time.Time,
) error {
	if err := t.applyAuthentication(
		NextStepChooseAuthenticationMethod,
		identityID,
		method,
		nextStep,
		now,
	); err != nil {
		return fmt.Errorf("complete primary authentication in authorization transaction: %w", err)
	}
	return nil
}

func (t *AuthorizationTransaction) CompleteMFA(
	identityID identitydomain.ID,
	method authenticationdomain.Method,
	now time.Time,
) error {
	if err := t.applyAuthentication(
		NextStepMFA,
		identityID,
		method,
		NextStepConsent,
		now,
	); err != nil {
		return fmt.Errorf("complete MFA in authorization transaction: %w", err)
	}
	return nil
}

func (t *AuthorizationTransaction) applyAuthentication(
	currentStep NextStep,
	identityID identitydomain.ID,
	method authenticationdomain.Method,
	nextStep NextStep,
	now time.Time,
) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return err
	}
	if t.Status != StatusPending {
		return ErrInvalidStatus
	}
	if t.NextStep != currentStep || t.AuthenticationLevel != nil {
		return ErrInvalidStep
	}
	if !identityID.IsValid() {
		return errors.New("identity identifier is invalid")
	}
	if !method.IsValid() {
		return errors.New("authentication method is invalid")
	}
	if t.IdentityID != nil && *t.IdentityID != identityID {
		return ErrIdentityMismatch
	}
	if slices.Contains(t.AuthenticatedMethods, method) {
		return errors.New("authentication method is duplicated")
	}

	methods := append(slices.Clone(t.AuthenticatedMethods), method)
	level, err := resolveAuthenticationCompletion(methods, nextStep)
	if err != nil {
		return err
	}
	if err := t.transitionTo(nextStep); err != nil {
		return err
	}

	t.IdentityID = domain.ClonePointer(&identityID)
	t.AuthenticatedMethods = methods
	t.completeAuthentication(level, now)
	return nil
}

func (t *AuthorizationTransaction) CompletePasswordChange(nextStep NextStep, now time.Time) error {
	if err := t.completeRequiredStep(NextStepPasswordChange, nextStep, now); err != nil {
		return fmt.Errorf("complete authorization transaction password change: %w", err)
	}
	return nil
}

func (t *AuthorizationTransaction) CompleteEmailVerification(nextStep NextStep, now time.Time) error {
	if t == nil || t.NextStep != NextStepEmailVerification || !t.NextStep.CanTransitionTo(nextStep) {
		return fmt.Errorf("complete authorization transaction email verification: %w", ErrInvalidStep)
	}
	if err := t.ensureVerificationReference(VerificationChannelEmail, now); err != nil {
		return fmt.Errorf("complete authorization transaction email verification: %w", err)
	}
	if err := t.completeRequiredStep(NextStepEmailVerification, nextStep, now); err != nil {
		return fmt.Errorf("complete authorization transaction email verification: %w", err)
	}
	t.Verification = nil
	return nil
}

func (t *AuthorizationTransaction) CompletePhoneVerification(nextStep NextStep, now time.Time) error {
	if t == nil || t.NextStep != NextStepPhoneVerification || !t.NextStep.CanTransitionTo(nextStep) {
		return fmt.Errorf("complete authorization transaction phone verification: %w", ErrInvalidStep)
	}
	if err := t.ensureVerificationReference(VerificationChannelPhone, now); err != nil {
		return fmt.Errorf("complete authorization transaction phone verification: %w", err)
	}
	if err := t.completeRequiredStep(NextStepPhoneVerification, nextStep, now); err != nil {
		return fmt.Errorf("complete authorization transaction phone verification: %w", err)
	}
	t.Verification = nil
	return nil
}

func (t *AuthorizationTransaction) BindEmailVerification(
	emailID identitydomain.EmailID,
	challengeID uuid.UUID,
	expiresAt time.Time,
	now time.Time,
) error {
	return t.bindVerification(
		NextStepEmailVerification,
		VerificationChannelEmail,
		emailID.UUID(),
		challengeID,
		expiresAt,
		now,
	)
}

func (t *AuthorizationTransaction) BindPhoneVerification(
	phoneID identitydomain.PhoneID,
	challengeID uuid.UUID,
	expiresAt time.Time,
	now time.Time,
) error {
	return t.bindVerification(
		NextStepPhoneVerification,
		VerificationChannelPhone,
		phoneID.UUID(),
		challengeID,
		expiresAt,
		now,
	)
}

func (t *AuthorizationTransaction) bindVerification(
	step NextStep,
	channel VerificationChannel,
	contactID uuid.UUID,
	challengeID uuid.UUID,
	expiresAt time.Time,
	now time.Time,
) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("bind verification to authorization transaction: %w", err)
	}
	if t.Status != StatusPending || t.NextStep != step {
		return fmt.Errorf("bind verification to authorization transaction: %w", ErrInvalidStep)
	}
	if t.Verification != nil {
		return errors.New("bind verification to authorization transaction: verification is already bound")
	}
	if !expiresAt.After(now) {
		return errors.New("bind verification to authorization transaction: challenge expiration is invalid")
	}

	reference, err := NewVerificationReference(channel, contactID, challengeID, expiresAt)
	if err != nil {
		return fmt.Errorf("bind verification to authorization transaction: %w", err)
	}
	t.Verification = domain.ClonePointer(&reference)
	return nil
}

func (t *AuthorizationTransaction) ensureVerificationReference(channel VerificationChannel, now time.Time) error {
	if t == nil || t.Verification == nil {
		return errors.New("verification reference is missing")
	}
	if t.Verification.Channel != channel {
		return errors.New("verification reference channel does not match current step")
	}
	if !now.Before(t.Verification.ExpiresAt) {
		return errors.New("verification challenge is expired")
	}
	return nil
}

func (t *AuthorizationTransaction) completeRequiredStep(
	currentStep NextStep,
	nextStep NextStep,
	now time.Time,
) error {
	if err := t.ensureChangeAllowed(now); err != nil {
		return err
	}
	if t.Status != StatusPending {
		return ErrInvalidStatus
	}
	if t.NextStep != currentStep || t.AuthenticationLevel != nil || t.IdentityID == nil {
		return ErrInvalidStep
	}

	level, err := resolveAuthenticationCompletion(t.AuthenticatedMethods, nextStep)
	if err != nil {
		return err
	}
	if err := t.transitionTo(nextStep); err != nil {
		return err
	}

	t.completeAuthentication(level, now)
	return nil
}

func resolveAuthenticationCompletion(
	methods []authenticationdomain.Method,
	nextStep NextStep,
) (*authenticationdomain.Level, error) {
	if nextStep != NextStepConsent {
		return nil, nil
	}

	level, err := authenticationdomain.ResolveLevel(methods)
	if err != nil {
		return nil, err
	}
	return domain.ClonePointer(&level), nil
}

func (t *AuthorizationTransaction) completeAuthentication(level *authenticationdomain.Level, now time.Time) {
	if level == nil {
		return
	}

	t.AuthenticationLevel = domain.ClonePointer(level)
	t.AuthenticatedAt = domain.ClonePointer(&now)
}
