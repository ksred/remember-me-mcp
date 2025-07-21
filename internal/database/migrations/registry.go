package migrations

import (
	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/utils"
)

// GetMigrations returns all registered migrations
func GetMigrations(encryptionService *utils.EncryptionService) []database.Migration {
	return []database.Migration{
		{
			Version: "20240101_001",
			Name:    "add_encryption_fields",
			Run:     AddEncryptionFields,
		},
		{
			Version: "20240101_002", 
			Name:    "encrypt_existing_memories",
			Run:     EncryptExistingMemories(encryptionService),
		},
	}
}