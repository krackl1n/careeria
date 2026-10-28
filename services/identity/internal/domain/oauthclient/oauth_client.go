package oauthclient

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
)

type OAuthClient struct {
	ID       ID
	ClientID ClientID

	ClientType              ClientType
	TokenEndpointAuthMethod TokenEndpointAuthMethod

	Name        string
	Description *string

	RequirePKCE bool
	ConsentMode ConsentMode
	Status      Status

	CreatedByIdentityID *identitydomain.ID

	RedirectURIs           []RedirectURI
	PostLogoutRedirectURIs []PostLogoutRedirectURI
	GrantTypes             []GrantTypeRegistration
	Scopes                 []ScopeRegistration
	Credentials            []Credential

	CreatedAt  time.Time
	UpdatedAt  time.Time
	DisabledAt *time.Time
	DeletedAt  *time.Time
}

func NewPublic(
	id ID,
	clientID ClientID,
	name string,
	description *string,
	createdByIdentityID *identitydomain.ID,
	redirectURIID RedirectURIID,
	redirectURI RedirectURIValue,
	now time.Time,
) (*OAuthClient, error) {
	return newOAuthClient(
		id,
		clientID,
		ClientTypePublic,
		name,
		description,
		createdByIdentityID,
		redirectURIID,
		redirectURI,
		now,
	)
}

func NewConfidential(
	id ID,
	clientID ClientID,
	name string,
	description *string,
	createdByIdentityID *identitydomain.ID,
	redirectURIID RedirectURIID,
	redirectURI RedirectURIValue,
	credentialID CredentialID,
	secretHash SecretHash,
	credentialName *string,
	credentialExpiresAt *time.Time,
	now time.Time,
) (*OAuthClient, error) {
	client, err := newOAuthClient(
		id,
		clientID,
		ClientTypeConfidential,
		name,
		description,
		createdByIdentityID,
		redirectURIID,
		redirectURI,
		now,
	)
	if err != nil {
		return nil, err
	}

	credential, err := newCredential(
		credentialID,
		secretHash,
		credentialName,
		createdByIdentityID,
		credentialExpiresAt,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create OAuth client: %w", err)
	}

	client.Credentials = append(client.Credentials, credential)
	return client, nil
}

func newOAuthClient(
	id ID,
	clientID ClientID,
	clientType ClientType,
	name string,
	description *string,
	createdByIdentityID *identitydomain.ID,
	redirectURIID RedirectURIID,
	redirectURI RedirectURIValue,
	now time.Time,
) (*OAuthClient, error) {
	normalizedName, ok := domain.NormalizeRequiredText(name)
	if !ok {
		return nil, errors.New("create OAuth client: name is required")
	}
	if !id.IsValid() {
		return nil, errors.New("create OAuth client: identifier is required")
	}
	if createdByIdentityID != nil && !createdByIdentityID.IsValid() {
		return nil, errors.New("create OAuth client: creator identifier is invalid")
	}
	if now.IsZero() {
		return nil, errors.New("create OAuth client: time is invalid")
	}
	initialRedirectURI, err := newRedirectURI(redirectURIID, redirectURI, now)
	if err != nil {
		return nil, fmt.Errorf("create OAuth client: redirect URI %w", err)
	}

	client := &OAuthClient{
		ID:                      id,
		ClientID:                clientID,
		ClientType:              clientType,
		TokenEndpointAuthMethod: authMethodFor(clientType),
		Name:                    normalizedName,
		Description:             domain.NormalizeOptionalText(description),
		RequirePKCE:             true,
		ConsentMode:             ConsentModeRemember,
		Status:                  StatusActive,
		CreatedByIdentityID:     domain.ClonePointer(createdByIdentityID),
		RedirectURIs:            []RedirectURI{initialRedirectURI},
		GrantTypes:              []GrantTypeRegistration{newGrantTypeRegistration(GrantTypeAuthorizationCode, now)},
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	return client, nil
}

func (c *OAuthClient) SetConsentMode(mode ConsentMode, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("change OAuth client consent mode: %w", err)
	}
	if _, err := ParseConsentMode(string(mode)); err != nil {
		return fmt.Errorf("change OAuth client consent mode: %w", err)
	}
	if c.ConsentMode == mode {
		return nil
	}

	c.ConsentMode = mode
	c.touch(now)
	return nil
}

