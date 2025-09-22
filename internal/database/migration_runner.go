package database

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// MigrationFunc is a function that performs a migration
type MigrationFunc func(ctx context.Context, db *gorm.DB, logger zerolog.Logger) error

// Migration represents a migration to be run
type Migration struct {
	Version string
	Name    string
	Run     MigrationFunc
}

// MigrationRunner handles running database migrations
type MigrationRunner struct {
	db         *gorm.DB
	logger     zerolog.Logger
	migrations []Migration
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *gorm.DB, logger zerolog.Logger) *MigrationRunner {
	return &MigrationRunner{
		db:         db,
		logger:     logger,
		migrations: []Migration{},
	}
}

// Register adds a migration to the runner
func (r *MigrationRunner) Register(migration Migration) {
	r.migrations = append(r.migrations, migration)
}

// Run executes all pending migrations
func (r *MigrationRunner) Run(ctx context.Context) error {
	// Ensure migrations table exists
	if err := r.db.AutoMigrate(&models.Migration{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Sort migrations by version
	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].Version < r.migrations[j].Version
	})

	// Get applied migrations
	var applied []string
	if err := r.db.Model(&models.Migration{}).Pluck("version", &applied).Error; err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, v := range applied {
		appliedMap[v] = true
	}

	// Run pending migrations
	for _, migration := range r.migrations {
		if appliedMap[migration.Version] {
			r.logger.Debug().
				Str("version", migration.Version).
				Str("name", migration.Name).
				Msg("Migration already applied, skipping")
			continue
		}

		r.logger.Info().
			Str("version", migration.Version).
			Str("name", migration.Name).
			Msg("Running migration")

		// Start transaction
		tx := r.db.Begin()
		if tx.Error != nil {
			return fmt.Errorf("failed to start transaction: %w", tx.Error)
		}

		// Run migration
		if err := migration.Run(ctx, tx, r.logger); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", migration.Version, err)
		}

		// Record migration
		record := &models.Migration{
			Version:   migration.Version,
			Name:      migration.Name,
			AppliedAt: time.Now(),
		}

		if err := tx.Create(record).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		r.logger.Info().
			Str("version", migration.Version).
			Str("name", migration.Name).
			Msg("Migration completed successfully")
	}

	return nil
}

// GetPendingMigrations returns a list of migrations that haven't been applied yet
func (r *MigrationRunner) GetPendingMigrations() ([]Migration, error) {
	// Get applied migrations
	var applied []string
	if err := r.db.Model(&models.Migration{}).Pluck("version", &applied).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, v := range applied {
		appliedMap[v] = true
	}

	var pending []Migration
	for _, migration := range r.migrations {
		if !appliedMap[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}