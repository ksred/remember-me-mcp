package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database manages the database connection and operations
type Database struct {
	db     *gorm.DB
	config map[string]interface{}
	mu     sync.RWMutex
}

// NewDatabase creates a new Database instance
func NewDatabase(config map[string]interface{}) *Database {
	return &Database{
		config: config,
	}
}

// Connect establishes a connection to the PostgreSQL database with retry logic
func (d *Database) Connect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Extract connection parameters from config
	dsn := d.buildDSN()
	
	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(d.getLogLevel()),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true,
	}

	// Retry logic for connection
	maxRetries := 5
	retryDelay := time.Second * 2

	var err error
	for i := 0; i < maxRetries; i++ {
		d.db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err == nil {
			break
		}
		
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	// Configure connection pool
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	maxIdleConns := d.getConfigInt("max_idle_conns", 10)
	maxOpenConns := d.getConfigInt("max_open_conns", 100)
	connMaxLifetime := d.getConfigDuration("conn_max_lifetime", time.Hour)
	connMaxIdleTime := d.getConfigDuration("conn_max_idle_time", time.Minute*10)

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	// Enable pgvector extension
	if err := d.enablePgVector(); err != nil {
		return fmt.Errorf("failed to enable pgvector extension: %w", err)
	}

	return nil
}

// Migrate runs auto-migrations for the provided models
func (d *Database) Migrate(models ...interface{}) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database not connected")
	}

	if err := d.db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Health checks the database connection health
func (d *Database) Health(ctx context.Context) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database not connected")
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Ping with context
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check pgvector extension
	var result int
	err = d.db.WithContext(ctx).Raw("SELECT 1 FROM pg_extension WHERE extname = 'vector'").Scan(&result).Error
	if err != nil {
		return fmt.Errorf("pgvector extension check failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	d.db = nil
	return nil
}

// DB returns the underlying gorm.DB instance
func (d *Database) DB() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// SetDB sets the underlying gorm.DB instance (for testing)
func (d *Database) SetDB(db *gorm.DB) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.db = db
}

// buildDSN constructs the PostgreSQL DSN from config
func (d *Database) buildDSN() string {
	host := d.getConfigString("host", "localhost")
	port := d.getConfigInt("port", 5432)
	user := d.getConfigString("user", "postgres")
	password := d.getConfigString("password", "")
	dbname := d.getConfigString("dbname", "remember_me")
	sslmode := d.getConfigString("sslmode", "disable")
	timezone := d.getConfigString("timezone", "UTC")

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host, port, user, password, dbname, sslmode, timezone)
}

// enablePgVector enables the pgvector extension
func (d *Database) enablePgVector() error {
	// Check if extension exists
	var exists bool
	err := d.db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check for pgvector extension: %w", err)
	}

	// Create extension if it doesn't exist
	if !exists {
		if err := d.db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
			return fmt.Errorf("failed to create pgvector extension: %w", err)
		}
	}

	return nil
}

// getLogLevel returns the GORM log level from config
func (d *Database) getLogLevel() logger.LogLevel {
	level := d.getConfigString("log_level", "error")
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Error
	}
}

// Helper methods for config access

func (d *Database) getConfigString(key string, defaultValue string) string {
	if val, ok := d.config[key].(string); ok {
		return val
	}
	return defaultValue
}

func (d *Database) getConfigInt(key string, defaultValue int) int {
	if val, ok := d.config[key].(int); ok {
		return val
	}
	// Try to convert from float64 (common in JSON parsing)
	if val, ok := d.config[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func (d *Database) getConfigDuration(key string, defaultValue time.Duration) time.Duration {
	if val, ok := d.config[key].(string); ok {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	// Try direct duration
	if val, ok := d.config[key].(time.Duration); ok {
		return val
	}
	return defaultValue
}

// WithTransaction executes a function within a database transaction
func (d *Database) WithTransaction(fn func(*gorm.DB) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database not connected")
	}

	return d.db.Transaction(fn, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
}

// Exec executes raw SQL with retry logic
func (d *Database) Exec(query string, args ...interface{}) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database not connected")
	}

	maxRetries := 3
	var err error
	
	for i := 0; i < maxRetries; i++ {
		err = d.db.Exec(query, args...).Error
		if err == nil {
			return nil
		}
		
		// Don't retry on syntax errors or similar
		if !isRetryableError(err) {
			break
		}
		
		if i < maxRetries-1 {
			time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
		}
	}
	
	return err
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for connection errors, deadlocks, etc.
	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"deadlock detected",
		"too many connections",
		"connection timeout",
	}
	
	for _, retryable := range retryableErrors {
		if containsIgnoreCase(errStr, retryable) {
			return true
		}
	}
	
	return false
}

// containsIgnoreCase checks if string contains substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}