func (c *OAuthClient) UpdateMetadata(name string, description *string, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("update OAuth client metadata: %w", err)
	}
	normalizedName, ok := domain.NormalizeRequiredText(name)
	if !ok {
		return errors.New("update OAuth client metadata: name is required")
	}
	normalizedDescription := domain.NormalizeOptionalText(description)
	if c.Name == normalizedName && domain.EqualOptionalText(c.Description, normalizedDescription) {
		return nil
	}

	c.Name = normalizedName
	c.Description = normalizedDescription
	c.touch(now)
	return nil
}

func (c *OAuthClient) SetPKCERequired(required bool, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("change OAuth client PKCE requirement: %w", err)
	}
	if c.ClientType == ClientTypePublic && !required {
		return fmt.Errorf("change OAuth client PKCE requirement: %w", ErrPKCERequired)
	}
	if c.RequirePKCE == required {
		return nil
	}

	c.RequirePKCE = required
	c.touch(now)
	return nil
}

func (c *OAuthClient) AddCredential(
	id CredentialID,
	secretHash SecretHash,
	name *string,
	createdByIdentityID *identitydomain.ID,
	expiresAt *time.Time,
	now time.Time,
) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add OAuth client credential: %w", err)
	}
	if c.ClientType != ClientTypeConfidential {
		return fmt.Errorf("add OAuth client credential: %w", ErrCredentialsNotAllowed)
	}

	credential, err := newCredential(
		id,
		secretHash,
		name,
		createdByIdentityID,
		expiresAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("add OAuth client credential: %w", err)
	}
	if c.credentialIndex(credential.ID) >= 0 {
		return errors.New("add OAuth client credential: identifier already exists")
	}

	c.Credentials = append(c.Credentials, credential)
	c.touch(now)
	return nil
}

func (c *OAuthClient) RenameCredential(id CredentialID, name *string, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("rename OAuth client credential: %w", err)
	}
	index := c.credentialIndex(id)
	if index < 0 {
		return fmt.Errorf("rename OAuth client credential: %w", ErrCredentialNotFound)
	}
	changed, err := c.Credentials[index].Rename(name, now)
	if err != nil {
		return fmt.Errorf("rename OAuth client credential: %w", err)
	}
	if !changed {
		return nil
	}

	c.touch(now)
	return nil
}

func (c *OAuthClient) RevokeCredential(id CredentialID, reason string, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("revoke OAuth client credential: %w", err)
	}
	index := c.credentialIndex(id)
	if index < 0 {
		return fmt.Errorf("revoke OAuth client credential: %w", ErrCredentialNotFound)
	}
	if err := c.Credentials[index].Revoke(now, reason); err != nil {
		return fmt.Errorf("revoke OAuth client credential: %w", err)
	}

	c.touch(now)
	return nil
}

func (c *OAuthClient) ValidateCredential(id CredentialID, now time.Time) error {
	if err := c.ensureActive(now); err != nil {
		return fmt.Errorf("validate OAuth client credential: %w", err)
	}
	if c.ClientType != ClientTypeConfidential {
		return fmt.Errorf("validate OAuth client credential: %w", ErrCredentialsNotAllowed)
	}
	index := c.credentialIndex(id)
	if index < 0 {
		return fmt.Errorf("validate OAuth client credential: %w", ErrCredentialNotFound)
	}
	if err := c.Credentials[index].Validate(now); err != nil {
		return fmt.Errorf("validate OAuth client credential: %w", err)
	}

	return nil
}

func (c *OAuthClient) RecordCredentialUsed(id CredentialID, now time.Time) error {
	if err := c.ValidateCredential(id, now); err != nil {
		return fmt.Errorf("record OAuth client credential use: %w", err)
	}

	index := c.credentialIndex(id)
	if err := c.Credentials[index].RecordUsed(now); err != nil {
		return fmt.Errorf("record OAuth client credential use: %w", err)
	}
	c.touch(now)
	return nil
}

