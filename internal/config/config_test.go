package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid configuration",
			config: func() Config {
				c := *NewDefault()
				c.OpenAI.APIKey = "test-api-key"
				return c
			}(),
			wantErr: false,
		},
		{
			name: "Missing database host",
			config: Config{
				Database: Database{
					Port:     5432,
					User:     "test",
					DBName:   "test",
					Password: "test",
				},
				OpenAI: OpenAI{
					APIKey: "test-key",
					Model:  "text-embedding-3-small",
				},
			},
			wantErr: true,
			errMsg:  "database host is required",
		},
		{
			name: "Missing database user",
			config: Config{
				Database: Database{
					Host:   "localhost",
					Port:   5432,
					DBName: "test",
				},
				OpenAI: OpenAI{
					APIKey: "test-key",
					Model:  "text-embedding-3-small",
				},
			},
			wantErr: true,
			errMsg:  "database user is required",
		},
		{
			name: "Missing database name",
			config: Config{
				Database: Database{
					Host: "localhost",
					Port: 5432,
					User: "test",
				},
				OpenAI: OpenAI{
					APIKey: "test-key",
					Model:  "text-embedding-3-small",
				},
			},
			wantErr: true,
			errMsg:  "database name is required",
		},
		{
			name: "Missing OpenAI model",
			config: Config{
				Database: Database{
					Host:           "localhost",
					Port:           5432,
					User:           "test",
					DBName:         "test",
					MaxConnections: 25,
				},
				OpenAI: OpenAI{
					APIKey:  "test-key",
					Model:   "",
					Timeout: 30 * time.Second,
				},
				Memory: Memory{
					MaxMemories: 1000,
				},
			},
			wantErr: true,
			errMsg:  "OpenAI model is required",
		},
		{
			name: "Invalid database port",
			config: Config{
				Database: Database{
					Host:   "localhost",
					Port:   -1,
					User:   "test",
					DBName: "test",
				},
				OpenAI: OpenAI{
					APIKey: "test-key",
					Model:  "text-embedding-3-small",
				},
			},
			wantErr: true,
			errMsg:  "database port must be between 1 and 65535",
		},
		{
			name: "Invalid max connections",
			config: Config{
				Database: Database{
					Host:           "localhost",
					Port:           5432,
					User:           "test",
					DBName:         "test",
					MaxConnections: -1,
				},
				OpenAI: OpenAI{
					APIKey: "test-key",
					Model:  "text-embedding-3-small",
				},
			},
			wantErr: true,
			errMsg:  "max connections must be greater than 0",
		},
		{
			name: "Invalid max idle connections",
			config: Config{
				Database: Database{
					Host:           "localhost",
					Port:           5432,
					User:           "test",
					DBName:         "test",
					MaxConnections: 10,
					MaxIdleConns:   15,
				},
				OpenAI: OpenAI{
					APIKey: "test-key",
					Model:  "text-embedding-3-small",
				},
			},
			wantErr: true,
			errMsg:  "max idle connections cannot exceed max connections",
		},
		{
			name: "Invalid log level",
			config: Config{
				Database: Database{
					Host:           "localhost",
					Port:           5432,
					User:           "test",
					DBName:         "test",
					MaxConnections: 25,
				},
				OpenAI: OpenAI{
					APIKey:  "test-key",
					Model:   "text-embedding-3-small",
					Timeout: 30 * time.Second,
				},
				Memory: Memory{
					MaxMemories: 1000,
				},
				Server: Server{
					LogLevel: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "Invalid memory limit",
			config: Config{
				Database: Database{
					Host:           "localhost",
					Port:           5432,
					User:           "test",
					DBName:         "test",
					MaxConnections: 25,
				},
				OpenAI: OpenAI{
					APIKey:  "test-key",
					Model:   "text-embedding-3-small",
					Timeout: 30 * time.Second,
				},
				Memory: Memory{
					MaxMemories: -1,
				},
			},
			wantErr: true,
			errMsg:  "max memories must be greater than 0",
		},
		{
			name: "Invalid similarity threshold",
			config: Config{
				Database: Database{
					Host:           "localhost",
					Port:           5432,
					User:           "test",
					DBName:         "test",
					MaxConnections: 25,
				},
				OpenAI: OpenAI{
					APIKey:  "test-key",
					Model:   "text-embedding-3-small",
					Timeout: 30 * time.Second,
				},
				Memory: Memory{
					MaxMemories:         1000,
					SimilarityThreshold: 1.5,
				},
			},
			wantErr: true,
			errMsg:  "similarity threshold must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_DatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		database Database
		expected string
	}{
		{
			name: "Basic URL",
			database: Database{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				DBName:   "remember_me",
				SSLMode:  "disable",
			},
			expected: "postgres://postgres:password@localhost:5432/remember_me?sslmode=disable",
		},
		{
			name: "URL with special characters in password",
			database: Database{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "p@ss!word#123",
				DBName:   "remember_me",
				SSLMode:  "disable",
			},
			expected: "postgres://postgres:p%40ss%21word%23123@localhost:5432/remember_me?sslmode=disable",
		},
		{
			name: "URL without password",
			database: Database{
				Host:    "localhost",
				Port:    5432,
				User:    "postgres",
				DBName:  "remember_me",
				SSLMode: "require",
			},
			expected: "postgres://postgres@localhost:5432/remember_me?sslmode=require",
		},
		{
			name: "Custom port",
			database: Database{
				Host:     "db.example.com",
				Port:     5433,
				User:     "dbuser",
				Password: "dbpass",
				DBName:   "mydb",
				SSLMode:  "prefer",
			},
			expected: "postgres://dbuser:dbpass@db.example.com:5433/mydb?sslmode=prefer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{Database: tt.database}
			result := config.DatabaseURL()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test configs
	tempDir := t.TempDir()

	t.Run("Load from file", func(t *testing.T) {
		// Create a test config file
		configPath := filepath.Join(tempDir, "test-config.yaml")
		configContent := `
database:
  host: testhost
  port: 5433
  user: testuser
  password: testpass
  dbname: testdb
  sslmode: require
openai:
  api_key: test-api-key
  model: text-embedding-3-small
memory:
  max_memories: 500
server:
  log_level: debug
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// Load config
		config, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify values
		assert.Equal(t, "testhost", config.Database.Host)
		assert.Equal(t, 5433, config.Database.Port)
		assert.Equal(t, "testuser", config.Database.User)
		assert.Equal(t, "testpass", config.Database.Password)
		assert.Equal(t, "testdb", config.Database.DBName)
		assert.Equal(t, "require", config.Database.SSLMode)
		assert.Equal(t, "test-api-key", config.OpenAI.APIKey)
		assert.Equal(t, 500, config.Memory.MaxMemories)
		assert.Equal(t, "debug", config.Server.LogLevel)
	})

	t.Run("Environment variable override", func(t *testing.T) {
		// Create a minimal config file
		configPath := filepath.Join(tempDir, "env-test-config.yaml")
		configContent := `
database:
  host: localhost
  user: postgres
  dbname: remember_me
openai:
  api_key: file-api-key
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// Set environment variables
		os.Setenv("REMEMBER_ME_DATABASE_HOST", "envhost")
		os.Setenv("OPENAI_API_KEY", "env-api-key")
		os.Setenv("LOG_LEVEL", "error")
		defer func() {
			os.Unsetenv("REMEMBER_ME_DATABASE_HOST")
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("LOG_LEVEL")
		}()

		// Load config
		config, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify environment variables took precedence
		assert.Equal(t, "envhost", config.Database.Host)
		assert.Equal(t, "env-api-key", config.OpenAI.APIKey)
		assert.Equal(t, "error", config.Server.LogLevel)
	})

	t.Run("DATABASE_URL parsing", func(t *testing.T) {
		// Create a minimal config file
		configPath := filepath.Join(tempDir, "db-url-config.yaml")
		configContent := `
database:
  host: localhost
  user: postgres
  dbname: remember_me
openai:
  api_key: test-key
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// Set DATABASE_URL
		os.Setenv("DATABASE_URL", "postgres://dbuser:dbpass@dbhost:5433/dbname?sslmode=require")
		defer os.Unsetenv("DATABASE_URL")

		// Load config
		config, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify DATABASE_URL was parsed correctly
		assert.Equal(t, "dbhost", config.Database.Host)
		assert.Equal(t, 5433, config.Database.Port)
		assert.Equal(t, "dbuser", config.Database.User)
		assert.Equal(t, "dbpass", config.Database.Password)
		assert.Equal(t, "dbname", config.Database.DBName)
		assert.Equal(t, "require", config.Database.SSLMode)
	})

	t.Run("Default values", func(t *testing.T) {
		// Create a minimal config with only required fields
		configPath := filepath.Join(tempDir, "minimal-config.yaml")
		configContent := `
database:
  user: postgres
  dbname: remember_me
openai:
  api_key: test-key
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// Load config
		config, err := LoadConfig(configPath)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify defaults were applied
		assert.Equal(t, "localhost", config.Database.Host)
		assert.Equal(t, 5432, config.Database.Port)
		assert.Equal(t, 25, config.Database.MaxConnections)
		assert.Equal(t, 1000, config.Memory.MaxMemories)
		assert.Equal(t, 0.7, config.Memory.SimilarityThreshold)
		assert.Equal(t, "info", config.Server.LogLevel)
		assert.Equal(t, false, config.Server.Debug)
	})

	t.Run("Invalid config file", func(t *testing.T) {
		// Create an invalid YAML file
		configPath := filepath.Join(tempDir, "invalid-config.yaml")
		configContent := `
database:
  host: [this is invalid yaml
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// Load config should fail
		config, err := LoadConfig(configPath)
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "error reading config file")
	})

	t.Run("Config file not found", func(t *testing.T) {
		// Try to load non-existent config
		config, err := LoadConfig(filepath.Join(tempDir, "non-existent.yaml"))
		
		// Should fail validation because required fields are missing
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestLoadConfigOrDefault(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		// Create a valid config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "valid-config.yaml")
		configContent := `
database:
  host: testhost
  user: testuser
  dbname: testdb
openai:
  api_key: test-key
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// Should load successfully
		config := LoadConfigOrDefault(configPath)
		assert.NotNil(t, config)
		assert.Equal(t, "testhost", config.Database.Host)
	})

	t.Run("Invalid config returns default", func(t *testing.T) {
		// Try to load non-existent config
		config := LoadConfigOrDefault("/non/existent/path.yaml")
		
		// Should return default config
		assert.NotNil(t, config)
		assert.Equal(t, "localhost", config.Database.Host)
		assert.Equal(t, 5432, config.Database.Port)
	})
}

func TestNewDefault(t *testing.T) {
	config := NewDefault()
	
	// Verify all defaults are set
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "postgres", config.Database.User)
	assert.Equal(t, "remember_me", config.Database.DBName)
	assert.Equal(t, "disable", config.Database.SSLMode)
	assert.Equal(t, 25, config.Database.MaxConnections)
	assert.Equal(t, 10, config.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, config.Database.ConnMaxLifetime)
	assert.Equal(t, 1*time.Minute, config.Database.ConnMaxIdleTime)
	
	assert.Equal(t, "", config.OpenAI.APIKey)
	assert.Equal(t, "text-embedding-3-small", config.OpenAI.Model)
	assert.Equal(t, 3, config.OpenAI.MaxRetries)
	assert.Equal(t, 30*time.Second, config.OpenAI.Timeout)
	
	assert.Equal(t, 1000, config.Memory.MaxMemories)
	assert.Equal(t, 0.7, config.Memory.SimilarityThreshold)
	
	assert.Equal(t, "info", config.Server.LogLevel)
	assert.Equal(t, false, config.Server.Debug)
	
	// Default config should validate (API key is optional)
	err := config.Validate()
	assert.NoError(t, err)
}