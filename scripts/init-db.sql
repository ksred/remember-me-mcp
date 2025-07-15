-- Initialize Remember Me MCP Server Database
-- This script is run when the PostgreSQL container starts for the first time

-- Create the pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Verify the extension is installed
SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';

-- Create indexes for better performance (will be created by GORM migrations, but included for reference)
-- These will be created automatically by the application, but documented here for clarity

-- Example queries to verify setup
-- SELECT version();
-- SELECT * FROM pg_extension WHERE extname = 'vector';

-- Set up any additional database configuration if needed
-- ALTER DATABASE remember_me SET timezone TO 'UTC';

-- Grant permissions (if using a different user)
-- GRANT ALL PRIVILEGES ON DATABASE remember_me TO postgres;

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'Remember Me MCP Server database initialized successfully';
END $$;