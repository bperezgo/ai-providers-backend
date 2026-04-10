package eval

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/port"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/judge"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// EvalHook is called after each evaluation completes (success or error).
// score is nil when evalErr is non-nil. Use WithHook to register it.
type EvalHook func(prompt string, score *providers.EvalScore, evalErr error)

// DecoratorOption configures an EvalDecorator.
type DecoratorOption func(*EvalDecorator)

// WithHook registers an optional callback invoked after each eval completes.
// Useful for testing and custom observability integrations.
func WithHook(hook EvalHook) DecoratorOption {
	return func(d *EvalDecorator) { d.hook = hook }
}

// WithMetrics enables OpenTelemetry metric recording for evaluations.
func WithMetrics(m *EvalMetrics) DecoratorOption {
	return func(d *EvalDecorator) { d.metrics = m }
}

// Strategy determines when evaluation should run.
type Strategy interface {
	ShouldEvaluate() bool
}

// AlwaysEvaluate runs the judge on every request. Use for testing/dev.
type AlwaysEvaluate struct{}

func (AlwaysEvaluate) ShouldEvaluate() bool { return true }

// NeverEvaluate skips the judge entirely. Use to disable without unwrapping.
type NeverEvaluate struct{}

func (NeverEvaluate) ShouldEvaluate() bool { return false }

// SampleEvaluate runs the judge every N requests.
//
// Production monitoring considerations:
// - Where to send eval results (metrics backend, logging pipeline, dedicated DB table)
// - Alerting thresholds (e.g. alert if average overall_score drops below 3 over a window)
// - Per-provider tracking (compare NanoBanana vs DALL-E quality over time)
// - Cost tracking (each eval = 1 Claude API call, budget N evals/day)
// - Dashboard for visual inspection (store image + score pairs for human review)
// - Drift detection (score trends over weeks to catch model degradation)
type SampleEvaluate struct {
	every   uint64
	counter atomic.Uint64
}

// NewSampleEvaluate creates a strategy that evaluates every N requests.
func NewSampleEvaluate(every uint64) *SampleEvaluate {
	return &SampleEvaluate{every: every}
}

func (s *SampleEvaluate) ShouldEvaluate() bool {
	n := s.counter.Add(1)
	return n%s.every == 0
}

// EvalDecorator wraps any TextToImageProvider and optionally evaluates results
// using a VLM judge. Evaluation runs asynchronously — it does not block the
// image response. Use WaitForPendingEvals to drain all goroutines (e.g. in tests).
//
// Usage:
//
//	// For testing — evaluate every image:
//	wrapped := eval.NewEvalDecorator(nanoBananaClient, judge, eval.AlwaysEvaluate{}, "social-post", logger, persister)
//
//	// For production sampling — evaluate every 100th image:
//	wrapped := eval.NewEvalDecorator(nanoBananaClient, judge, eval.NewSampleEvaluate(100), "marketing-banner", logger, jobManager)
//
//	// wrapped satisfies providers.TextToImageProvider, use it anywhere the original was used.
type EvalDecorator struct {
	inner     providers.TextToImageProvider
	judge     *judge.Judge
	strategy  Strategy
	useCase   string
	logger    *zap.Logger
	hook      EvalHook
	persister port.EvalPersister
	metrics   *EvalMetrics
	wg        sync.WaitGroup
}

var _ providers.TextToImageProvider = (*EvalDecorator)(nil)

