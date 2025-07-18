#!/bin/bash

# Add user_id column to existing memories table

set -e

# Find container
CONTAINER=$(docker ps --format "{{.Names}}" | grep -E "(postgres|remember-me-mcp-db)" | head -1)

echo "Adding user_id column to memories table in postgres database..."

docker exec $CONTAINER psql -U postgres -d postgres << 'EOF'
-- Add user_id column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'memories' AND column_name = 'user_id') THEN
        ALTER TABLE memories ADD COLUMN user_id INTEGER NOT NULL DEFAULT 1;
        
        -- Create index
        CREATE INDEX idx_memories_user_id ON memories(user_id);
        
        -- Add foreign key constraint to users table (if users table exists)
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
            ALTER TABLE memories ADD CONSTRAINT fk_memories_user_id 
            FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE;
        END IF;
        
        RAISE NOTICE 'Added user_id column to memories table';
    ELSE
        RAISE NOTICE 'user_id column already exists';
    END IF;
END $$;

-- Show the table structure
\d memories
EOF

echo "Done!"