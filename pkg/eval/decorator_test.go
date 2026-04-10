package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	portmocks "github.com/bryanperez/laguna-escondida-marketing/backend/internal/port/mocks"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/providers"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/common"
	"github.com/bryanperez/laguna-escondida-marketing/backend/pkg/judge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeProvider is a minimal TextToImageProvider for testing the decorator.
type fakeProvider struct {
	imagePath string
}

func (f *fakeProvider) ProviderName() string { return "fake" }

func (f *fakeProvider) GenerateImage(_ context.Context, _ string, _ providers.TextToImageOptions) (*providers.ImageResult, error) {
	return &providers.ImageResult{
		LocalPath:   f.imagePath,
		ProviderURL: "https://cdn.fake.com/image.png",
		Width:       1024,
		Height:      1024,
		MimeType:    "image/png",
	}, nil
}

func writeTempPNG(t *testing.T) string {
	t.Helper()
	// Minimal valid-ish PNG (just needs to be readable bytes for base64 encoding)
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	p := filepath.Join(t.TempDir(), "test.png")
	require.NoError(t, os.WriteFile(p, data, 0644))
	return p
}

func newTestHTTPClient() *common.HTTPClient {
	return common.NewHTTPClient(10*time.Second, 0, time.Second)
}

func mockJudgeServer(t *testing.T, score judge.ImageEvalScore) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scoreJSON, _ := json.Marshal(score)
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(scoreJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestEvalDecorator_AlwaysEvaluate(t *testing.T) {
	imgPath := writeTempPNG(t)
	goodScore := judge.ImageEvalScore{
		PromptAdherence: 5, ArtifactScore: 5, CompositionScore: 4,
		OverallScore: 5, Reasoning: "looks great", SuggestedRetry: false,
	}

	server := mockJudgeServer(t, goodScore)
	defer server.Close()

	j := judge.NewJudgeWithURL("test-key", newTestHTTPClient(), server.URL)
	persister := portmocks.NewMockEvalPersister(t) // no job ID in ctx → AppendJobResultData not called

	var capturedScore *providers.EvalScore
	decorator := NewEvalDecorator(&fakeProvider{imagePath: imgPath}, j, AlwaysEvaluate{}, "social-post", zap.NewNop(), persister,
		WithHook(func(_ string, score *providers.EvalScore, _ error) {
			capturedScore = score
		}),
	)

	// Verify it satisfies the interface
	var _ providers.TextToImageProvider = decorator

	result, err := decorator.GenerateImage(context.Background(), "a sunset", providers.TextToImageOptions{})
	require.NoError(t, err)

	// Image is returned immediately — EvalScore is always nil in the response now
	assert.Equal(t, "fake", decorator.ProviderName())
	assert.Nil(t, result.EvalScore, "EvalScore is delivered async via hook, not in response")

	// Wait for the background goroutine then assert via hook capture
	decorator.WaitForPendingEvals()
	require.NotNil(t, capturedScore)
	assert.Equal(t, 5, capturedScore.PromptAdherence)
	assert.Equal(t, 5, capturedScore.OverallScore)
	assert.False(t, capturedScore.SuggestedRetry)
}

func TestEvalDecorator_PersistsEvalScore(t *testing.T) {
	imgPath := writeTempPNG(t)
	goodScore := judge.ImageEvalScore{
		PromptAdherence: 4, ArtifactScore: 5, CompositionScore: 4,
		OverallScore: 4, Reasoning: "solid image", SuggestedRetry: false,
	}

	server := mockJudgeServer(t, goodScore)
	defer server.Close()

	j := judge.NewJudgeWithURL("test-key", newTestHTTPClient(), server.URL)
	persister := portmocks.NewMockEvalPersister(t)
	persister.EXPECT().AppendJobResultData(mock.Anything, "job-42", mock.Anything).Return(nil)

	decorator := NewEvalDecorator(&fakeProvider{imagePath: imgPath}, j, AlwaysEvaluate{}, "social-post", zap.NewNop(), persister)

	ctx := WithJobID(context.Background(), "job-42")
	_, err := decorator.GenerateImage(ctx, "a sunset", providers.TextToImageOptions{})
	require.NoError(t, err)

	decorator.WaitForPendingEvals()
	// mock.AssertExpectations enforced by NewMockEvalPersister cleanup
}

func TestEvalDecorator_NeverEvaluate(t *testing.T) {
	imgPath := writeTempPNG(t)

	// No mock server needed — judge should never be called
	j := judge.NewJudge("unused-key", newTestHTTPClient(), "")
	persister := portmocks.NewMockEvalPersister(t) // eval skipped → persister never called

	decorator := NewEvalDecorator(&fakeProvider{imagePath: imgPath}, j, NeverEvaluate{}, "social-post", zap.NewNop(), persister)

	result, err := decorator.GenerateImage(context.Background(), "a sunset", providers.TextToImageOptions{})
	require.NoError(t, err)
	assert.Nil(t, result.EvalScore)
}

func TestEvalDecorator_SampleEvaluate(t *testing.T) {
	imgPath := writeTempPNG(t)
	score := judge.ImageEvalScore{
		PromptAdherence: 4, ArtifactScore: 4, CompositionScore: 4,
		OverallScore: 4, Reasoning: "good", SuggestedRetry: false,
	}

	server := mockJudgeServer(t, score)
	defer server.Close()

	j := judge.NewJudgeWithURL("test-key", newTestHTTPClient(), server.URL)
	persister := portmocks.NewMockEvalPersister(t) // no job ID in ctx → persister never called
	strategy := NewSampleEvaluate(3)               // evaluate every 3rd request

	var evalCount atomic.Int32
	decorator := NewEvalDecorator(&fakeProvider{imagePath: imgPath}, j, strategy, "marketing-banner", zap.NewNop(), persister,
		WithHook(func(_ string, s *providers.EvalScore, err error) {
			if err == nil && s != nil {
				evalCount.Add(1)
			}
		}),
	)

	for range 9 {
		_, err := decorator.GenerateImage(context.Background(), "test", providers.TextToImageOptions{})
		require.NoError(t, err)
	}

	decorator.WaitForPendingEvals()
	assert.Equal(t, int32(3), evalCount.Load(), "should evaluate exactly 3 out of 9 requests (every 3rd)")
}

func TestSampleEvaluate_pattern(t *testing.T) {
	s := NewSampleEvaluate(5)

	var results []bool
	for range 15 {
		results = append(results, s.ShouldEvaluate())
	}

	// Should be true at positions 4, 9, 14 (every 5th call: count 5, 10, 15)
	for i, v := range results {
		if (i+1)%5 == 0 {
			assert.True(t, v, "expected true at call %d", i+1)
		} else {
			assert.False(t, v, "expected false at call %d", i+1)
		}
	}
}
