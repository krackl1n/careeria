package outbox

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	validCfg := Config{
		BatchSize:      10,
		PollInterval:   time.Second,
		PublishTimeout: time.Second,
	}

	tests := []struct {
		name       string
		cfg        Config
		repository Repository[string]
		publisher  Publisher[string]
		logger     Logger
		wantErr    bool
	}{
		{
			name:       "valid",
			cfg:        validCfg,
			repository: &testRepository[string]{},
			publisher:  &testPublisher[string]{},
			logger:     &testLogger{},
			wantErr:    false,
		},
		{
			name: "invalid config",
			cfg: Config{
				BatchSize:      0,
				PollInterval:   time.Second,
				PublishTimeout: time.Second,
			},
			repository: &testRepository[string]{},
			publisher:  &testPublisher[string]{},
			logger:     &testLogger{},
			wantErr:    true,
		},
		{
			name:       "nil repository",
			cfg:        validCfg,
			repository: nil,
			publisher:  &testPublisher[string]{},
			logger:     &testLogger{},
			wantErr:    true,
		},
		{
			name:       "nil publisher",
			cfg:        validCfg,
			repository: &testRepository[string]{},
			publisher:  nil,
			logger:     &testLogger{},
			wantErr:    true,
		},
		{
			name:       "nil logger",
			cfg:        validCfg,
			repository: &testRepository[string]{},
			publisher:  &testPublisher[string]{},
			logger:     nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(
				tt.cfg,
				tt.repository,
				tt.publisher,
				tt.logger,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, w)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, w)
		})
	}
}

func TestWorkerProcess(t *testing.T) {
	claimErr := errors.New("claim failed")
	publishErr := errors.New("publish failed")
	markPublishedErr := errors.New("mark published failed")
	markFailedErr := errors.New("mark failed failed")

	tests := []struct {
		name             string
		batchSize        int
		entries          []Entry[string]
		claimErr         error
		publishErrors    map[string]error
		markPublishedErr error
		markFailedErrors map[string]error

		wantFull         bool
		wantClaimCalls   int
		wantPublishCalls []string
		wantPublished    []Entry[string]
		wantFailed       []Entry[string]
		wantLogCount     int
	}{
		{
			name:             "empty batch",
			batchSize:        10,
			entries:          nil,
			wantFull:         false,
			wantClaimCalls:   1,
			wantPublishCalls: nil,
			wantPublished:    nil,
			wantFailed:       nil,
			wantLogCount:     0,
		},
		{
			name:      "full batch published",
			batchSize: 2,
			entries: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			wantFull:         true,
			wantClaimCalls:   1,
			wantPublishCalls: []string{"a", "b"},
			wantPublished: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			wantFailed:   nil,
			wantLogCount: 0,
		},
		{
			name:      "partial batch published",
			batchSize: 3,
			entries: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			wantFull:         false,
			wantClaimCalls:   1,
			wantPublishCalls: []string{"a", "b"},
			wantPublished: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			wantFailed:   nil,
			wantLogCount: 0,
		},
		{
			name:      "one publish failed",
			batchSize: 2,
			entries: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			publishErrors: map[string]error{
				"a": publishErr,
			},
			wantFull:         true,
			wantClaimCalls:   1,
			wantPublishCalls: []string{"a", "b"},
			wantPublished: []Entry[string]{
				newTestEntry("2", "b"),
			},
			wantFailed: []Entry[string]{
				newTestEntry("1", "a"),
			},
			wantLogCount: 1,
		},
		{
			name:      "mark failed failed",
			batchSize: 2,
			entries: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			publishErrors: map[string]error{
				"a": publishErr,
			},
			markFailedErrors: map[string]error{
				"a": markFailedErr,
			},
			wantFull:         false,
			wantClaimCalls:   1,
			wantPublishCalls: []string{"a", "b"},
			wantPublished: []Entry[string]{
				newTestEntry("2", "b"),
			},
			wantFailed: []Entry[string]{
				newTestEntry("1", "a"),
			},
			wantLogCount: 2,
		},
		{
			name:      "mark published failed",
			batchSize: 2,
			entries: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			markPublishedErr: markPublishedErr,
			wantFull:         false,
			wantClaimCalls:   1,
			wantPublishCalls: []string{"a", "b"},
			wantPublished: []Entry[string]{
				newTestEntry("1", "a"),
				newTestEntry("2", "b"),
			},
			wantFailed:   nil,
			wantLogCount: 1,
		},
		{
			name:             "claim failed",
			batchSize:        2,
			claimErr:         claimErr,
			wantFull:         false,
			wantClaimCalls:   1,
			wantPublishCalls: nil,
			wantPublished:    nil,
			wantFailed:       nil,
			wantLogCount:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &testRepository[string]{
				claimFn: func(context.Context, int) ([]Entry[string], error) {
					return tt.entries, tt.claimErr
				},
				markPublishedFn: func(context.Context, []Entry[string]) error {
					return tt.markPublishedErr
				},
				markFailedFn: func(_ context.Context, entry Entry[string], _ error) error {
					if tt.markFailedErrors == nil {
						return nil
					}

					return tt.markFailedErrors[entry.Message]
				},
			}

			publisher := &testPublisher[string]{
				publishFn: func(_ context.Context, message string) error {
					if tt.publishErrors == nil {
						return nil
					}

					return tt.publishErrors[message]
				},
			}

			logger := &testLogger{}

			w, err := New(
				Config{
					BatchSize:      tt.batchSize,
					PollInterval:   time.Second,
					PublishTimeout: time.Second,
				},
				repository,
				publisher,
				logger,
			)
			require.NoError(t, err)

			full := w.process(context.Background())

			require.Equal(t, tt.wantFull, full)
			require.Equal(t, tt.wantClaimCalls, repository.claimCallsCount())
			require.Equal(t, tt.wantPublishCalls, publisher.publishCallsSnapshot())
			require.Equal(t, tt.wantPublished, repository.publishedSnapshot())
			require.Equal(t, tt.wantFailed, repository.failedSnapshot())
			require.Equal(t, tt.wantLogCount, logger.count())
		})
	}
}

