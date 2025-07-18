package config

import (
	"os"
	"testing"
)

func TestLoadConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("DATABASE_URL", "postgres://testuser:testpass@testhost:5433/testdb?sslmode=require")
	os.Setenv("OPENAI_API_KEY", "test-api-key")
	os.Setenv("REMEMBER_ME_SERVER_LOG_LEVEL", "debug")
	os.Setenv("REMEMBER_ME_SERVER_DEBUG", "true")
	os.Setenv("REMEMBER_ME_JWT_SECRET", "test-secret")
	os.Setenv("REMEMBER_ME_HTTP_PORT", "8080")
	
	defer func() {
		// Clean up
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("REMEMBER_ME_SERVER_LOG_LEVEL")
		os.Unsetenv("REMEMBER_ME_SERVER_DEBUG")
		os.Unsetenv("REMEMBER_ME_JWT_SECRET")
		os.Unsetenv("REMEMBER_ME_HTTP_PORT")
	}()
	
	// Load config with no config file
	cfg := LoadConfigOrDefault("")
	
	// Test database configuration from DATABASE_URL
	if cfg.Database.User != "testuser" {
		t.Errorf("Expected database user 'testuser', got '%s'", cfg.Database.User)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("Expected database password 'testpass', got '%s'", cfg.Database.Password)
	}
	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected database host 'testhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("Expected database port 5433, got %d", cfg.Database.Port)
	}
	if cfg.Database.DBName != "testdb" {
		t.Errorf("Expected database name 'testdb', got '%s'", cfg.Database.DBName)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("Expected sslmode 'require', got '%s'", cfg.Database.SSLMode)
	}
	
	// Test other environment variables
	if cfg.OpenAI.APIKey != "test-api-key" {
		t.Errorf("Expected OpenAI API key 'test-api-key', got '%s'", cfg.OpenAI.APIKey)
	}
	if cfg.Server.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.Server.LogLevel)
	}
	if !cfg.Server.Debug {
		t.Error("Expected debug mode to be true")
	}
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("Expected JWT secret 'test-secret', got '%s'", cfg.JWT.Secret)
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("Expected HTTP port 8080, got %d", cfg.HTTP.Port)
	}
}

func TestDatabaseURLParsing(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected Database
	}{
		{
			name: "basic postgres URL",
			url:  "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
			expected: Database{
				User:     "user",
				Password: "pass",
				Host:     "localhost",
				Port:     5432,
				DBName:   "mydb",
				SSLMode:  "disable",
			},
		},
		{
			name: "postgresql prefix",
			url:  "postgresql://user:pass@remotehost:5433/db?sslmode=require",
			expected: Database{
				User:     "user",
				Password: "pass",
				Host:     "remotehost",
				Port:     5433,
				DBName:   "db",
				SSLMode:  "require",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("DATABASE_URL", tc.url)
			defer os.Unsetenv("DATABASE_URL")
			
			cfg, err := LoadConfig("")
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}
			
			if cfg.Database.User != tc.expected.User {
				t.Errorf("User: expected %s, got %s", tc.expected.User, cfg.Database.User)
			}
			if cfg.Database.Password != tc.expected.Password {
				t.Errorf("Password: expected %s, got %s", tc.expected.Password, cfg.Database.Password)
			}
			if cfg.Database.Host != tc.expected.Host {
				t.Errorf("Host: expected %s, got %s", tc.expected.Host, cfg.Database.Host)
			}
			if cfg.Database.Port != tc.expected.Port {
				t.Errorf("Port: expected %d, got %d", tc.expected.Port, cfg.Database.Port)
			}
			if cfg.Database.DBName != tc.expected.DBName {
				t.Errorf("DBName: expected %s, got %s", tc.expected.DBName, cfg.Database.DBName)
			}
			if cfg.Database.SSLMode != tc.expected.SSLMode {
				t.Errorf("SSLMode: expected %s, got %s", tc.expected.SSLMode, cfg.Database.SSLMode)
			}
		})
	}
}