func (c *OAuthClient) AddRedirectURI(id RedirectURIID, uri RedirectURIValue, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add OAuth client redirect URI: %w", err)
	}
	if c.redirectURIIndexByValue(uri) >= 0 {
		return fmt.Errorf("add OAuth client redirect URI: %w", ErrRedirectURIAlreadyExists)
	}
	if c.redirectURIIndex(id) >= 0 {
		return errors.New("add OAuth client redirect URI: identifier already exists")
	}
	redirectURI, err := newRedirectURI(id, uri, now)
	if err != nil {
		return fmt.Errorf("add OAuth client redirect URI: %w", err)
	}

	c.RedirectURIs = append(c.RedirectURIs, redirectURI)
	c.touch(now)
	return nil
}

func (c *OAuthClient) RemoveRedirectURI(id RedirectURIID, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("remove OAuth client redirect URI: %w", err)
	}
	index := c.redirectURIIndex(id)
	if index < 0 {
		return fmt.Errorf("remove OAuth client redirect URI: %w", ErrRedirectURINotFound)
	}
	if len(c.RedirectURIs) == 1 {
		return fmt.Errorf("remove OAuth client redirect URI: %w", ErrLastRedirectURI)
	}

	c.RedirectURIs = append(c.RedirectURIs[:index], c.RedirectURIs[index+1:]...)
	c.touch(now)
	return nil
}

func (c *OAuthClient) AddPostLogoutRedirectURI(id PostLogoutRedirectURIID, uri PostLogoutRedirectURIValue, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add OAuth client post-logout redirect URI: %w", err)
	}
	if c.postLogoutRedirectURIIndexByValue(uri) >= 0 {
		return fmt.Errorf("add OAuth client post-logout redirect URI: %w", ErrPostLogoutRedirectURIAlreadyExists)
	}
	if c.postLogoutRedirectURIIndex(id) >= 0 {
		return errors.New("add OAuth client post-logout redirect URI: identifier already exists")
	}
	redirectURI, err := newPostLogoutRedirectURI(id, uri, now)
	if err != nil {
		return fmt.Errorf("add OAuth client post-logout redirect URI: %w", err)
	}

	c.PostLogoutRedirectURIs = append(c.PostLogoutRedirectURIs, redirectURI)
	c.touch(now)
	return nil
}

func (c *OAuthClient) RemovePostLogoutRedirectURI(id PostLogoutRedirectURIID, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("remove OAuth client post-logout redirect URI: %w", err)
	}
	index := c.postLogoutRedirectURIIndex(id)
	if index < 0 {
		return fmt.Errorf("remove OAuth client post-logout redirect URI: %w", ErrPostLogoutRedirectURINotFound)
	}

	c.PostLogoutRedirectURIs = append(c.PostLogoutRedirectURIs[:index], c.PostLogoutRedirectURIs[index+1:]...)
	c.touch(now)
	return nil
}

func (c *OAuthClient) AddGrantType(grantType GrantType, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add OAuth client grant type: %w", err)
	}
	if c.grantTypeIndex(grantType) >= 0 {
		return fmt.Errorf("add OAuth client grant type: %w", ErrGrantTypeAlreadyExists)
	}

	c.GrantTypes = append(c.GrantTypes, newGrantTypeRegistration(grantType, now))
	c.touch(now)
	return nil
}

func (c *OAuthClient) RemoveGrantType(grantType GrantType, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("remove OAuth client grant type: %w", err)
	}
	index := c.grantTypeIndex(grantType)
	if index < 0 {
		return fmt.Errorf("remove OAuth client grant type: %w", ErrGrantTypeNotFound)
	}
	if grantType == GrantTypeAuthorizationCode {
		return fmt.Errorf("remove OAuth client grant type: %w", ErrAuthorizationCodeRequired)
	}

	c.GrantTypes = append(c.GrantTypes[:index], c.GrantTypes[index+1:]...)
	c.touch(now)
	return nil
}

func (c *OAuthClient) AddScope(scope oauthdomain.Scope, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("add OAuth client scope: %w", err)
	}
	if c.scopeIndex(scope) >= 0 {
		return fmt.Errorf("add OAuth client scope: %w", ErrScopeAlreadyExists)
	}

	c.Scopes = append(c.Scopes, newScopeRegistration(scope, now))
	c.touch(now)
	return nil
}

