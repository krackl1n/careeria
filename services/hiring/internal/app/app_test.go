package app

import (
	"bytes"
	"testing"
	"time"

	"github.com/krackl1n/careeria/closer"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	sharedlogger "github.com/krackl1n/careeria/logger"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	internalconfig "github.com/krackl1n/careeria/services/hiring/internal/config"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	_, err := New(nil)
	require.Error(t, err)

	dependencies := &Dependencies{
		Config: internalconfig.Config{},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	app, err := New(dependencies)
	require.NoError(t, err)
	require.NotNil(t, app)
}

func TestRuntimesInitialization(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			GRPCServer: grpcserver.Config{
				Addr: "127.0.0.1:0",
			},
			Metrics: sharedmetrics.Config{
				Enabled:           true,
				Addr:              "127.0.0.1:0",
				Namespace:         "careeria_hiring",
				ReadHeaderTimeout: 5 * time.Second,
			},
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	app, err := New(dependencies)
	require.NoError(t, err)

	runtimes, err := app.Runtimes(t.Context())
	require.NoError(t, err)
	require.Len(t, runtimes, 2)
}