func TestWorkerStartShutdown(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "shutdown waits current batch",
			run: func(t *testing.T) {
				publishStarted := make(chan struct{})
				releasePublish := make(chan struct{})

				repository := &testRepository[string]{
					claimFn: func(context.Context, int) ([]Entry[string], error) {
						return []Entry[string]{
							newTestEntry("1", "a"),
						}, nil
					},
				}

				publisher := &testPublisher[string]{
					publishFn: func(context.Context, string) error {
						close(publishStarted)
						<-releasePublish
						return nil
					},
				}

				w, err := New(
					Config{
						BatchSize:      10,
						PollInterval:   time.Hour,
						PublishTimeout: time.Second,
					},
					repository,
					publisher,
					&testLogger{},
				)
				require.NoError(t, err)

				startErrCh := make(chan error, 1)

				go func() {
					startErrCh <- w.Start(context.Background())
				}()

				require.Eventually(t, func() bool {
					select {
					case <-publishStarted:
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				shutdownErrCh := make(chan error, 1)

				go func() {
					shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()

					shutdownErrCh <- w.Shutdown(shutdownCtx)
				}()

				require.Never(t, func() bool {
					select {
					case <-shutdownErrCh:
						return true
					default:
						return false
					}
				}, 30*time.Millisecond, time.Millisecond)

				close(releasePublish)

				require.Eventually(t, func() bool {
					select {
					case err := <-shutdownErrCh:
						require.NoError(t, err)
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				require.Eventually(t, func() bool {
					select {
					case err := <-startErrCh:
						require.NoError(t, err)
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				require.Equal(t, []Entry[string]{
					newTestEntry("1", "a"),
				}, repository.publishedSnapshot())
			},
		},
		{
			name: "shutdown wakes worker during poll interval",
			run: func(t *testing.T) {
				claimCalled := make(chan struct{})

				repository := &testRepository[string]{
					claimFn: func(context.Context, int) ([]Entry[string], error) {
						close(claimCalled)
						return nil, nil
					},
				}

				w, err := New(
					Config{
						BatchSize:      10,
						PollInterval:   time.Hour,
						PublishTimeout: time.Second,
					},
					repository,
					&testPublisher[string]{},
					&testLogger{},
				)
				require.NoError(t, err)

				startErrCh := make(chan error, 1)

				go func() {
					startErrCh <- w.Start(context.Background())
				}()

				require.Eventually(t, func() bool {
					select {
					case <-claimCalled:
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				require.NoError(t, w.Shutdown(shutdownCtx))

				require.Eventually(t, func() bool {
					select {
					case err := <-startErrCh:
						require.NoError(t, err)
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				require.Equal(t, 1, repository.claimCallsCount())
			},
		},
		{
			name: "full batch continues without waiting poll interval",
			run: func(t *testing.T) {
				var calls atomic.Int64
				secondClaim := make(chan struct{})

				repository := &testRepository[string]{
					claimFn: func(context.Context, int) ([]Entry[string], error) {
						call := calls.Add(1)

						if call == 1 {
							return []Entry[string]{
								newTestEntry("1", "a"),
							}, nil
						}

						if call == 2 {
							close(secondClaim)
							return nil, nil
						}

						return nil, nil
					},
				}

				w, err := New(
					Config{
						BatchSize:      1,
						PollInterval:   time.Hour,
						PublishTimeout: time.Second,
					},
					repository,
					&testPublisher[string]{},
					&testLogger{},
				)
				require.NoError(t, err)

				startErrCh := make(chan error, 1)

				go func() {
					startErrCh <- w.Start(context.Background())
				}()

				require.Eventually(t, func() bool {
					select {
					case <-secondClaim:
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				require.NoError(t, w.Shutdown(shutdownCtx))

				require.Eventually(t, func() bool {
					select {
					case err := <-startErrCh:
						require.NoError(t, err)
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				require.Equal(t, 2, repository.claimCallsCount())
			},
		},
		{
			name: "shutdown returns context error on timeout",
			run: func(t *testing.T) {
				publishStarted := make(chan struct{})
				releasePublish := make(chan struct{})

				repository := &testRepository[string]{
					claimFn: func(context.Context, int) ([]Entry[string], error) {
						return []Entry[string]{
							newTestEntry("1", "a"),
						}, nil
					},
				}

				publisher := &testPublisher[string]{
					publishFn: func(context.Context, string) error {
						close(publishStarted)
						<-releasePublish
						return nil
					},
				}

				w, err := New(
					Config{
						BatchSize:      10,
						PollInterval:   time.Hour,
						PublishTimeout: time.Second,
					},
					repository,
					publisher,
					&testLogger{},
				)
				require.NoError(t, err)

				startErrCh := make(chan error, 1)

				go func() {
					startErrCh <- w.Start(context.Background())
				}()

				require.Eventually(t, func() bool {
					select {
					case <-publishStarted:
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)

				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()

				require.ErrorIs(t, w.Shutdown(shutdownCtx), context.DeadlineExceeded)

				close(releasePublish)

				require.Eventually(t, func() bool {
					select {
					case err := <-startErrCh:
						require.NoError(t, err)
						return true
					default:
						return false
					}
				}, time.Second, time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func newTestEntry(id string, message string) Entry[string] {
	return Entry[string]{
		ID:         id,
		ClaimToken: "claim-" + id,
		Message:    message,
	}
}

type testRepository[M any] struct {
	mu sync.Mutex

	claimFn         func(context.Context, int) ([]Entry[M], error)
	markPublishedFn func(context.Context, []Entry[M]) error
	markFailedFn    func(context.Context, Entry[M], error) error

	claimCalls int
	published  []Entry[M]
	failed     []Entry[M]
}

func (r *testRepository[M]) ClaimPending(ctx context.Context, limit int) ([]Entry[M], error) {
	r.mu.Lock()
	r.claimCalls++
	r.mu.Unlock()

	if r.claimFn == nil {
		return nil, nil
	}

	return r.claimFn(ctx, limit)
}

func (r *testRepository[M]) MarkPublished(ctx context.Context, entries []Entry[M]) error {
	r.mu.Lock()
	r.published = append(r.published, entries...)
	r.mu.Unlock()

	if r.markPublishedFn == nil {
		return nil
	}

	return r.markPublishedFn(ctx, entries)
}

func (r *testRepository[M]) MarkFailed(ctx context.Context, entry Entry[M], cause error) error {
	r.mu.Lock()
	r.failed = append(r.failed, entry)
	r.mu.Unlock()

	if r.markFailedFn == nil {
		return nil
	}

	return r.markFailedFn(ctx, entry, cause)
}

func (r *testRepository[M]) claimCallsCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.claimCalls
}

func (r *testRepository[M]) publishedSnapshot() []Entry[M] {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]Entry[M](nil), r.published...)
}

func (r *testRepository[M]) failedSnapshot() []Entry[M] {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]Entry[M](nil), r.failed...)
}

type testPublisher[M any] struct {
	mu sync.Mutex

	publishFn func(context.Context, M) error
	published []M
}

func (p *testPublisher[M]) Publish(ctx context.Context, message M) error {
	p.mu.Lock()
	p.published = append(p.published, message)
	p.mu.Unlock()

	if p.publishFn == nil {
		return nil
	}

	return p.publishFn(ctx, message)
}

func (p *testPublisher[M]) publishCallsSnapshot() []M {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]M(nil), p.published...)
}

type testLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *testLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.messages = append(l.messages, msg)
}

func (l *testLogger) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.messages)
}
