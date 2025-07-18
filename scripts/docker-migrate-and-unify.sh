#!/bin/bash

# Docker-based migration script to unify databases
# Moves all data from remember_me to postgres database and removes remember_me

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Find the PostgreSQL container
CONTAINER=$(docker ps --format "{{.Names}}" | grep -E "(postgres|remember-me-mcp-db)" | head -1)

if [ -z "$CONTAINER" ]; then
    print_error "No PostgreSQL container found running!"
    print_status "Please start the PostgreSQL container first with: make docker-db"
    exit 1
fi

print_status "Using PostgreSQL container: $CONTAINER"

# Check if remember_me database exists
print_status "Checking if remember_me database exists..."
DB_EXISTS=$(docker exec $CONTAINER psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='remember_me';" 2>/dev/null || echo "0")

if [ "$DB_EXISTS" != "1" ]; then
    print_error "Database 'remember_me' does not exist!"
    exit 1
fi

# Check memories count in remember_me database
print_status "Checking memories in remember_me database..."
MEMORY_COUNT=$(docker exec $CONTAINER psql -U postgres -d remember_me -tAc "SELECT COUNT(*) FROM memories;" 2>/dev/null || echo "0")
print_status "Found $MEMORY_COUNT memories in remember_me database"

# Backup memories from remember_me database
print_status "Creating backup of memories from remember_me database..."
docker exec $CONTAINER pg_dump -U postgres \
    --data-only \
    --table=memories \
    --column-inserts \
    remember_me > /tmp/remember_me_memories.sql

# Modify the backup to change user_id 1 to 2
print_status "Modifying backup to assign system user memories to user_id = 2..."
sed 's/\(user_id[^,]*,\s*\)1\([,)]\)/\12\2/g' /tmp/remember_me_memories.sql > /tmp/remember_me_memories_modified.sql

# Import into postgres database
print_status "Importing memories into postgres database..."
docker exec -i $CONTAINER psql -U postgres -d postgres < /tmp/remember_me_memories_modified.sql

# Get final count
FINAL_COUNT=$(docker exec $CONTAINER psql -U postgres -d postgres -tAc "SELECT COUNT(*) FROM memories WHERE user_id = 2;" 2>/dev/null || echo "0")
print_success "Imported memories for user_id = 2: $FINAL_COUNT"

# Show memory distribution
print_status "Memory distribution in postgres database:"
docker exec $CONTAINER psql -U postgres -d postgres -c \
    "SELECT user_id, COUNT(*) as count FROM memories GROUP BY user_id ORDER BY user_id;"

# Update configuration files to use postgres database
print_status "Updating configuration files..."

# Update example configs
if [ -f "config/config.example.yaml" ]; then
    sed -i.bak 's/dbname: remember_me/dbname: postgres/g' config/config.example.yaml
    print_status "Updated config/config.example.yaml"
fi

if [ -f "config/example.http.json" ]; then
    sed -i.bak 's/"dbname": "remember_me"/"dbname": "postgres"/g' config/example.http.json
    print_status "Updated config/example.http.json"
fi

# Update .env.dev
if [ -f ".env.dev" ]; then
    sed -i.bak 's/REMEMBER_ME_DATABASE_DBNAME=remember_me/REMEMBER_ME_DATABASE_DBNAME=postgres/g' .env.dev
    sed -i.bak 's/\/remember_me?/\/postgres?/g' .env.dev
    print_status "Updated .env.dev"
fi

# Update claude desktop config script
if [ -f "scripts/claude-desktop-config.sh" ]; then
    sed -i.bak 's/"REMEMBER_ME_DATABASE_DBNAME": "remember_me"/"REMEMBER_ME_DATABASE_DBNAME": "postgres"/g' scripts/claude-desktop-config.sh
    print_status "Updated scripts/claude-desktop-config.sh"
fi

# Clean up backup files
rm -f /tmp/remember_me_memories.sql /tmp/remember_me_memories_modified.sql
rm -f config/*.bak scripts/*.bak .env.dev.bak

print_success "Migration complete! All memories moved to postgres database."

# Ask user before dropping the database
echo ""
read -p "Do you want to DROP the remember_me database? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Dropping remember_me database..."
    docker exec $CONTAINER psql -U postgres -c "DROP DATABASE remember_me;"
    print_success "Database remember_me has been dropped."
else
    print_status "Keeping remember_me database. You can drop it later with:"
    print_status "docker exec $CONTAINER psql -U postgres -c 'DROP DATABASE remember_me;'"
fi

print_success "Database unification complete!"
print_status "All services now use the 'postgres' database."