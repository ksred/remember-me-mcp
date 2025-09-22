package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/utils"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to configuration file")
		dryRun     = flag.Bool("dry-run", false, "Show what would be decrypted without making changes")
		batchSize  = flag.Int("batch-size", 100, "Number of records to process at once")
		userID     = flag.Uint("user-id", 0, "Decrypt only memories for specific user (0 for all users)")
		force      = flag.Bool("force", false, "Force decryption even if encryption is enabled")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()

	// Check if we should proceed
	if cfg.Encryption.Enabled && !*force {
		logger.Fatal().Msg("Encryption is enabled. Use --force to decrypt anyway")
	}

	if cfg.Encryption.MasterKey == "" {
		logger.Fatal().Msg("No encryption master key provided")
	}

	// Create encryption service
	encryptionService, err := utils.NewEncryptionService(cfg.Encryption.MasterKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create encryption service")
	}

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	logger.Info().
		Bool("dry_run", *dryRun).
		Int("batch_size", *batchSize).
		Uint("user_id", *userID).
		Msg("Starting decryption migration")

	// Run migration
	ctx := context.Background()
	decrypted, err := runMigration(ctx, db.DB(), encryptionService, logger, *dryRun, *batchSize, *userID)
	if err != nil {
		logger.Fatal().Err(err).Msg("Migration failed")
	}

	logger.Info().
		Int("decrypted", decrypted).
		Bool("dry_run", *dryRun).
		Msg("Migration completed successfully")
}

func runMigration(ctx context.Context, db *gorm.DB, encSvc *utils.EncryptionService, logger zerolog.Logger, dryRun bool, batchSize int, specificUserID uint) (int, error) {
	var totalDecrypted int
	offset := 0

	for {
		// Build query for encrypted memories
		query := db.Model(&models.Memory{}).
			Where("is_encrypted = ?", true).
			Where("encrypted_content IS NOT NULL").
			Limit(batchSize).
			Offset(offset)

		// Filter by user if specified
		if specificUserID > 0 {
			query = query.Where("user_id = ?", specificUserID)
		}

		var memories []models.Memory
		if err := query.Find(&memories).Error; err != nil {
			return totalDecrypted, fmt.Errorf("failed to fetch memories: %w", err)
		}

		// No more records to process
		if len(memories) == 0 {
			break
		}

		logger.Info().
			Int("batch_size", len(memories)).
			Int("offset", offset).
			Msg("Processing batch")

		// Process each memory
		for _, memory := range memories {
			logger.Debug().
				Uint("id", memory.ID).
				Uint("user_id", memory.UserID).
				Str("type", memory.Type).
				Msg("Processing memory")

			if dryRun {
				logger.Info().
					Uint("id", memory.ID).
					Str("type", memory.Type).
					Str("category", memory.Category).
					Msg("Would decrypt memory")
				totalDecrypted++
				continue
			}

			// Unmarshal encrypted data
			var encryptedData utils.EncryptedData
			if err := json.Unmarshal(memory.EncryptedContent, &encryptedData); err != nil {
				logger.Error().
					Err(err).
					Uint("id", memory.ID).
					Msg("Failed to unmarshal encrypted data")
				continue
			}

			// Decrypt the content
			decryptedContent, err := encSvc.DecryptField(&encryptedData)
			if err != nil {
				logger.Error().
					Err(err).
					Uint("id", memory.ID).
					Msg("Failed to decrypt memory")
				continue
			}

			// Update the memory record
			updates := map[string]interface{}{
				"content":           decryptedContent,
				"encrypted_content": nil,
				"is_encrypted":      false,
			}

			if err := db.Model(&models.Memory{}).
				Where("id = ?", memory.ID).
				Updates(updates).Error; err != nil {
				logger.Error().
					Err(err).
					Uint("id", memory.ID).
					Msg("Failed to update memory")
				continue
			}

			logger.Info().
				Uint("id", memory.ID).
				Msg("Successfully decrypted memory")
			totalDecrypted++
		}

		// If we processed less than batch size, we're done
		if len(memories) < batchSize {
			break
		}

		offset += batchSize
	}

	return totalDecrypted, nil
}