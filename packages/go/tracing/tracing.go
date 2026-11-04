package tracing

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Tracing struct {
	provider *sdktrace.TracerProvider
	cfg      Config
	once     sync.Once
	err      error
}

func New(ctx context.Context, cfg Config) (*Tracing, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate tracing config: %w", err)
	}

	otel.SetTextMapPropagator(newPropagator())

	if !cfg.Enabled || cfg.Exporter == ExporterNone {
		provider := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(provider)

		return &Tracing{
			provider: provider,
			cfg:      cfg,
		}, nil
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create tracing exporter: %w", err)
	}

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create tracing resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(newSampler(cfg.SamplingRatio)),
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithExportTimeout(cfg.ExportTimeout),
			sdktrace.WithBatchTimeout(cfg.BatchTimeout),
			sdktrace.WithMaxQueueSize(cfg.MaxQueueSize),
			sdktrace.WithMaxExportBatchSize(cfg.MaxExportBatchSize),
		),
	)

	otel.SetTracerProvider(provider)

	return &Tracing{
		provider: provider,
		cfg:      cfg,
	}, nil
}

func (t *Tracing) Close(ctx context.Context) error {
	if t == nil || t.provider == nil {
		return nil
	}

	t.once.Do(func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, t.cfg.ShutdownTimeout)
		defer cancel()

		if err := t.provider.Shutdown(shutdownCtx); err != nil {
			t.err = fmt.Errorf("shutdown tracing provider: %w", err)
		}
	})

	return t.err
}

func (t *Tracing) Shutdown(ctx context.Context) error {
	return t.Close(ctx)
}
