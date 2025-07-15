#!/bin/bash

# Claude Desktop Configuration Script
# This script helps configure Claude Desktop to use the Remember Me MCP server

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

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration paths
CLAUDE_CONFIG_DIR="$HOME/Library/Application Support/Claude"
CLAUDE_CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"
BINARY_PATH="$(pwd)/remember-me-mcp"
CONFIG_PATH="$(pwd)/.env.dev"

# Function to create Claude Desktop configuration
create_claude_config() {
    local config_type=$1
    
    # Create directory if it doesn't exist
    mkdir -p "$CLAUDE_CONFIG_DIR"
    
    # Backup existing config
    if [ -f "$CLAUDE_CONFIG_FILE" ]; then
        cp "$CLAUDE_CONFIG_FILE" "$CLAUDE_CONFIG_FILE.backup.$(date +%Y%m%d_%H%M%S)"
        print_status "Backed up existing Claude Desktop config"
    fi
    
    case $config_type in
        "development")
            # Development configuration (local binary)
            cat > "$CLAUDE_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "remember-me": {
      "command": "$BINARY_PATH",
      "env": {
        "REMEMBER_ME_DATABASE_HOST": "localhost",
        "REMEMBER_ME_DATABASE_PORT": "5432",
        "REMEMBER_ME_DATABASE_USER": "postgres",
        "REMEMBER_ME_DATABASE_PASSWORD": "devpassword",
        "REMEMBER_ME_DATABASE_DBNAME": "remember_me",
        "REMEMBER_ME_DATABASE_SSLMODE": "disable",
        "OPENAI_API_KEY": "$(grep OPENAI_API_KEY .env.dev 2>/dev/null | cut -d'=' -f2- || echo '')",
        "LOG_LEVEL": "info",
        "DEBUG": "false"
      }
    }
  }
}
EOF
            ;;
        "production")
            # Production configuration (installed binary)
            cat > "$CLAUDE_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "remember-me": {
      "command": "/usr/local/bin/remember-me-mcp",
      "env": {
        "REMEMBER_ME_DATABASE_HOST": "localhost",
        "REMEMBER_ME_DATABASE_PORT": "5432",
        "REMEMBER_ME_DATABASE_USER": "postgres",
        "REMEMBER_ME_DATABASE_PASSWORD": "devpassword",
        "REMEMBER_ME_DATABASE_DBNAME": "remember_me",
        "REMEMBER_ME_DATABASE_SSLMODE": "disable",
        "OPENAI_API_KEY": "$(grep OPENAI_API_KEY .env.dev 2>/dev/null | cut -d'=' -f2- || echo '')",
        "LOG_LEVEL": "info",
        "DEBUG": "false"
      }
    }
  }
}
EOF
            ;;
        "docker")
            # Docker configuration
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
            ;;
    esac
}

# Function to test Claude Desktop configuration
test_claude_config() {
    print_status "Testing Claude Desktop configuration..."
    
    if [ ! -f "$CLAUDE_CONFIG_FILE" ]; then
        print_error "Claude Desktop config file not found"
        return 1
    fi
    
    # Check if config is valid JSON
    if ! jq empty "$CLAUDE_CONFIG_FILE" 2>/dev/null; then
        print_error "Invalid JSON in Claude Desktop config"
        return 1
    fi
    
    # Check if remember-me server is configured
    if jq -e '.mcpServers."remember-me"' "$CLAUDE_CONFIG_FILE" >/dev/null 2>&1; then
        print_success "Remember Me MCP server is configured"
        
        # Show configuration
        echo "Current configuration:"
        jq '.mcpServers."remember-me"' "$CLAUDE_CONFIG_FILE"
        
        return 0
    else
        print_error "Remember Me MCP server not found in configuration"
        return 1
    fi
}

# Function to show Claude Desktop instructions
show_claude_instructions() {
    cat << EOF

${GREEN}Claude Desktop Integration Instructions${NC}

${YELLOW}1. Configuration:${NC}
   Your Claude Desktop config is at: $CLAUDE_CONFIG_FILE
   
${YELLOW}2. Restart Claude Desktop:${NC}
   - Quit Claude Desktop completely
   - Restart the application
   
${YELLOW}3. Test the integration:${NC}
   Try these commands in Claude Desktop:
   
   ${BLUE}Basic Commands:${NC}
   - "Remember that I prefer TypeScript over JavaScript"
   - "What do you remember about my programming preferences?"
   - "Remember that I have a meeting with John on Friday"
   - "What meetings do I have this week?"
   
   ${BLUE}Advanced Commands:${NC}
   - "Remember that I'm working on the user authentication feature for the mobile app"
   - "What projects am I working on?"
   - "Search for memories about authentication"
   - "What do you remember about mobile development?"

${YELLOW}4. Troubleshooting:${NC}
   - Check if the MCP server is running: ps aux | grep remember-me-mcp
   - Check if the database is running: make docker-db-status
   - View server logs: Check Claude Desktop's console/logs
   - Test manually: ./scripts/test-mcp.sh

${YELLOW}5. Debug Mode:${NC}
   To enable debug logging, edit your Claude Desktop config and add:
   "env": {
     "LOG_LEVEL": "debug",
     "DEBUG": "true"
   }

${YELLOW}6. What Claude Can Do:${NC}
   - Store memories about your preferences, projects, and context
   - Search through your memories using keywords or semantic search
   - Categorize memories (personal, project, business)
   - Remember different types of information (facts, conversations, context, preferences)
   - Provide context-aware responses based on your stored memories

EOF
}

# Main function
main() {
    print_status "Configuring Claude Desktop for Remember Me MCP server..."
    
    # Check if binary exists
    if [ ! -f "$BINARY_PATH" ]; then
        print_error "Binary not found at $BINARY_PATH"
        print_status "Run 'make build' first"
        exit 1
    fi
    
    # Parse command line arguments
    config_type="development"
    case "${1:-}" in
        "dev"|"development")
            config_type="development"
            ;;
        "prod"|"production")
            config_type="production"
            ;;
        "docker")
            config_type="docker"
            ;;
        "test")
            test_claude_config
            exit $?
            ;;
        "help"|"--help")
            echo "Usage: $0 [development|production|docker|test]"
            echo "  development: Configure for local development (default)"
            echo "  production:  Configure for production installation"
            echo "  docker:      Configure for Docker deployment"
            echo "  test:        Test current configuration"
            exit 0
            ;;
        *)
            if [ -n "$1" ]; then
                print_error "Unknown option: $1"
                exit 1
            fi
            ;;
    esac
    
    # Create configuration
    create_claude_config "$config_type"
    
    print_success "Claude Desktop configured for $config_type mode"
    
    # Test configuration
    test_claude_config
    
    # Show instructions
    show_claude_instructions
}

# Run main function
main "$@"