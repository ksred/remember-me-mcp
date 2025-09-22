# Memory Encryption

Remember Me MCP now supports field-level encryption for memory content to protect sensitive data at rest.

## Overview

The encryption system uses:
- **AES-256-GCM** for authenticated encryption
- **Master key** from environment variable
- **Per-message encryption keys** for enhanced security
- **Base64 encoding** for storage

## Configuration

### 1. Generate a Master Key

```bash
go run cmd/keygen/main.go
```

This will output a base64-encoded master key. Save this securely!

### 2. Enable Encryption

Add to your `.env` file:

```env
ENCRYPTION_ENABLED=true
ENCRYPTION_MASTER_KEY=your-base64-encoded-master-key
```

Or set environment variables:
```bash
export ENCRYPTION_ENABLED=true
export ENCRYPTION_MASTER_KEY=your-base64-encoded-master-key
```

## Migration

### Automatic Migration

When encryption is enabled, the application automatically:
1. Adds encryption fields to the database on startup
2. Encrypts all existing unencrypted messages
3. Tracks migration status to prevent re-running

The migration runs automatically when you start the application with encryption enabled. Progress is logged during startup.

### Manual Migration Control

You can control migration behavior with the `--skip-migrations` flag:

```bash
# Skip all migrations on startup
./remember-me-mcp --skip-migrations

# HTTP server
./remember-me-http --skip-migrations
```

### Standalone Migration Tools

For manual control or troubleshooting, standalone tools are available:

#### Encrypt Messages
```bash
# Dry run to see what would be encrypted
go run cmd/migrate-encrypt/main.go --dry-run

# Encrypt all messages
go run cmd/migrate-encrypt/main.go

# Encrypt only for specific user
go run cmd/migrate-encrypt/main.go --user-id=2
```

#### Decrypt Messages
```bash
# Dry run
go run cmd/migrate-decrypt/main.go --dry-run

# Decrypt all messages (requires --force if encryption is enabled)
go run cmd/migrate-decrypt/main.go --force

# Decrypt for specific user
go run cmd/migrate-decrypt/main.go --user-id=2 --force
```

### Migration Tracking

Migrations are tracked in the `schema_migrations` table:
- Each migration has a unique version
- Migrations only run once
- You can check migration status in the database

```sql
SELECT * FROM schema_migrations ORDER BY applied_at;
```

## How It Works

1. **Storage**: When encryption is enabled, memory content is:
   - Encrypted with a unique data key
   - Data key is encrypted with the master key
   - Original content is replaced with "[encrypted]" marker
   - Encrypted data stored in `encrypted_content` JSONB field

2. **Retrieval**: When reading memories:
   - Encrypted data is decrypted automatically
   - Original content is returned to the user
   - If decryption fails, "[encrypted]" marker is shown

3. **Search**: 
   - Semantic search still works (embeddings are generated from original content)
   - Keyword search works on decrypted content in memory

## Security Considerations

1. **Master Key Security**:
   - Store the master key securely (use secrets management in production)
   - Never commit the master key to version control
   - Backup the master key - losing it means losing access to encrypted data

2. **Key Rotation**: Currently not implemented. To rotate keys:
   - Decrypt all data with old key
   - Generate new master key
   - Re-encrypt all data with new key

3. **Performance**: 
   - Minimal impact on write operations
   - Slight overhead on read operations for decryption
   - Batch operations recommended for large datasets

## Database Schema Changes

The encryption feature adds these fields to the `memories` table:
- `encrypted_content` (JSONB) - Stores encrypted data
- `is_encrypted` (boolean) - Indicates if content is encrypted

## Troubleshooting

1. **"content is encrypted but encryption service is not available"**
   - Ensure `ENCRYPTION_MASTER_KEY` is set
   - Check that the master key is valid base64

2. **Migration fails**
   - Check database connectivity
   - Ensure sufficient permissions to update records
   - Review logs for specific errors

3. **Cannot decrypt after key change**
   - You must use the same master key that encrypted the data
   - If key is lost, data cannot be recovered