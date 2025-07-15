package utils

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	// Level sets the minimum log level (debug, info, warn, error, fatal, panic)
	Level string
	// Pretty enables pretty console output for development
	Pretty bool
	// CallerInfo adds file and line number to logs
	CallerInfo bool
	// LogFile specifies the log file path (empty means stderr)
	LogFile string
}

// NewLogger creates a new logger instance with the given configuration
func NewLogger(config LoggerConfig) zerolog.Logger {
	// Parse log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Create output writer
	var output io.Writer
	
	// Determine output destination
	if config.LogFile != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(config.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			// Fall back to stderr if file creation fails
			output = os.Stderr
		} else {
			// Open log file
			file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				// Fall back to stderr if file opening fails
				output = os.Stderr
			} else {
				output = file
			}
		}
	} else {
		// Default to stderr
		output = os.Stderr
	}
	
	// Apply pretty formatting if requested (only for stderr)
	if config.Pretty && config.LogFile == "" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			FieldsExclude: []string{
				zerolog.TimestampFieldName,
			},
		}
	}

	// Create logger
	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	// Add caller information if enabled
	if config.CallerInfo {
		logger = logger.With().Caller().Logger()
	}

	return logger
}

// SetupGlobalLogger sets up the global logger with the given configuration
func SetupGlobalLogger(config LoggerConfig) {
	logger := NewLogger(config)
	log.Logger = logger
}

// WithContext adds the logger to the context
func WithContext(ctx context.Context, logger zerolog.Logger) context.Context {
	return logger.WithContext(ctx)
}

// FromContext retrieves the logger from the context
// If no logger is found, returns the global logger
func FromContext(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}

// WithField adds a field to the logger
func WithField(logger zerolog.Logger, key string, value interface{}) zerolog.Logger {
	return logger.With().Interface(key, value).Logger()
}

// WithFields adds multiple fields to the logger
func WithFields(logger zerolog.Logger, fields map[string]interface{}) zerolog.Logger {
	context := logger.With()
	for key, value := range fields {
		context = context.Interface(key, value)
	}
	return context.Logger()
}

// WithError adds an error field to the logger
func WithError(logger zerolog.Logger, err error) zerolog.Logger {
	return logger.With().Err(err).Logger()
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "info",
		Pretty:     false,
		CallerInfo: false,
	}
}

// DevelopmentConfig returns a logger configuration suitable for development
func DevelopmentConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "debug",
		Pretty:     true,
		CallerInfo: true,
	}
}

// ProductionConfig returns a logger configuration suitable for production
func ProductionConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "info",
		Pretty:     false,
		CallerInfo: false,
	}
}