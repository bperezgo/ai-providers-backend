//go:build acceptance

package nanobanana

import (
	"context"
	"os"
	"testing"
	"time"

	portmocks "github.com/bryanperez/laguna-escondida-marketing/backend/internal/port/mocks"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/eval"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/judge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func requireEnvAcceptance(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func TestAcceptance_FullPipeline_GenerateAndEvaluate(t *testing.T) {
	falKey := requireEnvAcceptance(t, "FAL_API_KEY")
	anthropicKey := requireEnvAcceptance(t, "ANTHROPIC_API_KEY")

	outputDir := t.TempDir()

	httpClient := common.NewHTTPClient(120*time.Second, 2, 2*time.Second)
	nbClient := NewClient(falKey, httpClient, outputDir, "")

	judgeHTTP := common.NewHTTPClient(60*time.Second, 2, time.Second)
	judgeClient := judge.NewJudge(anthropicKey, judgeHTTP, "")

	logger, _ := zap.NewDevelopment()
	persister := portmocks.NewMockEvalPersister(t) // no job ID in ctx → AppendJobResultData not called

	// Capture eval result via hook — eval runs async after GenerateImage returns.
	var capturedScore *providers.EvalScore
	decorator := eval.NewEvalDecorator(nbClient, judgeClient, eval.AlwaysEvaluate{}, "marketing", logger, persister,
		eval.WithHook(func(_ string, score *providers.EvalScore, err error) {
			if err == nil {
				capturedScore = score
			} else {
				t.Logf("eval hook error: %v", err)
			}
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	result, err := decorator.GenerateImage(ctx, "A single red fish on a white background", providers.TextToImageOptions{
		Size: "square",
		N:    1,
	})
	require.NoError(t, err)

	// Image was generated and saved immediately (eval is async)
	assert.NotEmpty(t, result.LocalPath)
	assert.FileExists(t, result.LocalPath)
	assert.NotEmpty(t, result.ProviderURL)
	assert.Greater(t, result.Width, 0)
	assert.Greater(t, result.Height, 0)

	// Wait for async eval goroutine to complete, then assert via captured hook value
	decorator.WaitForPendingEvals()
	require.NotNil(t, capturedScore, "AlwaysEvaluate should produce an EvalScore via hook")
	assert.GreaterOrEqual(t, capturedScore.PromptAdherence, 1)
	assert.LessOrEqual(t, capturedScore.PromptAdherence, 5)
	assert.GreaterOrEqual(t, capturedScore.ArtifactScore, 1)
	assert.LessOrEqual(t, capturedScore.ArtifactScore, 5)
	assert.GreaterOrEqual(t, capturedScore.CompositionScore, 1)
	assert.LessOrEqual(t, capturedScore.CompositionScore, 5)
	assert.GreaterOrEqual(t, capturedScore.OverallScore, 1)
	assert.LessOrEqual(t, capturedScore.OverallScore, 5)
	assert.NotEmpty(t, capturedScore.Reasoning)

	t.Logf("Generated image: %s (%dx%d)", result.LocalPath, result.Width, result.Height)
	t.Logf("Eval scores: adherence=%d artifact=%d composition=%d overall=%d retry=%v",
		capturedScore.PromptAdherence, capturedScore.ArtifactScore,
		capturedScore.CompositionScore, capturedScore.OverallScore,
		capturedScore.SuggestedRetry)
	t.Logf("Reasoning: %s", capturedScore.Reasoning)
}

func TestAcceptance_FullPipeline_NeverEvaluate(t *testing.T) {
	falKey := requireEnvAcceptance(t, "FAL_API_KEY")
	anthropicKey := requireEnvAcceptance(t, "ANTHROPIC_API_KEY")

	outputDir := t.TempDir()

	httpClient := common.NewHTTPClient(120*time.Second, 2, 2*time.Second)
	nbClient := NewClient(falKey, httpClient, outputDir, "")

	judgeHTTP := common.NewHTTPClient(60*time.Second, 2, time.Second)
	judgeClient := judge.NewJudge(anthropicKey, judgeHTTP, "")

	logger, _ := zap.NewDevelopment()
	persister := portmocks.NewMockEvalPersister(t) // eval skipped → persister never called

	decorator := eval.NewEvalDecorator(nbClient, judgeClient, eval.NeverEvaluate{}, "marketing", logger, persister)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	result, err := decorator.GenerateImage(ctx, "A green tree on a blue sky", providers.TextToImageOptions{
		Size: "square",
		N:    1,
	})
	require.NoError(t, err)

	// Image was generated successfully
	assert.NotEmpty(t, result.LocalPath)
	assert.FileExists(t, result.LocalPath)

	// NeverEvaluate should skip the judge entirely
	assert.Nil(t, result.EvalScore, "NeverEvaluate should not produce an EvalScore")

	t.Logf("Generated image (no eval): %s (%dx%d)", result.LocalPath, result.Width, result.Height)
}
