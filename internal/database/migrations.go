package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ksred/remember-me-mcp/internal/models"
	"gorm.io/gorm"
)

// SystemUserID is the reserved user ID for local MCP operations
const SystemUserID = 1

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	// Run auto-migrations for all models
	if err := db.AutoMigrate(
		&models.User{},
		&models.APIKey{},
		&models.Memory{},
		&models.ActivityLog{},
		&models.PerformanceMetric{},
	); err != nil {
		return fmt.Errorf("failed to run auto-migrations: %w", err)
	}

	// Create system user if it doesn't exist
	if err := createSystemUser(db); err != nil {
		return fmt.Errorf("failed to create system user: %w", err)
	}

	// Add composite index for user_id and update_key for efficient lookups
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_memories_user_update_key 
		ON memories(user_id, update_key) 
		WHERE update_key IS NOT NULL
	`).Error; err != nil {
		return fmt.Errorf("failed to create composite index: %w", err)
	}

	return nil
}

// createSystemUser creates the system user for local MCP operations
func createSystemUser(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int64
	if err := db.WithContext(ctx).Model(&models.User{}).Where("id = ?", SystemUserID).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		// System user already exists
		return nil
	}

	// Create system user
	systemUser := &models.User{
		Email:     "system@remember-me.local",
		Password:  "no-login", // This user cannot log in
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert with specific ID
	if err := db.WithContext(ctx).Exec(
		"INSERT INTO users (id, email, password, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		SystemUserID, systemUser.Email, systemUser.Password, systemUser.CreatedAt, systemUser.UpdatedAt,
	).Error; err != nil {
		return fmt.Errorf("failed to create system user: %w", err)
	}

	return nil
}