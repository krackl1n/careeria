package authn

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var validJWKSCacheConfig = JWKSCacheConfig{
	TTL:             5 * time.Minute,
	RefreshTimeout:  2 * time.Second,
	RefreshCooldown: 10 * time.Second,
}

type jwksLoaderFunc func(context.Context) (map[string][]byte, error)

func (fn jwksLoaderFunc) LoadJWKS(ctx context.Context) (map[string][]byte, error) {
	return fn(ctx)
}

func TestNewJWKSCacheValidatesDependencies(t *testing.T) {
	tests := []struct {
		name   string
		loader JWKSLoader
		config JWKSCacheConfig
		want   string
	}{
		{name: "missing loader", config: validJWKSCacheConfig, want: "JWKS loader is required"},
		{name: "missing TTL", loader: staticJWKSLoader("key-1"), config: JWKSCacheConfig{RefreshTimeout: time.Second, RefreshCooldown: time.Second}, want: "JWKS cache TTL must be positive"},
		{name: "missing refresh timeout", loader: staticJWKSLoader("key-1"), config: JWKSCacheConfig{TTL: time.Minute, RefreshCooldown: time.Second}, want: "JWKS refresh timeout must be positive"},
		{name: "missing refresh cooldown", loader: staticJWKSLoader("key-1"), config: JWKSCacheConfig{TTL: time.Minute, RefreshTimeout: time.Second}, want: "JWKS refresh cooldown must be positive"},
		{name: "cooldown exceeds TTL", loader: staticJWKSLoader("key-1"), config: JWKSCacheConfig{TTL: time.Second, RefreshTimeout: time.Second, RefreshCooldown: 2 * time.Second}, want: "JWKS refresh cooldown must not exceed cache TTL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewJWKSCache(test.loader, test.config)
			require.EqualError(t, err, test.want)
		})
	}
}

func TestJWKSCacheCachesUntilTTLAndDoesNotExposeMutableKeys(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	var calls int
	cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		calls++
		return map[string][]byte{"key-1": []byte("public-key")}, nil
	}), func() time.Time { return now })

	first, err := cache.ResolvePublicKey(t.Context(), "key-1")
	require.NoError(t, err)
	require.Equal(t, []byte("public-key"), first)
	require.Equal(t, 1, calls)

	first[0] = 'X'
	now = now.Add(time.Minute)
	second, err := cache.ResolvePublicKey(t.Context(), "key-1")
	require.NoError(t, err)
	require.Equal(t, []byte("public-key"), second)
	require.Equal(t, 1, calls)

	now = now.Add(validJWKSCacheConfig.TTL)
	_, err = cache.ResolvePublicKey(t.Context(), "key-1")
	require.NoError(t, err)
	require.Equal(t, 2, calls)
}

func TestJWKSCacheRefreshesForRotatedKeyAfterCooldown(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	var calls int
	cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		calls++
		if calls == 1 {
			return map[string][]byte{"old": []byte("old-key")}, nil
		}
		return map[string][]byte{"old": []byte("old-key"), "new": []byte("new-key")}, nil
	}), func() time.Time { return now })

	_, err := cache.ResolvePublicKey(t.Context(), "old")
	require.NoError(t, err)

	_, err = cache.ResolvePublicKey(t.Context(), "new")
	require.EqualError(t, err, `public key "new" was not found`)
	require.Equal(t, 1, calls)

	now = now.Add(validJWKSCacheConfig.RefreshCooldown)
	key, err := cache.ResolvePublicKey(t.Context(), "new")
	require.NoError(t, err)
	require.Equal(t, []byte("new-key"), key)
	require.Equal(t, 2, calls)
}

func TestJWKSCacheCollapsesConcurrentRefreshes(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return map[string][]byte{"key-1": []byte("public-key")}, nil
	}), time.Now)

	const goroutines = 64
	var wg sync.WaitGroup
	errorsByCall := make(chan error, goroutines)
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.ResolvePublicKey(context.Background(), "key-1")
			errorsByCall <- err
		}()
	}

	<-started
	close(release)
	wg.Wait()
	close(errorsByCall)

	for err := range errorsByCall {
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), calls.Load())
}

