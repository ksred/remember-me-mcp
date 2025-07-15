package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/rs/zerolog"
	"github.com/sashabaranov/go-openai"
)

// Ensure OpenAIEmbeddingService implements EmbeddingService
var _ EmbeddingService = (*OpenAIEmbeddingService)(nil)

// OpenAIEmbeddingService implements the EmbeddingService interface using OpenAI API
type OpenAIEmbeddingService struct {
	client *openai.Client
	config *config.OpenAI
	logger zerolog.Logger
}

// NewOpenAIEmbeddingService creates a new OpenAI embedding service
func NewOpenAIEmbeddingService(cfg *config.OpenAI, logger zerolog.Logger) (*OpenAIEmbeddingService, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(cfg.APIKey)

	service := &OpenAIEmbeddingService{
		client: client,
		config: cfg,
		logger: logger.With().Str("service", "openai_embedding").Logger(),
	}

	// Validate API key on startup
	go service.validateAPIKeyAsync()

	return service, nil
}

// validateAPIKeyAsync validates the OpenAI API key on startup
func (s *OpenAIEmbeddingService) validateAPIKeyAsync() {
	s.logger.Info().Msg("Validating OpenAI API key...")
	
	// Test with a simple embedding request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	_, err := s.generateEmbeddingDirect(ctx, "test")
	if err != nil {
		s.logger.Error().Err(err).Msg("OpenAI API key validation failed")
	} else {
		s.logger.Info().Msg("OpenAI API key validation successful")
	}
}

// generateEmbeddingDirect makes a direct HTTP request to OpenAI API
func (s *OpenAIEmbeddingService) generateEmbeddingDirect(ctx context.Context, text string) ([]float32, error) {
	// Create HTTP request
	reqBody := map[string]interface{}{
		"model": s.config.Model,
		"input": []string{text},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	
	// Convert to float32
	embedding := response.Data[0].Embedding
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}
	
	return result, nil
}

// GenerateEmbedding generates embeddings for the given text using OpenAI API
func (s *OpenAIEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Use direct HTTP approach to avoid any OpenAI client context issues
	s.logger.Debug().
		Str("model", s.config.Model).
		Int("text_length", len(text)).
		Dur("config_timeout", s.config.Timeout).
		Msg("Generating embedding with direct HTTP")

	// Force a longer timeout - ignore config timeout which might be too short
	timeout := 60 * time.Second
	s.logger.Debug().Dur("using_timeout", timeout).Msg("Using timeout for embedding generation")
	freshCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Retry logic
	var lastErr error
	maxRetries := s.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			s.logger.Debug().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Retrying after backoff")
			
			select {
			case <-time.After(backoff):
			case <-freshCtx.Done():
				return nil, freshCtx.Err()
			}
		}

		s.logger.Debug().
			Int("attempt", attempt+1).
			Msg("Making direct HTTP call to OpenAI API")

		start := time.Now()
		result, err := s.generateEmbeddingDirect(freshCtx, text)
		duration := time.Since(start)
		if err != nil {
			lastErr = err
			s.logger.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Dur("duration", duration).
				Msg("Failed to generate embedding")
			
			// Check if error is retryable
			if !isRetryableError(err) {
				return nil, fmt.Errorf("non-retryable error: %w", err)
			}
			continue
		}

		// Log success
		s.logger.Debug().
			Int("dimensions", len(result)).
			Int("attempts", attempt+1).
			Dur("duration", duration).
			Msg("Successfully generated embedding")

		return result, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// In a real implementation, you would check for specific error types
	// For now, we'll retry on any error except context cancellation
	return err != context.Canceled && err != context.DeadlineExceeded
}

// GetModel returns the configured model name
func (s *OpenAIEmbeddingService) GetModel() string {
	return s.config.Model
}

// ValidateAPIKey checks if the API key is valid by making a test request
func (s *OpenAIEmbeddingService) ValidateAPIKey(ctx context.Context) error {
	// Try to generate an embedding for a simple test string
	_, err := s.GenerateEmbedding(ctx, "test")
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}
	return nil
}