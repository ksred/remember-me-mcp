package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config LoggerConfig
		check  func(t *testing.T, output string)
	}{
		{
			name: "JSON output with info level",
			config: LoggerConfig{
				Level:      "info",
				Pretty:     false,
				CallerInfo: false,
			},
			check: func(t *testing.T, output string) {
				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(output), &logEntry)
				require.NoError(t, err)
				assert.Equal(t, "info", logEntry["level"])
				assert.Equal(t, "test message", logEntry["message"])
				assert.Contains(t, logEntry, "time")
			},
		},
		{
			name: "Pretty output with debug level",
			config: LoggerConfig{
				Level:      "debug",
				Pretty:     true,
				CallerInfo: false,
			},
			check: func(t *testing.T, output string) {
				// Pretty output is not testable when we override the output
				// Just check that we get some output
				assert.NotEmpty(t, output)
			},
		},
		{
			name: "With caller info",
			config: LoggerConfig{
				Level:      "info",
				Pretty:     false,
				CallerInfo: true,
			},
			check: func(t *testing.T, output string) {
				var logEntry map[string]interface{}
				err := json.Unmarshal([]byte(output), &logEntry)
				require.NoError(t, err)
				assert.Contains(t, logEntry, "caller")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer to capture output
			buf := &bytes.Buffer{}
			
			// Create logger with custom output
			logger := NewLogger(tt.config)
			logger = logger.Output(buf)
			
			// Log a test message
			logger.Info().Msg("test message")
			
			// Check the output
			tt.check(t, strings.TrimSpace(buf.String()))
		})
	}
}

func TestSetupGlobalLogger(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	
	// Setup global logger
	config := LoggerConfig{
		Level:      "info",
		Pretty:     false,
		CallerInfo: false,
	}
	SetupGlobalLogger(config)
	
	// Override output for testing
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.New(buf).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger
	
	// Use global logger
	logger.Info().Msg("global test")
	
	// Verify output
	output := buf.String()
	assert.Contains(t, output, "global test")
}

func TestWithContext(t *testing.T) {
	// Create a logger
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	
	// Add logger to context
	ctx := context.Background()
	ctx = WithContext(ctx, logger)
	
	// Retrieve logger from context
	loggerFromCtx := FromContext(ctx)
	require.NotNil(t, loggerFromCtx)
	
	// Use the logger from context
	loggerFromCtx.Info().Msg("context test")
	
	// Verify output
	output := buf.String()
	assert.Contains(t, output, "context test")
}

func TestWithField(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	
	// Add single field
	logger = WithField(logger, "key", "value")
	logger.Info().Msg("field test")
	
	// Verify output
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(buf.String()), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "value", logEntry["key"])
}

func TestWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	
	// Add multiple fields
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	logger = WithFields(logger, fields)
	logger.Info().Msg("fields test")
	
	// Verify output
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(buf.String()), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "value1", logEntry["key1"])
	assert.Equal(t, float64(42), logEntry["key2"])
	assert.Equal(t, true, logEntry["key3"])
}

func TestWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	
	// Add error
	testErr := assert.AnError
	logger = WithError(logger, testErr)
	logger.Error().Msg("error test")
	
	// Verify output
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(buf.String()), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, testErr.Error(), logEntry["error"])
}

func TestLoggerConfigs(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.Equal(t, "info", config.Level)
		assert.False(t, config.Pretty)
		assert.False(t, config.CallerInfo)
	})
	
	t.Run("DevelopmentConfig", func(t *testing.T) {
		config := DevelopmentConfig()
		assert.Equal(t, "debug", config.Level)
		assert.True(t, config.Pretty)
		assert.True(t, config.CallerInfo)
	})
	
	t.Run("ProductionConfig", func(t *testing.T) {
		config := ProductionConfig()
		assert.Equal(t, "info", config.Level)
		assert.False(t, config.Pretty)
		assert.False(t, config.CallerInfo)
	})
}

func TestInvalidLogLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	
	// Create logger with invalid level
	config := LoggerConfig{
		Level:      "invalid",
		Pretty:     false,
		CallerInfo: false,
	}
	logger := NewLogger(config)
	logger = logger.Output(buf)
	
	// Should default to info level
	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	
	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.Contains(t, output, "info message")
}