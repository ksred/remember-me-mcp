package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
)

const (
	// EmbeddingDimension is the dimension of the embedding vector (OpenAI's text-embedding-ada-002)
	EmbeddingDimension = 1536
)

// EmbeddingService defines the interface for generating text embeddings
type EmbeddingService interface {
	// GenerateEmbedding generates an embedding vector for the given text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// MockEmbeddingService is a mock implementation of EmbeddingService for testing
type MockEmbeddingService struct{}

// NewMockEmbeddingService creates a new mock embedding service
func NewMockEmbeddingService() *MockEmbeddingService {
	return &MockEmbeddingService{}
}

// GenerateEmbedding generates a deterministic embedding based on text hash
func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	
	// Generate a deterministic hash of the text
	hash := sha256.Sum256([]byte(text))
	
	// Create a 1536-dimensional vector
	embedding := make([]float32, EmbeddingDimension)
	
	// Use the hash to generate deterministic values
	for i := 0; i < EmbeddingDimension; i++ {
		// Use different parts of the hash for different dimensions
		hashIndex := i % len(hash)
		
		// Convert byte to float in range [-1, 1]
		// This creates a deterministic but pseudo-random distribution
		value := float64(hash[hashIndex]) / 127.5 - 1.0
		
		// Add some variation based on position
		if i > 0 {
			// Mix in the previous value for better distribution
			prevValue := float64(embedding[i-1])
			value = (value + prevValue*0.3) / 1.3
		}
		
		// Apply a sine transformation for more natural distribution
		value = math.Sin(value * math.Pi)
		
		// Ensure the value is in range [-1, 1]
		if value > 1.0 {
			value = 1.0
		} else if value < -1.0 {
			value = -1.0
		}
		
		embedding[i] = float32(value)
	}
	
	// Normalize the vector to unit length (common for embeddings)
	magnitude := float32(0)
	for _, v := range embedding {
		magnitude += v * v
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))
	
	if magnitude > 0 {
		for i := range embedding {
			embedding[i] /= magnitude
		}
	}
	
	return embedding, nil
}

// Ensure MockEmbeddingService implements EmbeddingService
var _ EmbeddingService = (*MockEmbeddingService)(nil)

// Helper function to calculate cosine similarity between two embeddings
func CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, nil
	}
	
	var dotProduct, magnitudeA, magnitudeB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		magnitudeA += a[i] * a[i]
		magnitudeB += b[i] * b[i]
	}
	
	magnitudeA = float32(math.Sqrt(float64(magnitudeA)))
	magnitudeB = float32(math.Sqrt(float64(magnitudeB)))
	
	if magnitudeA == 0 || magnitudeB == 0 {
		return 0, nil
	}
	
	return dotProduct / (magnitudeA * magnitudeB), nil
}