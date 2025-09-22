package config

import (
	"fmt"
	"net/url"
	"time"
)

// Config represents the main application configuration
type Config struct {
	Database   Database   `json:"database" mapstructure:"database"`
	OpenAI     OpenAI     `json:"openai" mapstructure:"openai"`
	Memory     Memory     `json:"memory" mapstructure:"memory"`
	Server     Server     `json:"server" mapstructure:"server"`
	JWT        JWT        `json:"jwt" mapstructure:"jwt"`
	HTTP       HTTP       `json:"http" mapstructure:"http"`
	Encryption Encryption `json:"encryption" mapstructure:"encryption"`
}

// Database represents database configuration
type Database struct {
	Host            string        `json:"host" mapstructure:"host"`
	Port            int           `json:"port" mapstructure:"port"`
	User            string        `json:"user" mapstructure:"user"`
	Password        string        `json:"password" mapstructure:"password"`
	DBName          string        `json:"dbname" mapstructure:"dbname"`
	SSLMode         string        `json:"sslmode" mapstructure:"sslmode"`
	MaxConnections  int           `json:"max_connections" mapstructure:"max_connections"`
	MaxIdleConns    int           `json:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
}

// OpenAI represents OpenAI API configuration
type OpenAI struct {
	APIKey     string        `json:"api_key" mapstructure:"api_key"`
	Model      string        `json:"model" mapstructure:"model"`
	MaxRetries int           `json:"max_retries" mapstructure:"max_retries"`
	Timeout    time.Duration `json:"timeout" mapstructure:"timeout"`
}

// Memory represents memory-related configuration
type Memory struct {
	MaxMemories         int     `json:"max_memories" mapstructure:"max_memories"`
	SimilarityThreshold float64 `json:"similarity_threshold" mapstructure:"similarity_threshold"`
}

// Server represents server configuration
type Server struct {
	LogLevel string `json:"log_level" mapstructure:"log_level"`
	Debug    bool   `json:"debug" mapstructure:"debug"`
}

// JWT represents JWT configuration
type JWT struct {
	Secret string `json:"secret" mapstructure:"secret"`
}

// HTTP represents HTTP server configuration  
type HTTP struct {
	Port         int      `json:"port" mapstructure:"port"`
	AllowOrigins []string `json:"allow_origins" mapstructure:"allow_origins"`
}

// Encryption represents encryption configuration
type Encryption struct {
	MasterKey string `json:"master_key" mapstructure:"master_key"`
	Enabled   bool   `json:"enabled" mapstructure:"enabled"`
}

// NewDefault returns a Config instance with default values
func NewDefault() *Config {
	return &Config{
		Database: Database{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "",
			DBName:          "postgres",
			SSLMode:         "disable",
			MaxConnections:  25,
			MaxIdleConns:    10,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		},
		OpenAI: OpenAI{
			APIKey:     "",
			Model:      "text-embedding-3-small",
			MaxRetries: 3,
			Timeout:    30 * time.Second,
		},
		Memory: Memory{
			MaxMemories:         1000,
			SimilarityThreshold: 0.7,
		},
		Server: Server{
			LogLevel: "info",
			Debug:    false,
		},
		JWT: JWT{
			Secret: "change-me-in-production",
		},
		HTTP: HTTP{
			Port: 8082,
			AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:5174"},
		},
		Encryption: Encryption{
			MasterKey: "",
			Enabled:   false,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Database validation
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be greater than 0")
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative")
	}
	if c.Database.MaxIdleConns > c.Database.MaxConnections {
		return fmt.Errorf("max idle connections cannot exceed max connections")
	}

	// OpenAI validation - API key is optional, will use mock if not provided
	if c.OpenAI.Model == "" {
		return fmt.Errorf("OpenAI model is required")
	}
	if c.OpenAI.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if c.OpenAI.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Memory validation
	if c.Memory.MaxMemories <= 0 {
		return fmt.Errorf("max memories must be greater than 0")
	}
	if c.Memory.SimilarityThreshold < 0 || c.Memory.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity threshold must be between 0 and 1")
	}

	// Server validation
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	if !validLogLevels[c.Server.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.Server.LogLevel)
	}

	// JWT validation - allow default in development
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	// HTTP validation
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("HTTP port must be between 1 and 65535")
	}

	// Encryption validation
	if c.Encryption.Enabled && c.Encryption.MasterKey == "" {
		return fmt.Errorf("encryption master key is required when encryption is enabled")
	}

	return nil
}

// DatabaseURL constructs a PostgreSQL connection string
func (c *Config) DatabaseURL() string {
	// Build the connection parameters
	params := url.Values{}
	params.Set("sslmode", c.Database.SSLMode)

	// Construct the URL
	var userInfo *url.Userinfo
	if c.Database.Password == "" {
		userInfo = url.User(c.Database.User)
	} else {
		userInfo = url.UserPassword(c.Database.User, c.Database.Password)
	}

	u := &url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     fmt.Sprintf("%s:%d", c.Database.Host, c.Database.Port),
		Path:     c.Database.DBName,
		RawQuery: params.Encode(),
	}

	return u.String()
}