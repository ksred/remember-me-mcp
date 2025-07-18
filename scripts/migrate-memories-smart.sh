#!/bin/bash

# Smart migration script that handles ID conflicts

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Smart Memory Migration Script${NC}"

# Find container
CONTAINER=$(docker ps --format "{{.Names}}" | grep -E "(postgres|remember-me-mcp-db)" | head -1)

# First, let's see what we're dealing with
echo -e "${BLUE}Analyzing current state...${NC}"

# Check max ID in postgres database
MAX_ID_POSTGRES=$(docker exec $CONTAINER psql -U postgres -d postgres -tAc "SELECT COALESCE(MAX(id), 0) FROM memories;" 2>/dev/null || echo "0")
echo "Max memory ID in postgres database: $MAX_ID_POSTGRES"

# Count memories in remember_me database
COUNT_REMEMBER_ME=$(docker exec $CONTAINER psql -U postgres -d remember_me -tAc "SELECT COUNT(*) FROM memories;" 2>/dev/null || echo "0")
echo "Total memories in remember_me database: $COUNT_REMEMBER_ME"

if [ "$COUNT_REMEMBER_ME" = "0" ]; then
    echo -e "${GREEN}No memories to migrate${NC}"
    exit 0
fi

# Export memories from remember_me, but without the ID column (let postgres assign new IDs)
echo -e "${BLUE}Exporting memories from remember_me database...${NC}"

docker exec $CONTAINER psql -U postgres -d remember_me -t -A -F'|' << 'EOF' > /tmp/memories_export.csv
SELECT 
    2 as user_id,  -- Assign to user_id 2
    type,
    category,
    content,
    priority,
    update_key,
    encode(embedding::text::bytea, 'hex') as embedding_hex,
    array_to_string(tags, '|||') as tags_str,
    metadata::text,
    created_at,
    updated_at
FROM memories
WHERE user_id = 1 OR user_id IS NULL;
EOF

# Import into postgres database
echo -e "${BLUE}Importing memories into postgres database...${NC}"

# Create a temporary import script
cat > /tmp/import_memories.sql << 'EOSQL'
\copy memories (user_id, type, category, content, priority, update_key, embedding, tags, metadata, created_at, updated_at) FROM stdin WITH (FORMAT csv, DELIMITER '|', NULL 'NULL')
EOSQL

# Append the CSV data, converting back the embedding and tags
awk -F'|' '{
    # Fields: user_id, type, category, content, priority, update_key, embedding_hex, tags_str, metadata, created_at, updated_at
    printf "%s|%s|%s|%s|%s|%s|", $1, $2, $3, $4, $5, $6;
    
    # Convert hex back to vector format or NULL
    if ($7 != "" && $7 != "NULL") {
        printf "\\\\x%s|", $7;
    } else {
        printf "NULL|";
    }
    
    # Convert tags back to array format
    if ($8 != "" && $8 != "NULL") {
        gsub(/\|\|\|/, ",", $8);
        printf "{%s}|", $8;
    } else {
        printf "NULL|";
    }
    
    # Print metadata, created_at, updated_at
    printf "%s|%s|%s\n", $9, $10, $11;
}' /tmp/memories_export.csv >> /tmp/import_memories.sql

echo "\\." >> /tmp/import_memories.sql

# Run the import
docker exec -i $CONTAINER psql -U postgres -d postgres < /tmp/import_memories.sql

# Count new memories
NEW_COUNT=$(docker exec $CONTAINER psql -U postgres -d postgres -tAc "SELECT COUNT(*) FROM memories WHERE user_id = 2;" 2>/dev/null || echo "0")
echo -e "${GREEN}Total memories for user_id = 2: $NEW_COUNT${NC}"

# Show distribution
echo -e "${BLUE}Memory distribution in postgres database:${NC}"
docker exec $CONTAINER psql -U postgres -d postgres -c "SELECT user_id, COUNT(*) as count FROM memories GROUP BY user_id ORDER BY user_id;"

# Clean up
rm -f /tmp/memories_export.csv /tmp/import_memories.sql

echo -e "${GREEN}Migration complete!${NC}"

# Ask about dropping remember_me database
echo ""
read -p "Drop the remember_me database? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    docker exec $CONTAINER psql -U postgres -c "DROP DATABASE remember_me;"
    echo -e "${GREEN}Database remember_me dropped.${NC}"
fi