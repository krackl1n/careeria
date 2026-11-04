package tracing

import sdktrace "go.opentelemetry.io/otel/sdk/trace"

func newSampler(ratio float64) sdktrace.Sampler {
	if ratio >= 1 {
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}

	if ratio <= 0 {
		return sdktrace.ParentBased(sdktrace.NeverSample())
	}

	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
}
