#!/bin/bash

# Quick fix script to set up postgres database properly

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Setting up postgres database schema...${NC}"

# Find container
CONTAINER=$(docker ps --format "{{.Names}}" | grep -E "(postgres|remember-me-mcp-db)" | head -1)

if [ -z "$CONTAINER" ]; then
    echo "Error: PostgreSQL container not running!"
    exit 1
fi

# First, ensure pgvector extension is installed
docker exec $CONTAINER psql -U postgres -d postgres -c "CREATE EXTENSION IF NOT EXISTS vector;"

# Run the HTTP server briefly to trigger migrations
echo -e "${BLUE}Running HTTP server to trigger migrations...${NC}"
cd /Users/ksred/Documents/GitHub/remember-me-mcp

# Start the server in background, wait a bit, then kill it
timeout 10s go run cmd/http-server/main.go -config config/example.http.json 2>&1 | grep -E "(migration|Migration|schema)" || true

echo -e "${GREEN}Done! Now checking the schema...${NC}"

# Check if tables exist
docker exec $CONTAINER psql -U postgres -d postgres -c "\dt"

# Check if user_id column exists in memories
docker exec $CONTAINER psql -U postgres -d postgres -c "\d memories"

echo -e "${GREEN}Setup complete!${NC}"