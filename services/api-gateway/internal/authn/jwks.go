package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const jwksRefreshKey = "jwks"

type JWKSLoader interface {
	LoadJWKS(context.Context) (map[string][]byte, error)
}

type JWKSCacheConfig struct {
	TTL             time.Duration
	RefreshTimeout  time.Duration
	RefreshCooldown time.Duration
}

func (c JWKSCacheConfig) Validate() error {
	if c.TTL <= 0 {
		return errors.New("JWKS cache TTL must be positive")
	}
	if c.RefreshTimeout <= 0 {
		return errors.New("JWKS refresh timeout must be positive")
	}
	if c.RefreshCooldown <= 0 {
		return errors.New("JWKS refresh cooldown must be positive")
	}
	if c.RefreshCooldown > c.TTL {
		return errors.New("JWKS refresh cooldown must not exceed cache TTL")
	}
	return nil
}

type JWKSCache struct {
	loader JWKSLoader
	config JWKSCacheConfig
	now    func() time.Time

	mu             sync.RWMutex
	keys           map[string][]byte
	expiresAt      time.Time
	nextRefreshAt  time.Time
	lastRefreshErr error
	generation     uint64

	refresh singleflight.Group
}

func NewJWKSCache(loader JWKSLoader, config JWKSCacheConfig) (*JWKSCache, error) {
	if loader == nil {
		return nil, errors.New("JWKS loader is required")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &JWKSCache{
		loader: loader,
		config: config,
		now:    time.Now,
		keys:   make(map[string][]byte),
	}, nil
}

func (c *JWKSCache) ResolvePublicKey(ctx context.Context, keyID string) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("resolve public key: context is required")
	}
	if strings.TrimSpace(keyID) == "" {
		return nil, errors.New("resolve public key: key ID is required")
	}

	key, found, fresh, generation := c.lookup(keyID, c.now())
	if found && fresh {
		return key, nil
	}

	if err := c.refreshJWKS(ctx, generation); err != nil {
		return nil, fmt.Errorf("resolve public key %q: %w", keyID, err)
	}

	key, found, _, _ = c.lookup(keyID, c.now())
	if !found {
		return nil, fmt.Errorf("public key %q was not found", keyID)
	}
	return key, nil
}

func (c *JWKSCache) lookup(keyID string, now time.Time) ([]byte, bool, bool, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key, found := c.keys[keyID]
	return cloneBytes(key), found, now.Before(c.expiresAt), c.generation
}

func (c *JWKSCache) refreshJWKS(ctx context.Context, observedGeneration uint64) error {
	result := c.refresh.DoChan(jwksRefreshKey, func() (any, error) {
		now := c.now()

		c.mu.Lock()
		if c.generation != observedGeneration {
			c.mu.Unlock()
			return nil, nil
		}
		if now.Before(c.nextRefreshAt) {
			err := c.lastRefreshErr
			c.mu.Unlock()
			return nil, err
		}
		c.mu.Unlock()

		refreshCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			c.config.RefreshTimeout,
		)
		defer cancel()

		loaded, err := c.loader.LoadJWKS(refreshCtx)
		if err != nil {
			err = fmt.Errorf("load JWKS: %w", err)
			c.recordRefreshError(err, c.now())
			return nil, err
		}

		keys, err := validateAndCloneJWKS(loaded)
		if err != nil {
			err = fmt.Errorf("validate JWKS: %w", err)
			c.recordRefreshError(err, c.now())
			return nil, err
		}

		now = c.now()
		c.mu.Lock()
		c.keys = keys
		c.expiresAt = now.Add(c.config.TTL)
		c.nextRefreshAt = now.Add(c.config.RefreshCooldown)
		c.lastRefreshErr = nil
		c.generation++
		c.mu.Unlock()

		return nil, nil
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case value := <-result:
		return value.Err
	}
}

func (c *JWKSCache) recordRefreshError(err error, now time.Time) {
	c.mu.Lock()
	c.lastRefreshErr = err
	c.nextRefreshAt = now.Add(c.config.RefreshCooldown)
	c.mu.Unlock()
}

func validateAndCloneJWKS(keys map[string][]byte) (map[string][]byte, error) {
	if len(keys) == 0 {
		return nil, errors.New("key set is empty")
	}

	result := make(map[string][]byte, len(keys))
	for keyID, key := range keys {
		if strings.TrimSpace(keyID) == "" {
			return nil, errors.New("key ID is empty")
		}
		if len(key) == 0 {
			return nil, fmt.Errorf("public key %q is empty", keyID)
		}
		result[keyID] = cloneBytes(key)
	}
	return result, nil
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}
