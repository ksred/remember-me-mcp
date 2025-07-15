#!/bin/bash

# Claude Memory MCP Server Setup Script
# This script sets up PostgreSQL, builds the binary, and configures Claude Desktop

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
POSTGRES_VERSION="15"
DB_NAME="remember_me"
DB_USER="postgres"
DB_PASSWORD=""
BINARY_NAME="remember-me-mcp"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/remember-me-mcp"
CLAUDE_CONFIG_DIR="$HOME/.config/claude-desktop"

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

# Function to detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin)
            OS="macos"
            ;;
        Linux)
            OS="linux"
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                DISTRO=$ID
            fi
            ;;
        *)
            print_error "Unsupported operating system"
            exit 1
            ;;
    esac
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install PostgreSQL
install_postgresql() {
    print_status "Installing PostgreSQL $POSTGRES_VERSION..."
    
    case $OS in
        macos)
            if command_exists brew; then
                brew install postgresql@$POSTGRES_VERSION
                brew services start postgresql@$POSTGRES_VERSION
            else
                print_error "Homebrew not found. Please install Homebrew first."
                exit 1
            fi
            ;;
        linux)
            case $DISTRO in
                ubuntu|debian)
                    sudo apt-get update
                    sudo apt-get install -y postgresql-$POSTGRES_VERSION postgresql-contrib postgresql-$POSTGRES_VERSION-pgvector
                    sudo systemctl start postgresql
                    sudo systemctl enable postgresql
                    ;;
                centos|rhel|fedora)
                    sudo yum install -y postgresql$POSTGRES_VERSION-server postgresql$POSTGRES_VERSION-contrib
                    sudo postgresql-$POSTGRES_VERSION-setup initdb
                    sudo systemctl start postgresql-$POSTGRES_VERSION
                    sudo systemctl enable postgresql-$POSTGRES_VERSION
                    ;;
                *)
                    print_error "Unsupported Linux distribution: $DISTRO"
                    exit 1
                    ;;
            esac
            ;;
    esac
}

# Function to setup PostgreSQL database
setup_database() {
    print_status "Setting up PostgreSQL database..."
    
    # Create database and user
    case $OS in
        macos)
            createdb $DB_NAME 2>/dev/null || true
            ;;
        linux)
            sudo -u postgres createdb $DB_NAME 2>/dev/null || true
            ;;
    esac
    
    # Install pgvector extension
    case $OS in
        macos)
            psql -d $DB_NAME -c "CREATE EXTENSION IF NOT EXISTS vector;" 2>/dev/null || {
                print_warning "pgvector extension not available. Installing..."
                brew install pgvector
                psql -d $DB_NAME -c "CREATE EXTENSION IF NOT EXISTS vector;"
            }
            ;;
        linux)
            sudo -u postgres psql -d $DB_NAME -c "CREATE EXTENSION IF NOT EXISTS vector;" 2>/dev/null || {
                print_warning "pgvector extension not available. Please install pgvector manually."
            }
            ;;
    esac
    
    print_success "Database setup completed"
}

# Function to install Go if not present
install_go() {
    if ! command_exists go; then
        print_status "Installing Go..."
        case $OS in
            macos)
                if command_exists brew; then
                    brew install go
                else
                    print_error "Homebrew not found. Please install Go manually."
                    exit 1
                fi
                ;;
            linux)
                case $DISTRO in
                    ubuntu|debian)
                        sudo apt-get update
                        sudo apt-get install -y golang-go
                        ;;
                    centos|rhel|fedora)
                        sudo yum install -y golang
                        ;;
                esac
                ;;
        esac
    fi
}

# Function to build the binary
build_binary() {
    print_status "Building Remember Me MCP server..."
    
    # Navigate to project directory
    cd "$(dirname "$0")/.."
    
    # Build the binary
    go build -o $BINARY_NAME ./cmd/main.go
    
    # Install to system path
    sudo mv $BINARY_NAME $INSTALL_DIR/
    sudo chmod +x $INSTALL_DIR/$BINARY_NAME
    
    print_success "Binary built and installed to $INSTALL_DIR/$BINARY_NAME"
}

# Function to create configuration
create_config() {
    print_status "Creating configuration files..."
    
    # Create config directory
    mkdir -p $CONFIG_DIR
    
    # Create main config file
    cat > $CONFIG_DIR/config.yaml << EOF
database:
  host: localhost
  port: 5432
  user: $DB_USER
  password: "$DB_PASSWORD"
  dbname: $DB_NAME
  sslmode: disable
  max_connections: 25
  max_idle_conns: 10
  conn_max_lifetime: 5m
  conn_max_idle_time: 1m

openai:
  api_key: ""  # Set your OpenAI API key here or use OPENAI_API_KEY environment variable
  model: text-embedding-3-small
  max_retries: 3
  timeout: 30s

memory:
  max_memories: 1000
  similarity_threshold: 0.7

server:
  log_level: info
  debug: false
EOF
    
    # Create environment file
    cat > $CONFIG_DIR/.env << EOF
# Remember Me MCP Server Environment Variables
# Copy this file and update the values as needed

# Database Configuration
REMEMBER_ME_DATABASE_HOST=localhost
REMEMBER_ME_DATABASE_PORT=5432
REMEMBER_ME_DATABASE_USER=$DB_USER
REMEMBER_ME_DATABASE_PASSWORD=$DB_PASSWORD
REMEMBER_ME_DATABASE_DBNAME=$DB_NAME
REMEMBER_ME_DATABASE_SSLMODE=disable

# OpenAI Configuration
OPENAI_API_KEY=your-openai-api-key-here

# Server Configuration
LOG_LEVEL=info
DEBUG=false
EOF
    
    print_success "Configuration files created in $CONFIG_DIR"
}

