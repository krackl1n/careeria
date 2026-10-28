package authenticationflow

import (
	"errors"
	"slices"

	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type ContactVerificationRequirement string

const (
	ContactVerificationNone  ContactVerificationRequirement = ""
	ContactVerificationEmail ContactVerificationRequirement = "email"
	ContactVerificationPhone ContactVerificationRequirement = "phone"
)

func (r ContactVerificationRequirement) IsValid() bool {
	switch r {
	case ContactVerificationNone, ContactVerificationEmail, ContactVerificationPhone:
		return true
	default:
		return false
	}
}

type Requirements struct {
	PasswordChangeRequired bool
	ContactVerification    ContactVerificationRequirement
}

func (r Requirements) Validate() error {
	if !r.ContactVerification.IsValid() {
		return errors.New("authentication flow contact verification requirement is invalid")
	}
	return nil
}

type Facts struct {
	availableMFAMethods   []authenticationdomain.Method
	consentMode           oauthclientdomain.ConsentMode
	rememberedGrantCovers bool
}

func NewFacts(
	availableMFAMethods []authenticationdomain.Method,
	consentMode oauthclientdomain.ConsentMode,
	rememberedGrantCovers bool,
) (Facts, error) {
	if _, err := oauthclientdomain.ParseConsentMode(string(consentMode)); err != nil {
		return Facts{}, errors.New("create authentication flow facts: consent mode is invalid")
	}

	seen := make(map[authenticationdomain.Method]struct{}, len(availableMFAMethods))
	for _, method := range availableMFAMethods {
		if !method.IsValid() || method == authenticationdomain.MethodPassword {
			return Facts{}, errors.New("create authentication flow facts: MFA method is invalid")
		}
		if _, exists := seen[method]; exists {
			return Facts{}, errors.New("create authentication flow facts: MFA method is duplicated")
		}
		seen[method] = struct{}{}
	}
	if rememberedGrantCovers && consentMode != oauthclientdomain.ConsentModeRemember {
		return Facts{}, errors.New("create authentication flow facts: remembered grant is inconsistent with consent mode")
	}

	return Facts{
		availableMFAMethods:   slices.Clone(availableMFAMethods),
		consentMode:           consentMode,
		rememberedGrantCovers: rememberedGrantCovers,
	}, nil
}

func (f Facts) HasUsableMFA() bool {
	return len(f.availableMFAMethods) > 0
}

func (f Facts) AvailableMFAMethods() []authenticationdomain.Method {
	return slices.Clone(f.availableMFAMethods)
}

func (f Facts) ConsentMode() oauthclientdomain.ConsentMode {
	return f.consentMode
}

func (f Facts) RememberedGrantCovers() bool {
	return f.rememberedGrantCovers
}
