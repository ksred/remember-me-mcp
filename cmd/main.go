package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/mcp"
	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/services"
	"github.com/ksred/remember-me-mcp/internal/utils"
	"github.com/rs/zerolog"
)

const version = "v0.2.0-debug-context-fix"

func main() {
	// Parse command line flags
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := loadConfiguration(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logger := setupLogging(cfg)
	logger.Info().Str("version", version).Msg("Starting Remember Me MCP server")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Connect to database
	db, err := connectToDatabase(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close database connection")
		}
	}()

	// Run migrations
	if err := runMigrations(db, logger); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Create services
	embeddingService := createEmbeddingService(cfg, logger)
	memoryService := services.NewMemoryService(db.DB(), embeddingService, logger, map[string]interface{}{
		"memory_limit": cfg.Memory.MaxMemories,
	})

	// Create and configure MCP server
	mcpServer, err := mcp.NewServer(memoryService, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create MCP server")
	}

	// Start MCP server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		logger.Info().Msg("Starting MCP server on stdio")
		if err := mcpServer.Serve(ctx); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case err := <-serverErrChan:
		logger.Error().Err(err).Msg("MCP server error")
	}

	// Graceful shutdown
	logger.Info().Msg("Starting graceful shutdown")
	
	// Cancel context to stop the server
	cancel()

	// Give the server time to clean up
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownCtx.Done():
		logger.Warn().Msg("Shutdown timeout exceeded")
	case <-time.After(2 * time.Second):
		// Allow some time for cleanup
	}

	logger.Info().Msg("Shutdown complete")
}

// loadConfiguration loads the application configuration
func loadConfiguration(configPath string) (*config.Config, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// If we can't load config, try with defaults
		cfg = config.NewDefault()
		
		// Validate the default configuration
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// setupLogging configures the application logger
func setupLogging(cfg *config.Config) zerolog.Logger {
	// Create log file path
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		// Default log file location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		logFile = filepath.Join(homeDir, ".config", "remember-me-mcp", "logs", "remember-me.log")
	}

	// Create logger configuration
	logConfig := utils.LoggerConfig{
		Level:      cfg.Server.LogLevel,
		Pretty:     cfg.Server.Debug,
		CallerInfo: cfg.Server.Debug,
		LogFile:    logFile,
	}

	// Set up global logger
	utils.SetupGlobalLogger(logConfig)

	// Create and return logger for main
	logger := utils.NewLogger(logConfig)
	return logger
}

// connectToDatabase establishes database connection
func connectToDatabase(cfg *config.Config, logger zerolog.Logger) (*database.Database, error) {
	logger.Info().Msg("Connecting to PostgreSQL database")

	// Convert config to map for database package
	dbConfig := map[string]interface{}{
		"host":              cfg.Database.Host,
		"port":              cfg.Database.Port,
		"user":              cfg.Database.User,
		"password":          cfg.Database.Password,
		"dbname":            cfg.Database.DBName,
		"sslmode":           cfg.Database.SSLMode,
		"max_open_conns":    cfg.Database.MaxConnections,
		"max_idle_conns":    cfg.Database.MaxIdleConns,
		"conn_max_lifetime": cfg.Database.ConnMaxLifetime.String(),
		"conn_max_idle_time": cfg.Database.ConnMaxIdleTime.String(),
		"log_level":         "silent", // Use silent level for GORM to prevent interference with JSON-RPC
	}

	// Create database instance
	db := database.NewDatabase(dbConfig)

	// Connect with retries
	if err := db.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Health(ctx); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	logger.Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("database", cfg.Database.DBName).
		Msg("Successfully connected to database")

	return db, nil
}

// runMigrations runs database migrations
func runMigrations(db *database.Database, logger zerolog.Logger) error {
	logger.Info().Msg("Running database migrations")

	// Run auto-migrations for models
	if err := db.Migrate(&models.Memory{}); err != nil {
		return fmt.Errorf("failed to migrate Memory model: %w", err)
	}

	logger.Info().Msg("Database migrations completed successfully")
	return nil
}

// createEmbeddingService creates the appropriate embedding service
func createEmbeddingService(cfg *config.Config, logger zerolog.Logger) services.EmbeddingService {
	// Check if we should use mock service
	if cfg.OpenAI.APIKey == "" {
		logger.Warn().Msg("No OpenAI API key provided, using mock embedding service")
		return services.NewMockEmbeddingService()
	}

	// Create OpenAI embedding service
	logger.Info().
		Str("model", cfg.OpenAI.Model).
		Msg("Creating OpenAI embedding service")

	embeddingService, err := services.NewOpenAIEmbeddingService(&cfg.OpenAI, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create OpenAI embedding service, falling back to mock")
		return services.NewMockEmbeddingService()
	}

	return embeddingService
}