func TestJWKSCacheRefreshOutlivesCanceledWaiter(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		calls.Add(1)
		close(started)
		<-release
		return map[string][]byte{"key-1": []byte("public-key")}, nil
	}), time.Now)

	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := cache.ResolvePublicKey(firstCtx, "key-1")
		firstResult <- err
	}()

	<-started
	secondResult := make(chan error, 1)
	go func() {
		_, err := cache.ResolvePublicKey(context.Background(), "key-1")
		secondResult <- err
	}()

	cancelFirst()
	require.ErrorIs(t, <-firstResult, context.Canceled)
	close(release)
	require.NoError(t, <-secondResult)
	require.Equal(t, int32(1), calls.Load())
}

func TestJWKSCacheThrottlesFailedRefreshAndPreservesPreviousSet(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	var calls int
	cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		calls++
		if calls > 1 {
			return nil, errors.New("identity unavailable")
		}
		return map[string][]byte{"key-1": []byte("public-key")}, nil
	}), func() time.Time { return now })

	_, err := cache.ResolvePublicKey(t.Context(), "key-1")
	require.NoError(t, err)

	now = now.Add(validJWKSCacheConfig.TTL)
	_, err = cache.ResolvePublicKey(t.Context(), "key-1")
	require.ErrorContains(t, err, "identity unavailable")
	require.Equal(t, 2, calls)

	_, err = cache.ResolvePublicKey(t.Context(), "key-1")
	require.ErrorContains(t, err, "identity unavailable")
	require.Equal(t, 2, calls)

	cache.mu.RLock()
	require.Equal(t, []byte("public-key"), cache.keys["key-1"])
	cache.mu.RUnlock()
}

func TestJWKSCacheThrottlesUnknownKeyRefreshes(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	var calls int
	cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		calls++
		return map[string][]byte{"known": []byte("public-key")}, nil
	}), func() time.Time { return now })

	_, err := cache.ResolvePublicKey(t.Context(), "known")
	require.NoError(t, err)

	now = now.Add(validJWKSCacheConfig.RefreshCooldown)
	_, err = cache.ResolvePublicKey(t.Context(), "unknown-1")
	require.EqualError(t, err, `public key "unknown-1" was not found`)
	require.Equal(t, 2, calls)

	_, err = cache.ResolvePublicKey(t.Context(), "unknown-2")
	require.EqualError(t, err, `public key "unknown-2" was not found`)
	require.Equal(t, 2, calls)
}

func TestJWKSCacheRejectsInvalidLoadedSet(t *testing.T) {
	tests := []struct {
		name string
		keys map[string][]byte
		want string
	}{
		{name: "empty set", keys: map[string][]byte{}, want: `resolve public key "key-1": validate JWKS: key set is empty`},
		{name: "empty key ID", keys: map[string][]byte{"": []byte("public-key")}, want: `resolve public key "key-1": validate JWKS: key ID is empty`},
		{name: "empty key", keys: map[string][]byte{"key-1": nil}, want: `resolve public key "key-1": validate JWKS: public key "key-1" is empty`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache := newTestJWKSCache(t, jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
				return test.keys, nil
			}), time.Now)

			_, err := cache.ResolvePublicKey(t.Context(), "key-1")
			require.EqualError(t, err, test.want)
		})
	}
}

func TestJWKSCacheAppliesRefreshTimeout(t *testing.T) {
	config := validJWKSCacheConfig
	config.RefreshTimeout = 10 * time.Millisecond
	cache, err := NewJWKSCache(jwksLoaderFunc(func(ctx context.Context) (map[string][]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}), config)
	require.NoError(t, err)

	_, err = cache.ResolvePublicKey(t.Context(), "key-1")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func staticJWKSLoader(keyID string) JWKSLoader {
	return jwksLoaderFunc(func(context.Context) (map[string][]byte, error) {
		return map[string][]byte{keyID: []byte("public-key")}, nil
	})
}

func newTestJWKSCache(t *testing.T, loader JWKSLoader, now func() time.Time) *JWKSCache {
	t.Helper()
	cache, err := NewJWKSCache(loader, validJWKSCacheConfig)
	require.NoError(t, err)
	cache.now = now
	return cache
}
