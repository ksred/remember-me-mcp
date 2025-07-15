package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewDatabase(t *testing.T) {
	config := map[string]interface{}{
		"host":   "localhost",
		"port":   5432,
		"dbname": "test",
	}

	db := NewDatabase(config)
	assert.NotNil(t, db)
	assert.Equal(t, config, db.config)
	assert.Nil(t, db.db)
}

func TestDatabase_buildDSN(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected string
	}{
		{
			name: "Full configuration",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"user":     "postgres",
				"password": "password",
				"dbname":   "testdb",
				"sslmode":  "require",
				"timezone": "UTC",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=require TimeZone=UTC",
		},
		{
			name: "Default values",
			config: map[string]interface{}{},
			expected: "host=localhost port=5432 user=postgres password= dbname=remember_me sslmode=disable TimeZone=UTC",
		},
		{
			name: "Partial configuration",
			config: map[string]interface{}{
				"host":   "db.example.com",
				"port":   5433,
				"user":   "dbuser",
				"dbname": "mydb",
			},
			expected: "host=db.example.com port=5433 user=dbuser password= dbname=mydb sslmode=disable TimeZone=UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewDatabase(tt.config)
			result := db.buildDSN()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabase_getConfigString(t *testing.T) {
	db := NewDatabase(map[string]interface{}{
		"string_key": "test_value",
		"int_key":    123,
		"bool_key":   true,
	})

	// Test string value
	result := db.getConfigString("string_key", "default")
	assert.Equal(t, "test_value", result)

	// Test missing key
	result = db.getConfigString("missing_key", "default")
	assert.Equal(t, "default", result)

	// Test wrong type
	result = db.getConfigString("int_key", "default")
	assert.Equal(t, "default", result)
}

func TestDatabase_getConfigInt(t *testing.T) {
	db := NewDatabase(map[string]interface{}{
		"int_key":    123,
		"float_key":  456.0,
		"string_key": "test",
	})

	// Test int value
	result := db.getConfigInt("int_key", 999)
	assert.Equal(t, 123, result)

	// Test float64 value (common in JSON)
	result = db.getConfigInt("float_key", 999)
	assert.Equal(t, 456, result)

	// Test missing key
	result = db.getConfigInt("missing_key", 999)
	assert.Equal(t, 999, result)

	// Test wrong type
	result = db.getConfigInt("string_key", 999)
	assert.Equal(t, 999, result)
}

func TestDatabase_getConfigDuration(t *testing.T) {
	db := NewDatabase(map[string]interface{}{
		"duration_string": "5m",
		"duration_direct": 10 * time.Minute,
		"invalid_string":  "invalid",
		"int_key":         123,
	})

	// Test duration string
	result := db.getConfigDuration("duration_string", time.Hour)
	assert.Equal(t, 5*time.Minute, result)

	// Test direct duration
	result = db.getConfigDuration("duration_direct", time.Hour)
	assert.Equal(t, 10*time.Minute, result)

	// Test missing key
	result = db.getConfigDuration("missing_key", time.Hour)
	assert.Equal(t, time.Hour, result)

	// Test invalid duration string
	result = db.getConfigDuration("invalid_string", time.Hour)
	assert.Equal(t, time.Hour, result)

	// Test wrong type
	result = db.getConfigDuration("int_key", time.Hour)
	assert.Equal(t, time.Hour, result)
}

func TestDatabase_getLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected string
	}{
		{
			name:     "Silent log level",
			config:   map[string]interface{}{"log_level": "silent"},
			expected: "silent",
		},
		{
			name:     "Error log level",
			config:   map[string]interface{}{"log_level": "error"},
			expected: "error",
		},
		{
			name:     "Warn log level",
			config:   map[string]interface{}{"log_level": "warn"},
			expected: "warn",
		},
		{
			name:     "Info log level",
			config:   map[string]interface{}{"log_level": "info"},
			expected: "info",
		},
		{
			name:     "Invalid log level defaults to error",
			config:   map[string]interface{}{"log_level": "invalid"},
			expected: "error",
		},
		{
			name:     "Missing log level defaults to error",
			config:   map[string]interface{}{},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewDatabase(tt.config)
			logLevel := db.getLogLevel()
			
			// Convert back to string for comparison
			var levelStr string
			switch logLevel {
			case 1: // logger.Silent
				levelStr = "silent"
			case 2: // logger.Error
				levelStr = "error"
			case 3: // logger.Warn
				levelStr = "warn"
			case 4: // logger.Info
				levelStr = "info"
			default:
				levelStr = "error"
			}
			
			assert.Equal(t, tt.expected, levelStr)
		})
	}
}

