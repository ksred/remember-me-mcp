package migrations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/utils"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// EncryptExistingMemories encrypts all existing unencrypted memories
func EncryptExistingMemories(encryptionService *utils.EncryptionService) func(ctx context.Context, db *gorm.DB, logger zerolog.Logger) error {
	return func(ctx context.Context, db *gorm.DB, logger zerolog.Logger) error {
		// Skip if encryption service is not available
		if encryptionService == nil {
			logger.Warn().Msg("Encryption service not available, skipping memory encryption migration")
			return nil
		}

		logger.Info().Msg("Starting encryption of existing memories with active encryption service")

		var totalEncrypted int
		batchSize := 100
		offset := 0

		for {
			var memories []models.Memory
			
			// Fetch unencrypted memories in batches
			if err := db.Model(&models.Memory{}).
				Where("is_encrypted = ? OR is_encrypted IS NULL", false).
				Where("content != ?", "[encrypted]").
				Limit(batchSize).
				Offset(offset).
				Find(&memories).Error; err != nil {
				return fmt.Errorf("failed to fetch memories: %w", err)
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
				// Skip if content is empty
				if memory.Content == "" {
					continue
				}

				// Encrypt the content
				encryptedData, err := encryptionService.EncryptField(memory.Content)
				if err != nil {
					logger.Error().
						Err(err).
						Uint("id", memory.ID).
						Msg("Failed to encrypt memory, skipping")
					continue
				}

				// Marshal encrypted data
				encryptedJSON, err := json.Marshal(encryptedData)
				if err != nil {
					logger.Error().
						Err(err).
						Uint("id", memory.ID).
						Msg("Failed to marshal encrypted data, skipping")
					continue
				}

				// Update the memory record directly without model validation
				if err := db.Exec(
					"UPDATE memories SET encrypted_content = ?, is_encrypted = ?, content = ? WHERE id = ?",
					encryptedJSON, true, "[encrypted]", memory.ID,
				).Error; err != nil {
					logger.Error().
						Err(err).
						Uint("id", memory.ID).
						Msg("Failed to update memory, skipping")
					continue
				}

				logger.Debug().
					Uint("id", memory.ID).
					Str("type", memory.Type).
					Msg("Successfully encrypted memory")
				
				totalEncrypted++
			}

			// If we processed less than batch size, we're done
			if len(memories) < batchSize {
				break
			}

			offset += batchSize
		}

		logger.Info().
			Int("total_encrypted", totalEncrypted).
			Msg("Completed encryption of existing memories")

		return nil
	}
}