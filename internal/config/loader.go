package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set config type
	v.SetConfigType("yaml")

	// Set config name
	v.SetConfigName("config")

	// Add config search paths
	if configPath != "" {
		// Use explicit path if provided
		v.SetConfigFile(configPath)
	} else {
		// Search in multiple locations
		v.AddConfigPath(".")              // Current directory
		v.AddConfigPath("./config")       // Config subdirectory
		v.AddConfigPath("/etc/remember-me-mcp") // System config directory

		// Also check home directory
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".remember-me-mcp"))
		}
	}

	// Set defaults (these will be overridden by config file and env vars)
	setDefaults(v)

	// Configure environment variable handling
	v.SetEnvPrefix("REMEMBER_ME")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific environment variables
	bindEnvVars(v)

	// Read configuration file (if exists)
	if err := v.ReadInConfig(); err != nil {
		// It's ok if config file doesn't exist, we have defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Handle DATABASE_URL environment variable specially
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		// Parse DATABASE_URL and override individual database settings
		if err := parseDatabaseURL(v, dbURL); err != nil {
			return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
		}
	}

	// Unmarshal configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "remember_me")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_connections", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.conn_max_idle_time", "10m")

	// OpenAI defaults
	v.SetDefault("openai.model", "text-embedding-3-small")
	v.SetDefault("openai.max_retries", 3)
	v.SetDefault("openai.timeout", 30)

	// Memory defaults
	v.SetDefault("memory.max_memories", 1000)
	v.SetDefault("memory.similarity_threshold", 0.7)

	// Server defaults
	v.SetDefault("server.log_level", "info")
	v.SetDefault("server.debug", false)
}

// bindEnvVars binds specific environment variables to configuration keys
func bindEnvVars(v *viper.Viper) {
	// OpenAI API key can be set via OPENAI_API_KEY or REMEMBER_ME_OPENAI_API_KEY
	v.BindEnv("openai.api_key", "OPENAI_API_KEY", "REMEMBER_ME_OPENAI_API_KEY")

	// Log level can be set via LOG_LEVEL or REMEMBER_ME_SERVER_LOG_LEVEL
	v.BindEnv("server.log_level", "LOG_LEVEL", "REMEMBER_ME_SERVER_LOG_LEVEL")

	// Memory limit can be set via MEMORY_LIMIT or REMEMBER_ME_MEMORY_MAX_MEMORIES
	v.BindEnv("memory.max_memories", "MEMORY_LIMIT", "REMEMBER_ME_MEMORY_MAX_MEMORIES")

	// Debug mode
	v.BindEnv("server.debug", "DEBUG", "REMEMBER_ME_SERVER_DEBUG")
}

// parseDatabaseURL parses a PostgreSQL connection URL and sets individual database config values
func parseDatabaseURL(v *viper.Viper, dbURL string) error {
	// Basic parsing of postgres://user:password@host:port/dbname?sslmode=disable
	// This is a simplified parser - in production, you might want to use a proper URL parser

	if !strings.HasPrefix(dbURL, "postgres://") && !strings.HasPrefix(dbURL, "postgresql://") {
		return fmt.Errorf("URL must start with postgres:// or postgresql://")
	}

	// Remove the scheme
	dbURL = strings.TrimPrefix(dbURL, "postgres://")
	dbURL = strings.TrimPrefix(dbURL, "postgresql://")

	// Split by @
	parts := strings.SplitN(dbURL, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid URL format")
	}

	// Parse user:password
	userParts := strings.SplitN(parts[0], ":", 2)
	if len(userParts) > 0 {
		v.Set("database.user", userParts[0])
	}
	if len(userParts) > 1 {
		v.Set("database.password", userParts[1])
	}

	// Parse host:port/dbname?params
	remaining := parts[1]

	// Extract query parameters
	var queryParams string
	if idx := strings.Index(remaining, "?"); idx != -1 {
		queryParams = remaining[idx+1:]
		remaining = remaining[:idx]
	}

	// Parse host:port/dbname
	hostDBParts := strings.SplitN(remaining, "/", 2)
	if len(hostDBParts) != 2 {
		return fmt.Errorf("database name not found in URL")
	}

	// Parse host:port
	hostParts := strings.SplitN(hostDBParts[0], ":", 2)
	v.Set("database.host", hostParts[0])
	if len(hostParts) > 1 {
		v.Set("database.port", hostParts[1])
	}

	// Set database name
	v.Set("database.dbname", hostDBParts[1])

	// Parse query parameters
	if queryParams != "" {
		params := strings.Split(queryParams, "&")
		for _, param := range params {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) == 2 && kv[0] == "sslmode" {
				v.Set("database.sslmode", kv[1])
			}
		}
	}

	return nil
}

// LoadConfigOrDefault loads configuration or returns default if loading fails
func LoadConfigOrDefault(configPath string) *Config {
	config, err := LoadConfig(configPath)
	if err != nil {
		// Log the error but return default config
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v. Using defaults.\n", err)
		return NewDefault()
	}
	return config
}