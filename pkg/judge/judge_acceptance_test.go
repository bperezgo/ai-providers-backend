//go:build acceptance

package judge

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireEnvAcceptance(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping acceptance test: %s not set", key)
	}
	return v
}

func TestAcceptance_Judge_GoodImage(t *testing.T) {
	anthropicKey := requireEnvAcceptance(t, "ANTHROPIC_API_KEY")

	httpClient := common.NewHTTPClient(60*time.Second, 2, time.Second)
	j := NewJudge(anthropicKey, httpClient, "")

	score, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "a beautiful sunset over the ocean with orange and purple clouds",
		ImagePath:      testdataPath("fixture_good.png"),
		UseCase:        "social-post",
	})
	require.NoError(t, err)

	assert.GreaterOrEqual(t, score.PromptAdherence, 1)
	assert.LessOrEqual(t, score.PromptAdherence, 5)
	assert.GreaterOrEqual(t, score.ArtifactScore, 1)
	assert.LessOrEqual(t, score.ArtifactScore, 5)
	assert.GreaterOrEqual(t, score.CompositionScore, 1)
	assert.LessOrEqual(t, score.CompositionScore, 5)
	assert.GreaterOrEqual(t, score.OverallScore, 1)
	assert.LessOrEqual(t, score.OverallScore, 5)
	assert.NotEmpty(t, score.Reasoning)

	// A known-good image should score reasonably well
	assert.GreaterOrEqual(t, score.OverallScore, 3, "good fixture should score at least 3")
	assert.False(t, score.SuggestedRetry, "good fixture should not suggest retry")

	t.Logf("Good image scores: adherence=%d artifact=%d composition=%d overall=%d retry=%v",
		score.PromptAdherence, score.ArtifactScore, score.CompositionScore,
		score.OverallScore, score.SuggestedRetry)
	t.Logf("Reasoning: %s", score.Reasoning)
}

func TestAcceptance_Judge_BadImage(t *testing.T) {
	anthropicKey := requireEnvAcceptance(t, "ANTHROPIC_API_KEY")

	httpClient := common.NewHTTPClient(60*time.Second, 2, time.Second)
	j := NewJudge(anthropicKey, httpClient, "")

	score, err := j.Evaluate(context.Background(), ImageEvalRequest{
		OriginalPrompt: "a professional corporate meeting in a modern office with people in suits",
		ImagePath:      testdataPath("fixture_bad.png"),
		UseCase:        "marketing-banner",
	})
	require.NoError(t, err)

	assert.GreaterOrEqual(t, score.PromptAdherence, 1)
	assert.LessOrEqual(t, score.PromptAdherence, 5)
	assert.GreaterOrEqual(t, score.ArtifactScore, 1)
	assert.LessOrEqual(t, score.ArtifactScore, 5)
	assert.GreaterOrEqual(t, score.CompositionScore, 1)
	assert.LessOrEqual(t, score.CompositionScore, 5)
	assert.GreaterOrEqual(t, score.OverallScore, 1)
	assert.LessOrEqual(t, score.OverallScore, 5)
	assert.NotEmpty(t, score.Reasoning)

	// A known-bad image should score poorly
	assert.LessOrEqual(t, score.OverallScore, 3, "bad fixture should score at most 3")
	assert.True(t, score.SuggestedRetry, "bad fixture should suggest retry")

	t.Logf("Bad image scores: adherence=%d artifact=%d composition=%d overall=%d retry=%v",
		score.PromptAdherence, score.ArtifactScore, score.CompositionScore,
		score.OverallScore, score.SuggestedRetry)
	t.Logf("Reasoning: %s", score.Reasoning)
}
