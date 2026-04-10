package eval

import (
	"go.opentelemetry.io/otel/metric"
)

// EvalMetrics holds OpenTelemetry instruments for AI image evaluation observability.
// Initialized once at startup and injected into EvalDecorator.
type EvalMetrics struct {
	OverallScore   metric.Float64Histogram
	SuggestedRetry metric.Int64Counter
	EvalDuration   metric.Float64Histogram
}

// NewEvalMetrics creates OTel instruments from a Meter.
func NewEvalMetrics(meter metric.Meter) (*EvalMetrics, error) {
	overallScore, err := meter.Float64Histogram(
		"eval.overall_score",
		metric.WithDescription("Distribution of AI image evaluation overall scores (1-5)"),
		metric.WithExplicitBucketBoundaries(1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5),
	)
	if err != nil {
		return nil, err
	}

	suggestedRetry, err := meter.Int64Counter(
		"eval.suggested_retry_total",
		metric.WithDescription("Number of evaluations where the judge recommended a retry"),
	)
	if err != nil {
		return nil, err
	}

	evalDuration, err := meter.Float64Histogram(
		"eval.duration_seconds",
		metric.WithDescription("Time taken to run the VLM judge evaluation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.5, 1, 2, 5, 10, 15, 30),
	)
	if err != nil {
		return nil, err
	}

	return &EvalMetrics{
		OverallScore:   overallScore,
		SuggestedRetry: suggestedRetry,
		EvalDuration:   evalDuration,
	}, nil
}
