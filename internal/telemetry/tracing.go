package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// NewTracerProvider creates a TracerProvider that exports to Alloy/Tempo via OTLP HTTP.
//
//   - "otlp" → sends via HTTP to an OTel Collector (Alloy → Tempo)
//   - ""     → noop (no traces exported)
func NewTracerProvider(ctx context.Context, exporter string) (*sdktrace.TracerProvider, error) {
	if exporter == "" {
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(), // local dev; use TLS in production
	)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("ai-providers-backend"),
			semconv.DeploymentEnvironmentKey.String("development"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
