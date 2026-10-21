package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case ExporterGRPC:
		exporter, err := newGRPCExporter(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("create grpc exporter: %w", err)
		}

		return exporter, nil

	case ExporterHTTP:
		exporter, err := newHTTPExporter(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("create http exporter: %w", err)
		}

		return exporter, nil

	default:
		return nil, fmt.Errorf("unsupported exporter: %s", cfg.Exporter)
	}
}

func newGRPCExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(cfg.ExportTimeout),
	}

	if cfg.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	client := otlptracegrpc.NewClient(options...)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("initialize otlp grpc trace exporter: %w", err)
	}

	return exporter, nil
}

func newHTTPExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithTimeout(cfg.ExportTimeout),
	}

	if cfg.Insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("initialize otlp http trace exporter: %w", err)
	}

	return exporter, nil
}
