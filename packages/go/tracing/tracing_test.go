package tracing

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/require"
)

func TestConfigAndDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig("svc")
	require.True(t, cfg.Enabled)
	require.Equal(t, "svc", cfg.ServiceName)
	require.Equal(t, ExporterGRPC, cfg.Exporter)

	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing service",
			cfg:     DefaultConfig(""),
			wantErr: "service name is required",
		},
		{
			name: "unsupported exporter",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.Exporter = "bad"
				return cfg
			}(),
			wantErr: "unsupported exporter: bad",
		},
		{
			name: "bad ratio",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.SamplingRatio = 2
				return cfg
			}(),
			wantErr: "sampling ratio must be between 0 and 1",
		},
		{
			name: "disabled skips validation",
			cfg: Config{
				Enabled: false,
			},
		},
		{
			name: "missing exporter",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.Exporter = ""
				return cfg
			}(),
			wantErr: "exporter is required",
		},
		{
			name: "missing endpoint",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.Endpoint = ""
				return cfg
			}(),
			wantErr: "endpoint is required",
		},
		{
			name: "bad export timeout",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.ExportTimeout = 0
				return cfg
			}(),
			wantErr: "export timeout must be positive",
		},
		{
			name: "bad shutdown timeout",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.ShutdownTimeout = 0
				return cfg
			}(),
			wantErr: "shutdown timeout must be positive",
		},
		{
			name: "bad batch timeout",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.BatchTimeout = 0
				return cfg
			}(),
			wantErr: "batch timeout must be positive",
		},
		{
			name: "bad max queue size",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.MaxQueueSize = 0
				return cfg
			}(),
			wantErr: "max queue size must be positive",
		},
		{
			name: "bad max export batch size",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.MaxExportBatchSize = 0
				return cfg
			}(),
			wantErr: "max export batch size must be positive",
		},
		{
			name: "batch size must not exceed queue size",
			cfg: func() Config {
				cfg := DefaultConfig("svc")
				cfg.MaxQueueSize = 1
				cfg.MaxExportBatchSize = 2
				return cfg
			}(),
			wantErr: "max export batch size must not be greater than max queue size",
		},
		{
			name: "valid",
			cfg:  DefaultConfig("svc"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestHelpers(t *testing.T) {
	t.Parallel()

	propagator := newPropagator()
	require.NotNil(t, propagator)
	require.ElementsMatch(t, []string{"traceparent", "tracestate", "baggage"}, propagator.Fields())

	always := newSampler(1)
	never := newSampler(0)
	ratio := newSampler(0.5)

	require.Equal(t, sdktrace.RecordAndSample, always.ShouldSample(sdktrace.SamplingParameters{}).Decision)
	require.Equal(t, sdktrace.Drop, never.ShouldSample(sdktrace.SamplingParameters{}).Decision)
	require.NotEmpty(t, ratio.Description())
}

func TestNewResource(t *testing.T) {
	t.Parallel()

	res, err := newResource(context.Background(), Config{
		ServiceName:    "svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	require.NoError(t, err)

	value, ok := res.Set().Value(attribute.Key("service.name"))
	require.True(t, ok)
	require.Equal(t, "svc", value.AsString())
}

func TestNewAndClose(t *testing.T) {
	t.Parallel()

	tracing, err := New(context.Background(), Config{
		Enabled:         false,
		ShutdownTimeout: time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, tracing)
	require.NoError(t, tracing.Close(context.Background()))
	require.NoError(t, tracing.Shutdown(context.Background()))

	noneCfg := DefaultConfig("svc")
	noneCfg.Exporter = ExporterNone
	tracing, err = New(context.Background(), noneCfg)
	require.NoError(t, err)
	require.NotNil(t, tracing)
	require.NoError(t, tracing.Close(context.Background()))

	var nilTracing *Tracing
	require.NoError(t, nilTracing.Close(context.Background()))
}

func TestExporterUnsupported(t *testing.T) {
	t.Parallel()

	_, err := newExporter(context.Background(), Config{
		Exporter: "bad",
	})
	require.EqualError(t, err, "unsupported exporter: bad")
}
