package tracing

import (
	"context"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
			attribute.String("go.version", runtime.Version()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	defaultResource := resource.Default()

	merged, err := resource.Merge(defaultResource, res)
	if err != nil {
		return nil, fmt.Errorf("merge resources: %w", err)
	}

	return merged, nil
}
