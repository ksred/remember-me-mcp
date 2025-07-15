package services

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIEmbeddingService_NewService(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("Valid configuration", func(t *testing.T) {
		cfg := &config.OpenAI{
			APIKey:     "test-api-key",
			Model:      "text-embedding-3-small",
			MaxRetries: 3,
			Timeout:    30 * time.Second,
		}

		service, err := NewOpenAIEmbeddingService(cfg, logger)
		assert.NoError(t, err)
		assert.NotNil(t, service)
		assert.Equal(t, "text-embedding-3-small", service.GetModel())
	})

	t.Run("Missing API key", func(t *testing.T) {
		cfg := &config.OpenAI{
			Model:      "text-embedding-3-small",
			MaxRetries: 3,
			Timeout:    30 * time.Second,
		}

		service, err := NewOpenAIEmbeddingService(cfg, logger)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "API key is required")
	})
}

func TestOpenAIEmbeddingService_GenerateEmbedding(t *testing.T) {
	// Skip if no API key is available
	if testing.Short() {
		t.Skip("Skipping OpenAI API test in short mode")
	}

	logger := zerolog.Nop()
	cfg := &config.OpenAI{
		APIKey:     "sk-test-key", // This would need a real key for integration tests
		Model:      "text-embedding-3-small",
		MaxRetries: 1,
		Timeout:    5 * time.Second,
	}

	service, err := NewOpenAIEmbeddingService(cfg, logger)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Empty text", func(t *testing.T) {
		embedding, err := service.GenerateEmbedding(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, embedding)
		assert.Contains(t, err.Error(), "text cannot be empty")
	})

	// Additional integration tests would require a valid API key
	// and would be run separately from unit tests
}

func TestMockEmbeddingService_GenerateEmbedding(t *testing.T) {
	service := NewMockEmbeddingService()
	ctx := context.Background()

	t.Run("Generate deterministic embeddings", func(t *testing.T) {
		// Same text should produce same embedding
		embedding1, err := service.GenerateEmbedding(ctx, "test text")
		require.NoError(t, err)
		require.NotNil(t, embedding1)
		assert.Equal(t, EmbeddingDimension, len(embedding1))

		embedding2, err := service.GenerateEmbedding(ctx, "test text")
		require.NoError(t, err)
		require.NotNil(t, embedding2)

		// Should be identical
		assert.Equal(t, embedding1, embedding2)
	})

	t.Run("Different text produces different embeddings", func(t *testing.T) {
		embedding1, err := service.GenerateEmbedding(ctx, "first text")
		require.NoError(t, err)

		embedding2, err := service.GenerateEmbedding(ctx, "second text")
		require.NoError(t, err)

		// Should be different
		assert.NotEqual(t, embedding1, embedding2)
	})

	t.Run("Empty text returns error", func(t *testing.T) {
		embedding, err := service.GenerateEmbedding(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, embedding)
	})

	t.Run("Embeddings are normalized", func(t *testing.T) {
		embedding, err := service.GenerateEmbedding(ctx, "normalize test")
		require.NoError(t, err)

		// Calculate magnitude
		var magnitudeSquared float32
		for _, v := range embedding {
			magnitudeSquared += v * v
		}
		magnitude := float32(math.Sqrt(float64(magnitudeSquared)))

		// Should be close to 1.0 (unit vector)
		assert.InDelta(t, 1.0, magnitude, 0.001)
	})
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "Identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "Orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "Opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity, err := CosineSimilarity(tt.a, tt.b)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, similarity, 0.001)
		})
	}
}

