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

	"github.com/ksred/remember-me-mcp/internal/api"
	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/services"
	"github.com/ksred/remember-me-mcp/internal/utils"
	"github.com/rs/zerolog"

	// Import swagger docs
	_ "github.com/ksred/remember-me-mcp/docs"
)

// @title Remember Me MCP API
// @version 1.0
// @description API for Remember Me MCP Server - A persistent memory system for Claude

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8082
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

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
	logger.Info().Msg("Starting Remember Me MCP HTTP API server")

	// Create context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
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
	activityService := services.NewActivityService(db.DB(), logger)

	// Create and start HTTP server
	server, err := api.NewServer(cfg, db, memoryService, activityService, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create HTTP server")
	}

	// Start server in goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := server.Start(cfg.HTTP.Port); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case err := <-serverErrChan:
		logger.Error().Err(err).Msg("HTTP server error")
	}

	// Graceful shutdown
	logger.Info().Msg("Starting graceful shutdown")
	
	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Failed to gracefully shutdown HTTP server")
	}

	logger.Info().Msg("Shutdown complete")
}

// loadConfiguration loads configuration from file or environment
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

// setupLogging configures the logger based on configuration
func setupLogging(cfg *config.Config) zerolog.Logger {
	// Create log file path
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		// Default log file location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		logFile = filepath.Join(homeDir, ".config", "remember-me-mcp", "logs", "http-server.log")
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
	
	// Create and return logger
	return utils.NewLogger(logConfig)
}

// connectToDatabase establishes database connection with retry logic
func connectToDatabase(cfg *config.Config, logger zerolog.Logger) (*database.Database, error) {
	logger.Info().Msg("Connecting to database")
	
	// Create database instance
	db := database.NewDatabase(map[string]interface{}{
		"host":              cfg.Database.Host,
		"port":              cfg.Database.Port,
		"user":              cfg.Database.User,
		"password":          cfg.Database.Password,
		"dbname":            cfg.Database.DBName,
		"sslmode":          cfg.Database.SSLMode,
		"max_idle_conns":   cfg.Database.MaxIdleConns,
		"max_open_conns":   cfg.Database.MaxConnections,
		"conn_max_lifetime": cfg.Database.ConnMaxLifetime,
		"conn_max_idle_time": cfg.Database.ConnMaxIdleTime,
		"log_level":        cfg.Server.LogLevel,
	})
	
	// Connect
	if err := db.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.Health(ctx); err != nil {
		return nil, fmt.Errorf("database health check failed: %w", err)
	}
	
	logger.Info().Msg("Database connection established")
	return db, nil
}

// runMigrations runs database migrations
func runMigrations(db *database.Database, logger zerolog.Logger) error {
	logger.Info().Msg("Running database migrations")

	// Use the centralized migration function
	if err := database.RunMigrations(db.DB()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info().Msg("Database migrations completed successfully")
	return nil
}

// createEmbeddingService creates the appropriate embedding service
func createEmbeddingService(cfg *config.Config, logger zerolog.Logger) services.EmbeddingService {
	// Check if we should use mock service
	if cfg.OpenAI.APIKey == "" {
		logger.Warn().Msg("OpenAI API key not configured, using mock embedding service")
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
