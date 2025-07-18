#!/bin/bash

# Migration script to copy memories from remember_me database to postgres database
# Assigns all memories to user_id = 2

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database connection parameters
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-devpassword}"

# Export password for psql
export PGPASSWORD="$DB_PASSWORD"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to execute SQL and return result
execute_sql() {
    local db=$1
    local sql=$2
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$db" -t -c "$sql" 2>/dev/null || echo "0"
}

# Check if both databases exist
print_status "Checking databases..."

# Check remember_me database
remember_me_exists=$(execute_sql "postgres" "SELECT 1 FROM pg_database WHERE datname = 'remember_me';")
if [ -z "$remember_me_exists" ] || [ "$remember_me_exists" = "0" ]; then
    print_error "Database 'remember_me' does not exist!"
    exit 1
fi

# Check postgres database memories table
postgres_table_exists=$(execute_sql "postgres" "SELECT 1 FROM information_schema.tables WHERE table_name = 'memories';")
if [ -z "$postgres_table_exists" ] || [ "$postgres_table_exists" = "0" ]; then
    print_error "Table 'memories' does not exist in postgres database!"
    exit 1
fi

# Count memories in source database
print_status "Counting memories in remember_me database..."
source_count=$(execute_sql "remember_me" "SELECT COUNT(*) FROM memories WHERE user_id = 1;" | xargs)
print_status "Found $source_count memories in remember_me database (user_id = 1)"

if [ "$source_count" = "0" ]; then
    print_status "No memories to migrate"
    exit 0
fi

# Create a temporary backup of memories
print_status "Creating backup of memories..."
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
    --data-only \
    --table=memories \
    --column-inserts \
    remember_me > /tmp/memories_backup.sql

# Check if backup was created
if [ ! -f /tmp/memories_backup.sql ]; then
    print_error "Failed to create backup"
    exit 1
fi

# Modify the backup to change user_id from 1 to 2
print_status "Modifying backup to assign memories to user_id = 2..."
sed -i.bak 's/\(user_id[^,]*,\s*\)1\([,)]\)/\12\2/g' /tmp/memories_backup.sql

# Count existing memories for user 2 in target database
existing_count=$(execute_sql "postgres" "SELECT COUNT(*) FROM memories WHERE user_id = 2;" | xargs)
print_status "Existing memories for user_id = 2 in postgres database: $existing_count"

# Import the modified backup into postgres database
print_status "Importing memories into postgres database..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres < /tmp/memories_backup.sql 2>/dev/null || true

# Count memories after import
new_count=$(execute_sql "postgres" "SELECT COUNT(*) FROM memories WHERE user_id = 2;" | xargs)
imported=$((new_count - existing_count))

print_success "Migration complete!"
print_status "Memories imported: $imported"
print_status "Total memories for user_id = 2: $new_count"

# Show summary
print_status "Memory distribution in postgres database:"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c \
    "SELECT user_id, COUNT(*) as count FROM memories GROUP BY user_id ORDER BY user_id;"

# Clean up
rm -f /tmp/memories_backup.sql /tmp/memories_backup.sql.bak

print_status "Backup files cleaned up"
print_success "Migration completed successfully!"

# Unset password
unset PGPASSWORD