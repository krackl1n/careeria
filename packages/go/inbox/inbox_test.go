package inbox

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type testMessage struct {
	Consumer  string
	MessageID string
}

type fakeTxManager struct {
	err error

	calls int

	ctxValueKey   any
	ctxValueValue any
}

func (m *fakeTxManager) Do(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	m.calls++

	if m.err != nil {
		return m.err
	}

	if m.ctxValueKey != nil {
		ctx = context.WithValue(ctx, m.ctxValueKey, m.ctxValueValue)
	}

	return fn(ctx)
}

type fakeRepository struct {
	inserted bool
	err      error

	calls    int
	messages []testMessage
}

func (r *fakeRepository) Insert(
	ctx context.Context,
	message testMessage,
) (bool, error) {
	r.calls++
	r.messages = append(r.messages, message)

	if r.err != nil {
		return false, r.err
	}

	return r.inserted, nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		txManager  TxManager
		repository Repository[testMessage]
		wantErr    error
	}{
		{
			name:       "nil tx manager",
			txManager:  nil,
			repository: &fakeRepository{},
			wantErr:    ErrNilTxManager,
		},
		{
			name:       "nil repository",
			txManager:  &fakeTxManager{},
			repository: nil,
			wantErr:    ErrNilRepository,
		},
		{
			name:       "success",
			txManager:  &fakeTxManager{},
			repository: &fakeRepository{},
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inbox, err := New(
				tt.txManager,
				tt.repository,
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, inbox)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, inbox)
		})
	}
}

func TestInbox_WithInbox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		txErr        error
		repoInserted bool
		repoErr      error
		handlerErr   error
		handlerNil   bool

		wantErr        error
		wantTxCalls    int
		wantRepoCalls  int
		wantHandlerRun bool
	}{
		{
			name: "nil handler",

			handlerNil: true,

			wantErr:        ErrNilHandler,
			wantTxCalls:    0,
			wantRepoCalls:  0,
			wantHandlerRun: false,
		},
		{
			name: "tx manager error",

			txErr: errors.New("tx failed"),

			wantErr:        errors.New("tx failed"),
			wantTxCalls:    1,
			wantRepoCalls:  0,
			wantHandlerRun: false,
		},
		{
			name: "repository insert error",

			repoErr: errors.New("insert failed"),

			wantErr:        errors.New("insert failed"),
			wantTxCalls:    1,
			wantRepoCalls:  1,
			wantHandlerRun: false,
		},
		{
			name: "duplicate message",

			repoInserted: false,

			wantErr:        nil,
			wantTxCalls:    1,
			wantRepoCalls:  1,
			wantHandlerRun: false,
		},
		{
			name: "handler success",

			repoInserted: true,

			wantErr:        nil,
			wantTxCalls:    1,
			wantRepoCalls:  1,
			wantHandlerRun: true,
		},
		{
			name: "handler error",

			repoInserted: true,
			handlerErr:   errors.New("handler failed"),

			wantErr:        errors.New("handler failed"),
			wantTxCalls:    1,
			wantRepoCalls:  1,
			wantHandlerRun: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txManager := &fakeTxManager{
				err: tt.txErr,
			}

			repository := &fakeRepository{
				inserted: tt.repoInserted,
				err:      tt.repoErr,
			}

			inbox, err := New(
				txManager,
				repository,
			)
			require.NoError(t, err)

			message := testMessage{
				Consumer:  "orders.payments-events",
				MessageID: "message-1",
			}

			handlerRun := false

			var handler Handler
			if !tt.handlerNil {
				handler = func(ctx context.Context) error {
					handlerRun = true

					return tt.handlerErr
				}
			}

			err = inbox.WithInbox(
				context.Background(),
				message,
				handler,
			)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantTxCalls, txManager.calls)
			require.Equal(t, tt.wantRepoCalls, repository.calls)
			require.Equal(t, tt.wantHandlerRun, handlerRun)

			if tt.wantRepoCalls > 0 {
				require.Equal(t, []testMessage{message}, repository.messages)
			}
		})
	}
}

func TestInbox_WithInbox_PassesTransactionContextToHandler(t *testing.T) {
	t.Parallel()

	type contextKey string

	const key contextKey = "tx"

	txManager := &fakeTxManager{
		ctxValueKey:   key,
		ctxValueValue: "transaction",
	}

	repository := &fakeRepository{
		inserted: true,
	}

	inbox, err := New(
		txManager,
		repository,
	)
	require.NoError(t, err)

	var gotValue any

	err = inbox.WithInbox(
		context.Background(),
		testMessage{
			Consumer:  "orders.payments-events",
			MessageID: "message-1",
		},
		func(ctx context.Context) error {
			gotValue = ctx.Value(key)
			return nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, "transaction", gotValue)
	require.Equal(t, 1, txManager.calls)
	require.Equal(t, 1, repository.calls)
}

func TestInbox_WithInbox_DuplicateDoesNotCallHandler(t *testing.T) {
	t.Parallel()

	txManager := &fakeTxManager{}

	repository := &fakeRepository{
		inserted: false,
	}

	inbox, err := New(
		txManager,
		repository,
	)
	require.NoError(t, err)

	handlerRun := false

	err = inbox.WithInbox(
		context.Background(),
		testMessage{
			Consumer:  "orders.payments-events",
			MessageID: "message-1",
		},
		func(ctx context.Context) error {
			handlerRun = true
			return nil
		},
	)

	require.NoError(t, err)
	require.False(t, handlerRun)
	require.Equal(t, 1, txManager.calls)
	require.Equal(t, 1, repository.calls)
}
