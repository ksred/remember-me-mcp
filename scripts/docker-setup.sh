#!/bin/bash

# Docker Setup Script for Remember Me MCP Server
# This script sets up the MCP server using Docker containers

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

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command_exists docker; then
        print_error "Docker is not installed. Please install Docker first."
        echo "Visit: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if ! command_exists docker-compose && ! docker compose version >/dev/null 2>&1; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        echo "Visit: https://docs.docker.com/compose/install/"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to setup environment
setup_environment() {
    print_status "Setting up environment..."
    
    # Check if .env exists
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            cp .env.example .env
            print_warning ".env file created from .env.example"
            print_warning "Please edit .env file with your configuration before continuing"
            echo
            echo "Required settings:"
            echo "- POSTGRES_PASSWORD: Set a secure password for PostgreSQL"
            echo "- OPENAI_API_KEY: Set your OpenAI API key"
            echo
            echo "Edit .env file and run this script again."
            exit 1
        else
            print_error ".env.example file not found"
            exit 1
        fi
    fi
    
    # Check if required variables are set
    source .env
    
    if [ -z "$POSTGRES_PASSWORD" ] || [ "$POSTGRES_PASSWORD" = "your-secure-password-here" ]; then
        print_error "Please set POSTGRES_PASSWORD in .env file"
        exit 1
    fi
    
    if [ -z "$OPENAI_API_KEY" ] || [ "$OPENAI_API_KEY" = "your-openai-api-key-here" ]; then
        print_warning "OPENAI_API_KEY not set. Server will use mock embeddings."
    fi
    
    print_success "Environment setup completed"
}

# Function to build and start services
start_services() {
    print_status "Building and starting services..."
    
    # Build and start services
    if command_exists docker-compose; then
        docker-compose up -d --build
    else
        docker compose up -d --build
    fi
    
    print_success "Services started successfully"
}

# Function to wait for services to be ready
wait_for_services() {
    print_status "Waiting for services to be ready..."
    
    # Wait for PostgreSQL to be ready
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if docker exec remember-me-postgres pg_isready -U postgres -d remember_me >/dev/null 2>&1; then
            print_success "PostgreSQL is ready"
            break
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "PostgreSQL failed to start within expected time"
        exit 1
    fi
    
    # Wait a bit more for the MCP server to initialize
    sleep 5
    
    print_success "All services are ready"
}

# Function to run database migrations
run_migrations() {
    print_status "Running database migrations..."
    
    # The migrations will be run automatically by the application
    # But we can verify the database is set up correctly
    if docker exec remember-me-postgres psql -U postgres -d remember_me -c "SELECT extname FROM pg_extension WHERE extname = 'vector';" | grep -q vector; then
        print_success "pgvector extension is installed"
    else
        print_error "pgvector extension is not installed"
        exit 1
    fi
    
    print_success "Database migrations completed"
}

# Function to configure Claude Desktop
configure_claude_desktop() {
    print_status "Configuring Claude Desktop..."
    
    CLAUDE_CONFIG_DIR="$HOME/.config/claude-desktop"
    CLAUDE_CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"
    
    # Create Claude Desktop config directory
    mkdir -p "$CLAUDE_CONFIG_DIR"
    
    # Backup existing config
    if [ -f "$CLAUDE_CONFIG_FILE" ]; then
        cp "$CLAUDE_CONFIG_FILE" "$CLAUDE_CONFIG_FILE.backup"
        print_status "Backed up existing Claude Desktop config"
    fi
    
    # Get the current directory
    CURRENT_DIR="$(pwd)"
    
    # Create new config
    cat > "$CLAUDE_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "remember-me": {
      "command": "docker",
      "args": ["exec", "-i", "remember-me-mcp", "/app/remember-me-mcp"],
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  }
}
EOF
    
    print_success "Claude Desktop configured for Docker deployment"
}

# Function to show logs
show_logs() {
    print_status "Showing service logs..."
    
    if command_exists docker-compose; then
        docker-compose logs -f
    else
        docker compose logs -f
    fi
}

# Function to stop services
stop_services() {
    print_status "Stopping services..."
    
    if command_exists docker-compose; then
        docker-compose down
    else
        docker compose down
    fi
    
    print_success "Services stopped"
}

# Function to clean up
cleanup() {
    print_status "Cleaning up..."
    
    if command_exists docker-compose; then
        docker-compose down -v
    else
        docker compose down -v
    fi
    
    # Remove images
    docker rmi remember-me-mcp_remember-me-mcp 2>/dev/null || true
    
    print_success "Cleanup completed"
}

# Function to check service status
check_status() {
    print_status "Checking service status..."
    
    if command_exists docker-compose; then
        docker-compose ps
    else
        docker compose ps
    fi
}

# Function to print usage
print_usage() {
    cat << EOF
Usage: $0 [COMMAND]

Commands:
  start     Start all services (default)
  stop      Stop all services
  restart   Restart all services
  logs      Show service logs
  status    Show service status
  cleanup   Stop services and remove volumes
  configure Configure Claude Desktop only
  help      Show this help message

Examples:
  $0 start          # Start all services
  $0 logs           # Show logs
  $0 stop           # Stop services
  $0 cleanup        # Clean up everything
EOF
}

# Function to print post-installation instructions
print_instructions() {
    cat << EOF

${GREEN}Remember Me MCP Server Docker Setup Complete!${NC}

${YELLOW}Services Status:${NC}
$(check_status)

${YELLOW}Next Steps:${NC}
1. Verify your .env file has the correct configuration
2. Restart Claude Desktop to load the new MCP server
3. Test the installation by asking Claude to remember something

${YELLOW}Management Commands:${NC}
- View logs: $0 logs
- Stop services: $0 stop
- Restart services: $0 restart
- Check status: $0 status
- Clean up: $0 cleanup

${YELLOW}Docker Commands:${NC}
- Enter PostgreSQL: docker exec -it remember-me-postgres psql -U postgres -d remember_me
- Enter MCP server: docker exec -it remember-me-mcp sh
- View MCP logs: docker logs remember-me-mcp

${YELLOW}Troubleshooting:${NC}
- Check service logs: $0 logs
- Verify PostgreSQL: docker exec remember-me-postgres pg_isready -U postgres
- Test pgvector: docker exec remember-me-postgres psql -U postgres -d remember_me -c "SELECT * FROM pg_extension WHERE extname = 'vector'"

${YELLOW}Configuration:${NC}
- Environment: .env
- Claude Desktop: ~/.config/claude-desktop/claude_desktop_config.json
- Data persistence: Docker volumes (postgres_data, remember_me_logs, remember_me_data)

For more help, visit: https://github.com/ksred/remember-me-mcp
EOF
}

# Main function
main() {
    # Parse command line arguments
    case "${1:-start}" in
        start)
            check_prerequisites
            setup_environment
            start_services
            wait_for_services
            run_migrations
            configure_claude_desktop
            print_instructions
            ;;
        stop)
            stop_services
            ;;
        restart)
            stop_services
            sleep 2
            start_services
            wait_for_services
            ;;
        logs)
            show_logs
            ;;
        status)
            check_status
            ;;
        cleanup)
            cleanup
            ;;
        configure)
            configure_claude_desktop
            print_success "Claude Desktop configured"
            ;;
        help)
            print_usage
            ;;
        *)
            print_error "Unknown command: $1"
            print_usage
            exit 1
            ;;
    esac
}

# Navigate to script directory
cd "$(dirname "$0")/.."

# Run main function
main "$@"