package ctxmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	ctx := t.Context()
	ctx = WithRequestID(ctx, "request-123")
	ctx = WithCorrelationID(ctx, "correlation-123")
	ctx = WithMessageID(ctx, "message-123")
	ctx = WithCausationID(ctx, "causation-123")
	ctx = WithTenantID(ctx, "tenant-123")

	requestID, ok := RequestID(ctx)
	require.True(t, ok)
	assert.Equal(t, "request-123", requestID)
	correlationID, ok := CorrelationID(ctx)
	require.True(t, ok)
	assert.Equal(t, "correlation-123", correlationID)
	messageID, ok := MessageID(ctx)
	require.True(t, ok)
	assert.Equal(t, "message-123", messageID)
	causationID, ok := CausationID(ctx)
	require.True(t, ok)
	assert.Equal(t, "causation-123", causationID)
	tenantID, ok := TenantID(ctx)
	require.True(t, ok)
	assert.Equal(t, "tenant-123", tenantID)
}

func TestPrincipal(t *testing.T) {
	want := Principal{
		SubjectID: "user-123",
	}

	ctx := WithPrincipal(t.Context(), want)

	got, ok := PrincipalFromContext(ctx)

	require.True(t, ok)
	require.Equal(t, want, got)
}