// NewEvalDecorator creates a decorator that wraps a TextToImageProvider with VLM evaluation.
// Both logger and persister are required — use zap.NewNop() and a no-op persister in tests
// that don't care about logging or persistence.
func NewEvalDecorator(inner providers.TextToImageProvider, j *judge.Judge, strategy Strategy, useCase string, logger *zap.Logger, persister port.EvalPersister, opts ...DecoratorOption) *EvalDecorator {
	d := &EvalDecorator{
		inner:     inner,
		judge:     j,
		strategy:  strategy,
		useCase:   useCase,
		logger:    logger,
		persister: persister,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// WaitForPendingEvals blocks until all background eval goroutines complete.
// Useful in tests to synchronise before asserting eval results.
func (a *EvalDecorator) WaitForPendingEvals() {
	a.wg.Wait()
}

func (a *EvalDecorator) ProviderName() string {
	return a.inner.ProviderName()
}

// GenerateImage delegates to the inner provider, then asynchronously evaluates
// the result if the strategy decides to. The returned ImageResult never contains
// EvalScore — scores are delivered via the optional EvalHook and persisted to the
// database when a job ID is present in the context (see WithJobID).
func (a *EvalDecorator) GenerateImage(ctx context.Context, prompt string, opts providers.TextToImageOptions) (*providers.ImageResult, error) {
	result, err := a.inner.GenerateImage(ctx, prompt, opts)
	if err != nil {
		return nil, err
	}

	if !a.strategy.ShouldEvaluate() {
		return result, nil
	}

	useCase := a.useCase
	if useCase == "" {
		useCase = "general marketing"
	}

	imagePath := result.LocalPath
	a.wg.Go(func() {
		// Detach from the request context so the goroutine survives after the HTTP
		// handler returns. context.WithoutCancel inherits all values (trace IDs,
		// baggage) but drops the cancellation signal and deadline.
		evalCtx := context.WithoutCancel(ctx)
		evalCtx, cancel := context.WithTimeout(evalCtx, 30*time.Second)
		defer cancel()

		evalStart := time.Now()
		score, evalErr := a.judge.Evaluate(evalCtx, judge.ImageEvalRequest{
			OriginalPrompt: prompt,
			ImagePath:      imagePath,
			UseCase:        useCase,
		})
		evalDuration := time.Since(evalStart).Seconds()

		if a.metrics != nil {
			metricAttrs := metric.WithAttributes(
				attribute.String("provider", a.inner.ProviderName()),
				attribute.String("use_case", useCase),
			)
			a.metrics.EvalDuration.Record(evalCtx, evalDuration, metricAttrs)
		}

		if evalErr != nil {
			a.logger.Warn("image eval failed",
				zap.String("provider", a.inner.ProviderName()),
				zap.String("use_case", useCase),
				zap.Error(evalErr),
			)
			if a.hook != nil {
				a.hook(prompt, nil, evalErr)
			}
			return
		}

		a.logger.Info("image eval complete",
			zap.String("provider", a.inner.ProviderName()),
			zap.String("use_case", useCase),
			zap.Int("prompt_adherence", score.PromptAdherence),
			zap.Int("artifact_score", score.ArtifactScore),
			zap.Int("composition_score", score.CompositionScore),
			zap.Int("overall_score", score.OverallScore),
			zap.Bool("suggested_retry", score.SuggestedRetry),
		)

		if a.metrics != nil {
			metricAttrs := metric.WithAttributes(
				attribute.String("provider", a.inner.ProviderName()),
				attribute.String("use_case", useCase),
			)
			a.metrics.OverallScore.Record(evalCtx, float64(score.OverallScore), metricAttrs)
			if score.SuggestedRetry {
				a.metrics.SuggestedRetry.Add(evalCtx, 1, metricAttrs)
			}
		}

		evalScore := &providers.EvalScore{
			PromptAdherence:  score.PromptAdherence,
			ArtifactScore:    score.ArtifactScore,
			CompositionScore: score.CompositionScore,
			OverallScore:     score.OverallScore,
			Reasoning:        score.Reasoning,
			SuggestedRetry:   score.SuggestedRetry,
		}

		if jobID, ok := JobIDFromContext(evalCtx); ok {
			data := map[string]any{
				"eval_score": map[string]any{
					"prompt_adherence":  evalScore.PromptAdherence,
					"artifact_score":    evalScore.ArtifactScore,
					"composition_score": evalScore.CompositionScore,
					"overall_score":     evalScore.OverallScore,
					"suggested_retry":   evalScore.SuggestedRetry,
					"reasoning":         evalScore.Reasoning,
				},
			}
			if err := a.persister.AppendJobResultData(evalCtx, jobID, data); err != nil {
				a.logger.Warn("failed to persist eval score",
					zap.String("job_id", jobID),
					zap.Error(err),
				)
			}
		}

		if a.hook != nil {
			a.hook(prompt, evalScore, nil)
		}
	})

	return result, nil
}
