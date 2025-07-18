-- Migration script to move memories from remember_me database to postgres database
-- This script copies all memories from the remember_me.memories table to postgres.memories table
-- and assigns them to user_id = 2

-- First, let's check what we have in the source database
\c remember_me;
SELECT COUNT(*) as total_memories FROM memories;
SELECT COUNT(*) as system_user_memories FROM memories WHERE user_id = 1;

-- Now connect to the target database
\c postgres;

-- Ensure the memories table exists with all required columns
-- (The table should already exist from migrations, but let's be safe)
CREATE TABLE IF NOT EXISTS memories (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL DEFAULT 1,
    type VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    priority VARCHAR(255) DEFAULT 'medium',
    update_key VARCHAR(255),
    embedding vector(1536),
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create a temporary function to copy data between databases
-- Note: This requires the postgres_fdw extension
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

-- Create a foreign server for the remember_me database
DROP SERVER IF EXISTS remember_me_server CASCADE;
CREATE SERVER remember_me_server
    FOREIGN DATA WRAPPER postgres_fdw
    OPTIONS (host 'localhost', port '5432', dbname 'remember_me');

-- Create user mapping
CREATE USER MAPPING IF NOT EXISTS FOR postgres
    SERVER remember_me_server
    OPTIONS (user 'postgres', password 'devpassword');

-- Import the foreign schema
DROP SCHEMA IF EXISTS remember_me_import CASCADE;
CREATE SCHEMA remember_me_import;
IMPORT FOREIGN SCHEMA public
    LIMIT TO (memories)
    FROM SERVER remember_me_server
    INTO remember_me_import;

-- Now copy the memories, assigning them to user_id = 2
INSERT INTO memories (
    user_id,
    type,
    category,
    content,
    priority,
    update_key,
    embedding,
    tags,
    metadata,
    created_at,
    updated_at
)
SELECT 
    2 as user_id,  -- Assign all memories to user_id = 2
    type,
    category,
    content,
    priority,
    update_key,
    embedding,
    tags,
    metadata,
    created_at,
    updated_at
FROM remember_me_import.memories
WHERE user_id = 1  -- Only copy system user memories
ON CONFLICT DO NOTHING;  -- Skip if memory already exists

-- Show the results
SELECT COUNT(*) as memories_copied FROM memories WHERE user_id = 2;

-- Clean up
DROP SCHEMA remember_me_import CASCADE;
DROP SERVER remember_me_server CASCADE;

-- Show final state
SELECT user_id, COUNT(*) as count FROM memories GROUP BY user_id ORDER BY user_id;