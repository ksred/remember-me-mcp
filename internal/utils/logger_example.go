// +build ignore

package main

import (
	"context"
	"errors"
	"time"

	"github.com/ksred/remember-me-mcp/internal/utils"
	"github.com/rs/zerolog/log"
)

// Example demonstrates various ways to use the logger utility
func main() {
	// Example 1: Setup global logger for development
	utils.SetupGlobalLogger(utils.DevelopmentConfig())
	
	// Use the global logger
	log.Info().Msg("Application started")
	log.Debug().Str("environment", "development").Msg("Debug information")
	
	// Example 2: Create a custom logger instance
	customLogger := utils.NewLogger(utils.LoggerConfig{
		Level:      "debug",
		Pretty:     true,
		CallerInfo: true,
	})
	
	customLogger.Info().
		Str("component", "database").
		Str("action", "connect").
		Msg("Connecting to database")
	
	// Example 3: Using logger with context
	ctx := context.Background()
	ctx = utils.WithContext(ctx, customLogger)
	
	// Simulate a function that uses logger from context
	processRequest(ctx, "user-123")
	
	// Example 4: Adding fields to logger
	userLogger := utils.WithFields(customLogger, map[string]interface{}{
		"user_id":    "user-123",
		"session_id": "sess-456",
		"ip":         "192.168.1.1",
	})
	
	userLogger.Info().Msg("User logged in")
	userLogger.Warn().Msg("Invalid password attempt")
	
	// Example 5: Logging with error
	err := errors.New("connection timeout")
	errorLogger := utils.WithError(customLogger, err)
	errorLogger.Error().Msg("Failed to connect to external service")
	
	// Example 6: Structured logging
	customLogger.Info().
		Str("method", "GET").
		Str("path", "/api/users").
		Int("status", 200).
		Dur("duration", 150*time.Millisecond).
		Msg("HTTP request processed")
	
	// Example 7: Production configuration
	prodLogger := utils.NewLogger(utils.ProductionConfig())
	prodLogger.Info().
		Str("service", "remember-me-mcp").
		Str("version", "1.0.0").
		Msg("Service started in production mode")
}

// processRequest demonstrates getting logger from context
func processRequest(ctx context.Context, userID string) {
	logger := utils.FromContext(ctx)
	
	logger.Info().
		Str("user_id", userID).
		Msg("Processing user request")
	
	// Simulate some processing
	time.Sleep(100 * time.Millisecond)
	
	logger.Debug().
		Str("user_id", userID).
		Dur("processing_time", 100*time.Millisecond).
		Msg("Request processed successfully")
}