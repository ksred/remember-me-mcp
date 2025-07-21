package migrations

import (
	"context"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AddEncryptionFields adds the encryption-related fields to the memories table
func AddEncryptionFields(ctx context.Context, db *gorm.DB, logger zerolog.Logger) error {
	logger.Info().Msg("Adding encryption fields to memories table")

	// Add encrypted_content column if it doesn't exist
	if !db.Migrator().HasColumn("memories", "encrypted_content") {
		if err := db.Exec(`
			ALTER TABLE memories 
			ADD COLUMN encrypted_content JSONB
		`).Error; err != nil {
			return err
		}
		logger.Info().Msg("Added encrypted_content column")
	}

	// Add is_encrypted column if it doesn't exist
	if !db.Migrator().HasColumn("memories", "is_encrypted") {
		if err := db.Exec(`
			ALTER TABLE memories 
			ADD COLUMN is_encrypted BOOLEAN DEFAULT FALSE
		`).Error; err != nil {
			return err
		}
		logger.Info().Msg("Added is_encrypted column")
	}

	// Create index on is_encrypted for faster queries
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_memories_is_encrypted 
		ON memories(is_encrypted)
	`).Error; err != nil {
		return err
	}

	return nil
}