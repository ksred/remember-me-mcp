#!/bin/bash

# Quick Development Setup Script
# This script sets up the development environment with Docker PostgreSQL

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Main setup function
main() {
    print_status "Setting up Remember Me MCP Server for development..."
    
    # Check prerequisites
    if ! command_exists docker; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command_exists go; then
        print_error "Go is not installed. Please install Go 1.21+ first."
        exit 1
    fi
    
    # Create development environment file
    if [ ! -f ".env" ]; then
        print_status "Creating .env file from .env.dev..."
        cp .env.dev .env
    fi
    
    # Install Go dependencies
    print_status "Installing Go dependencies..."
    go mod download
    
    # Start PostgreSQL container
    print_status "Starting PostgreSQL database container..."
    make docker-db
    
    # Wait for database to be ready
    print_status "Waiting for database to be ready..."
    sleep 3
    
    # Test database connection
    print_status "Testing database connection..."
    make db-test
    
    # Build the binary
    print_status "Building the binary..."
    make build
    
    # Run tests
    print_status "Running tests..."
    make test
    
    print_success "Development setup complete!"
    
    # Print usage instructions
    cat << EOF

${GREEN}Development Environment Ready!${NC}

${YELLOW}Quick Start:${NC}
1. Start the MCP server: ${BLUE}make run${NC}
2. Test the connection: ${BLUE}make db-test${NC}
3. View database logs: ${BLUE}make docker-db-logs${NC}
4. Connect to database: ${BLUE}make docker-db-connect${NC}

${YELLOW}Development Commands:${NC}
- Start database: ${BLUE}make docker-db${NC}
- Stop database: ${BLUE}make docker-db-down${NC}
- Run server: ${BLUE}make run${NC}
- Run tests: ${BLUE}make test${NC}
- Build binary: ${BLUE}make build${NC}
- View all commands: ${BLUE}make help${NC}

${YELLOW}Database Access:${NC}
- Host: localhost
- Port: 5432
- Database: remember_me
- User: postgres
- Password: devpassword

${YELLOW}Configuration:${NC}
- Edit .env file for local settings
- Server runs in debug mode by default
- Uses mock embeddings if no OpenAI API key

${YELLOW}Next Steps:${NC}
1. Configure Claude Desktop (if needed)
2. Set OPENAI_API_KEY in .env (optional)
3. Start coding!

EOF
}

# Run main function
main "$@"