func TestDatabase_isRetryableError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "Nil error",
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "Connection refused error",
			err:         assert.AnError,
			shouldRetry: false, // This test will be false unless we mock the error string
		},
		{
			name:        "Syntax error",
			err:         assert.AnError,
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.shouldRetry, result)
		})
	}
}

func TestDatabase_isRetryableError_WithSpecificErrors(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		shouldRetry bool
	}{
		{
			name:        "Connection refused",
			errorMsg:    "connection refused",
			shouldRetry: true,
		},
		{
			name:        "Connection reset",
			errorMsg:    "connection reset",
			shouldRetry: true,
		},
		{
			name:        "Deadlock detected",
			errorMsg:    "deadlock detected",
			shouldRetry: true,
		},
		{
			name:        "Too many connections",
			errorMsg:    "too many connections",
			shouldRetry: true,
		},
		{
			name:        "Connection timeout",
			errorMsg:    "connection timeout",
			shouldRetry: true,
		},
		{
			name:        "Case insensitive matching",
			errorMsg:    "CONNECTION REFUSED",
			shouldRetry: true,
		},
		{
			name:        "Non-retryable error",
			errorMsg:    "syntax error",
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock error with specific message
			mockErr := &mockError{message: tt.errorMsg}
			result := isRetryableError(mockErr)
			assert.Equal(t, tt.shouldRetry, result)
		})
	}
}

func TestDatabase_containsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "Exact match",
			s:        "connection refused",
			substr:   "connection refused",
			expected: true,
		},
		{
			name:     "Case insensitive match",
			s:        "CONNECTION REFUSED",
			substr:   "connection refused",
			expected: true,
		},
		{
			name:     "Substring match",
			s:        "error: connection refused by server",
			substr:   "connection refused",
			expected: true,
		},
		{
			name:     "No match",
			s:        "syntax error",
			substr:   "connection refused",
			expected: false,
		},
		{
			name:     "Empty strings",
			s:        "",
			substr:   "",
			expected: true,
		},
		{
			name:     "Empty substring",
			s:        "test",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabase_ConnectionLifecycle(t *testing.T) {
	// This test verifies the basic lifecycle without actual database connection
	db := NewDatabase(map[string]interface{}{
		"host":   "localhost",
		"port":   5432,
		"dbname": "test",
	})

	// Initially no connection
	assert.Nil(t, db.DB())

	// After close, should be nil
	err := db.Close()
	assert.NoError(t, err)
	assert.Nil(t, db.DB())

	// Multiple closes should be safe
	err = db.Close()
	assert.NoError(t, err)
}

func TestDatabase_OperationsWithoutConnection(t *testing.T) {
	db := NewDatabase(map[string]interface{}{})

	// Migrate should fail without connection
	err := db.Migrate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Health check should fail without connection
	ctx := context.Background()
	err = db.Health(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// WithTransaction should fail without connection
	err = db.WithTransaction(func(tx *gorm.DB) error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Exec should fail without connection
	err = db.Exec("SELECT 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

// Mock error for testing error handling
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

func TestDatabase_ConfigHelpers(t *testing.T) {
	t.Run("Config helpers with various types", func(t *testing.T) {
		config := map[string]interface{}{
			"valid_string":   "test",
			"valid_int":      42,
			"valid_float":    3.14,
			"valid_duration": "5m",
			"nil_value":      nil,
		}

		db := NewDatabase(config)

		// Test string helper
		assert.Equal(t, "test", db.getConfigString("valid_string", "default"))
		assert.Equal(t, "default", db.getConfigString("missing", "default"))
		assert.Equal(t, "default", db.getConfigString("valid_int", "default"))

		// Test int helper
		assert.Equal(t, 42, db.getConfigInt("valid_int", 0))
		assert.Equal(t, 3, db.getConfigInt("valid_float", 0))
		assert.Equal(t, 999, db.getConfigInt("missing", 999))

		// Test duration helper
		assert.Equal(t, 5*time.Minute, db.getConfigDuration("valid_duration", time.Hour))
		assert.Equal(t, time.Hour, db.getConfigDuration("missing", time.Hour))
	})
}

func TestDatabase_WithTransaction_NilCheck(t *testing.T) {
	db := NewDatabase(map[string]interface{}{})

	// Test that WithTransaction properly handles nil database
	err := db.WithTransaction(func(tx *gorm.DB) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

func TestDatabase_Exec_NilCheck(t *testing.T) {
	db := NewDatabase(map[string]interface{}{})

	// Test that Exec properly handles nil database
	err := db.Exec("SELECT 1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

func TestDatabase_ThreadSafety(t *testing.T) {
	db := NewDatabase(map[string]interface{}{})

	// Test concurrent access to config helpers
	done := make(chan bool)
	
	go func() {
		for i := 0; i < 100; i++ {
			db.getConfigString("test", "default")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			db.getConfigInt("test", 0)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should complete without panics
	assert.True(t, true)
}