func (c *OAuthClient) RemoveScope(scope oauthdomain.Scope, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("remove OAuth client scope: %w", err)
	}
	index := c.scopeIndex(scope)
	if index < 0 {
		return fmt.Errorf("remove OAuth client scope: %w", ErrScopeNotFound)
	}

	c.Scopes = append(c.Scopes[:index], c.Scopes[index+1:]...)
	c.touch(now)
	return nil
}

func (c *OAuthClient) Disable(now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("disable OAuth client: %w", err)
	}
	if c.Status != StatusActive {
		return fmt.Errorf("disable OAuth client: %w", ErrInvalidStatus)
	}

	c.Status = StatusDisabled
	c.DisabledAt = domain.ClonePointer(&now)
	c.touch(now)
	return nil
}

func (c *OAuthClient) Enable(now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enable OAuth client: %w", err)
	}
	if c.Status != StatusDisabled {
		return fmt.Errorf("enable OAuth client: %w", ErrInvalidStatus)
	}
	if c.ClientType == ClientTypeConfidential && !c.hasUsableCredential(now) {
		return fmt.Errorf("enable OAuth client: %w", ErrUsableCredentialRequired)
	}

	c.Status = StatusActive
	c.DisabledAt = nil
	c.touch(now)
	return nil
}

func (c *OAuthClient) Delete(now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("delete OAuth client: %w", err)
	}

	for index := range c.Credentials {
		if err := c.Credentials[index].Revoke(now, "oauth_client_deleted"); err != nil && !errors.Is(err, ErrCredentialRevoked) {
			return fmt.Errorf("delete OAuth client: revoke credential: %w", err)
		}
	}

	c.Status = StatusDeleted
	c.DeletedAt = domain.ClonePointer(&now)
	c.touch(now)
	return nil
}

func (c *OAuthClient) ensureActive(now time.Time) error {
	if c == nil {
		return errors.New("OAuth client is nil")
	}
	if now.IsZero() || now.Before(c.UpdatedAt) {
		return errors.New("OAuth client use time is invalid")
	}
	if c.Status != StatusActive {
		return ErrInvalidStatus
	}

	return nil
}

func (c *OAuthClient) ensureChangeAllowed(now time.Time) error {
	if c == nil {
		return errors.New("OAuth client is nil")
	}
	if c.Status == StatusDeleted {
		return ErrInvalidStatus
	}
	if now.IsZero() || now.Before(c.UpdatedAt) {
		return errors.New("OAuth client change time is invalid")
	}

	return nil
}

func (c *OAuthClient) hasUsableCredential(now time.Time) bool {
	for index := range c.Credentials {
		if c.Credentials[index].IsUsable(now) {
			return true
		}
	}
	return false
}

func (c *OAuthClient) credentialIndex(id CredentialID) int {
	for index := range c.Credentials {
		if c.Credentials[index].ID == id {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) redirectURIIndex(id RedirectURIID) int {
	for index := range c.RedirectURIs {
		if c.RedirectURIs[index].ID == id {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) redirectURIIndexByValue(uri RedirectURIValue) int {
	for index := range c.RedirectURIs {
		if c.RedirectURIs[index].URI == uri {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) postLogoutRedirectURIIndex(id PostLogoutRedirectURIID) int {
	for index := range c.PostLogoutRedirectURIs {
		if c.PostLogoutRedirectURIs[index].ID == id {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) postLogoutRedirectURIIndexByValue(uri PostLogoutRedirectURIValue) int {
	for index := range c.PostLogoutRedirectURIs {
		if c.PostLogoutRedirectURIs[index].URI == uri {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) grantTypeIndex(grantType GrantType) int {
	for index := range c.GrantTypes {
		if c.GrantTypes[index].GrantType == grantType {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) scopeIndex(scope oauthdomain.Scope) int {
	for index := range c.Scopes {
		if c.Scopes[index].Scope == scope {
			return index
		}
	}
	return -1
}

func (c *OAuthClient) touch(now time.Time) {
	c.UpdatedAt = now
}

func authMethodFor(clientType ClientType) TokenEndpointAuthMethod {
	if clientType == ClientTypeConfidential {
		return TokenEndpointAuthMethodClientSecretBasic
	}
	return TokenEndpointAuthMethodNone
}
