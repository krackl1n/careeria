package oauth

import (
	"errors"
	"time"
)

type Config struct {
	AuthorizationLifetime time.Duration
	CodeLifetime          time.Duration
	SessionIdleLifetime   time.Duration
	SessionLifetime       time.Duration
	RefreshTokenLifetime  time.Duration
	RefreshFamilyLifetime time.Duration
}

func (c Config) Validate() error {
	if c.AuthorizationLifetime <= 0 {
		return errors.New("authorization lifetime must be greater than 0")
	}
	if c.CodeLifetime <= 0 {
		return errors.New("code lifetime must be greater than 0")
	}
	if c.SessionIdleLifetime <= 0 || c.SessionIdleLifetime > c.SessionLifetime {
		return errors.New("session idle lifetime is invalid")
	}
	if c.SessionLifetime <= 0 {
		return errors.New("session lifetime must be greater than 0")
	}
	if c.RefreshTokenLifetime <= 0 || c.RefreshTokenLifetime > c.RefreshFamilyLifetime {
		return errors.New("refresh token lifetime is invalid")
	}
	if c.RefreshFamilyLifetime <= 0 || c.RefreshFamilyLifetime > c.SessionLifetime {
		return errors.New("refresh token family lifetime is invalid")
	}

	return nil
}