# Function to configure Claude Desktop
configure_claude_desktop() {
    print_status "Configuring Claude Desktop..."
    
    # Create Claude Desktop config directory
    mkdir -p $CLAUDE_CONFIG_DIR
    
    # Path to Claude Desktop config file
    CLAUDE_CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"
    
    # Create or update Claude Desktop configuration
    if [ -f "$CLAUDE_CONFIG_FILE" ]; then
        # Backup existing config
        cp "$CLAUDE_CONFIG_FILE" "$CLAUDE_CONFIG_FILE.backup"
        print_status "Backed up existing Claude Desktop config"
    fi
    
    # Create new config or update existing one
    cat > "$CLAUDE_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "remember-me": {
      "command": "$INSTALL_DIR/$BINARY_NAME",
      "args": ["--config", "$CONFIG_DIR/config.yaml"],
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  }
}
EOF
    
    print_success "Claude Desktop configured"
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    
    cd "$(dirname "$0")/.."
    
    if go test ./... -v; then
        print_success "All tests passed"
    else
        print_error "Some tests failed"
        exit 1
    fi
}

# Function to verify installation
verify_installation() {
    print_status "Verifying installation..."
    
    # Check if binary exists and is executable
    if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        print_success "Binary installed successfully"
    else
        print_error "Binary not found or not executable"
        exit 1
    fi
    
    # Check if config files exist
    if [ -f "$CONFIG_DIR/config.yaml" ]; then
        print_success "Configuration files created"
    else
        print_error "Configuration files not found"
        exit 1
    fi
    
    # Check if Claude Desktop config exists
    if [ -f "$CLAUDE_CONFIG_DIR/claude_desktop_config.json" ]; then
        print_success "Claude Desktop configuration updated"
    else
        print_error "Claude Desktop configuration not found"
        exit 1
    fi
    
    # Test database connection
    print_status "Testing database connection..."
    if $INSTALL_DIR/$BINARY_NAME --config $CONFIG_DIR/config.yaml 2>&1 | grep -q "Starting Remember Me MCP server"; then
        print_success "Database connection test passed"
    else
        print_warning "Database connection test failed. Please check your configuration."
    fi
}

# Function to print usage instructions
print_usage() {
    cat << EOF

${GREEN}Remember Me MCP Server Setup Complete!${NC}

${YELLOW}Next Steps:${NC}
1. Set your OpenAI API key in one of these ways:
   - Edit $CONFIG_DIR/config.yaml and add your API key
   - Set the OPENAI_API_KEY environment variable
   - Edit $CONFIG_DIR/.env and source it

2. Restart Claude Desktop to load the new MCP server

3. Test the installation by asking Claude to remember something

${YELLOW}Configuration Files:${NC}
- Main config: $CONFIG_DIR/config.yaml
- Environment: $CONFIG_DIR/.env
- Claude Desktop: $CLAUDE_CONFIG_DIR/claude_desktop_config.json

${YELLOW}Commands:${NC}
- Start server: $INSTALL_DIR/$BINARY_NAME --config $CONFIG_DIR/config.yaml
- View logs: tail -f ~/.config/remember-me-mcp/logs/remember-me.log
- Test connection: $INSTALL_DIR/$BINARY_NAME --config $CONFIG_DIR/config.yaml

${YELLOW}Troubleshooting:${NC}
- Check logs in ~/.config/remember-me-mcp/logs/
- Verify PostgreSQL is running: pg_isready
- Test database connection: psql -d $DB_NAME -c "SELECT 1"
- Verify pgvector extension: psql -d $DB_NAME -c "SELECT * FROM pg_extension WHERE extname = 'vector'"

For more help, visit: https://github.com/ksred/remember-me-mcp
EOF
}

# Main setup function
main() {
    print_status "Starting Remember Me MCP Server setup..."
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-postgresql)
                SKIP_POSTGRESQL=1
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=1
                shift
                ;;
            --db-password)
                DB_PASSWORD="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --skip-postgresql    Skip PostgreSQL installation"
                echo "  --skip-tests         Skip running tests"
                echo "  --db-password PASS   Set database password"
                echo "  --help               Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Detect operating system
    detect_os
    print_status "Detected OS: $OS"
    
    # Check for required tools
    print_status "Checking prerequisites..."
    
    # Install PostgreSQL if not present and not skipped
    if [ -z "$SKIP_POSTGRESQL" ]; then
        if ! command_exists psql; then
            install_postgresql
        else
            print_status "PostgreSQL already installed"
        fi
        
        setup_database
    else
        print_status "Skipping PostgreSQL setup"
    fi
    
    # Install Go if not present
    install_go
    
    # Run tests if not skipped
    if [ -z "$SKIP_TESTS" ]; then
        run_tests
    else
        print_status "Skipping tests"
    fi
    
    # Build and install binary
    build_binary
    
    # Create configuration files
    create_config
    
    # Configure Claude Desktop
    configure_claude_desktop
    
    # Verify installation
    verify_installation
    
    # Print usage instructions
    print_usage
    
    print_success "Setup completed successfully!"
}

# Run main function
main "$@"