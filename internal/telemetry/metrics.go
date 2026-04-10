package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	prometheusexp "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// NewMeterProvider creates a MeterProvider based on the exporter name.
//
//   - "stdout" → prints metrics to stdout (dev/debug)
//   - "otlp"   → sends via gRPC to an OTel Collector (production)
//   - ""       → noop (no metrics exported)
func NewMeterProvider(ctx context.Context, exporter string) (*sdkmetric.MeterProvider, error) {
	var reader sdkmetric.Reader

	switch exporter {
	case "stdout":
		exp, err := stdoutmetric.New()
		if err != nil {
			return nil, fmt.Errorf("stdout metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(15*time.Second))

	case "otlp":
		exp, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("otlp metric exporter: %w", err)
		}
		reader = sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(30*time.Second))

	case "prometheus":
		exp, err := prometheusexp.New()
		if err != nil {
			return nil, fmt.Errorf("prometheus metric exporter: %w", err)
		}
		reader = exp // prometheus.Exporter implements Reader

	case "":
		// No exporter — return a provider with no reader (noop).
		return sdkmetric.NewMeterProvider(), nil

	default:
		return nil, fmt.Errorf("unknown metric exporter: %q (supported: stdout, otlp, prometheus)", exporter)
	}

	return sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), nil
}
