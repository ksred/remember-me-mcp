# Remember Me MCP Server

A powerful Model Context Protocol (MCP) server that provides persistent memory capabilities for Claude Desktop, enabling Claude to remember and recall information across conversations.

## Features

- **Persistent Memory**: Store and retrieve memories across Claude Desktop sessions
- **Semantic Search**: OpenAI embeddings with vector similarity search for intelligent memory retrieval
- **Automatic Pattern Detection**: Intelligently detects memory-worthy content from conversations
- **Smart Updates**: Updates existing memories instead of creating duplicates using update keys
- **Async Processing**: Memories stored instantly, embeddings generated in background
- **Memory Categories**: Organize by type (fact, conversation, context, preference) and category (personal, project, business)
- **Priority System**: Memories prioritized as low, medium, high, or critical
- **PostgreSQL + pgvector**: Robust database with vector operations for semantic search
- **HTTP API**: RESTful API with authentication for third-party integrations
- **Comprehensive Testing**: Full test coverage for all components

## Quick Start

### Option 1: Claude Desktop Extension (Easiest)

1. Download the latest `remember-me.dxt` from [Releases](https://github.com/ksred/remember-me-mcp/releases)
2. Open Claude Desktop → Extensions → Add Extension
3. Select the downloaded `remember-me.dxt` file
4. Configure your API URL and API Key
5. Start using Remember Me!

For detailed extension instructions, see the [extension README](./extension/README.md).

### Option 2: One-Command Setup (Recommended for Self-Hosting)

```bash
curl -sSL https://raw.githubusercontent.com/ksred/remember-me-mcp/main/scripts/setup.sh | bash
```

### Option 3: Development Setup (Recommended for Development)

```bash
git clone https://github.com/ksred/remember-me-mcp.git
cd remember-me-mcp
make dev-setup
```

This sets up PostgreSQL in Docker while running the MCP server locally.

### Option 4: Docker Setup (Full Containerized)

```bash
git clone https://github.com/ksred/remember-me-mcp.git
cd remember-me-mcp
make docker-setup
```

### Option 5: Manual Installation

1. **Prerequisites**: PostgreSQL 15+, Go 1.21+, pgvector extension
2. **Clone and build**:
   ```bash
   git clone https://github.com/ksred/remember-me-mcp.git
   cd remember-me-mcp
   make build
   sudo make install
   ```
3. **Configure Claude Desktop**: Add to `~/Library/Application Support/Claude/claude_desktop_config.json`
4. **Restart Claude Desktop**: Required for configuration changes to take effect

## Configuration

The server can be configured through environment variables or a YAML configuration file:

### Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/remember_me
REMEMBER_ME_DATABASE_HOST=localhost
REMEMBER_ME_DATABASE_PORT=5432
REMEMBER_ME_DATABASE_USER=postgres
REMEMBER_ME_DATABASE_PASSWORD=your-password
REMEMBER_ME_DATABASE_DBNAME=remember_me

# OpenAI
OPENAI_API_KEY=your-api-key-here

# Server
LOG_LEVEL=info
DEBUG=false
```

### Configuration File

Create `~/.config/remember-me-mcp/config.yaml`:

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: your-password
  dbname: remember_me
  sslmode: disable

openai:
  api_key: your-api-key-here
  model: text-embedding-3-small

memory:
  max_memories: 1000
  similarity_threshold: 0.7

server:
  log_level: info
  debug: false
```

## Claude Desktop Integration

Configure Claude Desktop by editing the configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/claude-desktop/claude_desktop_config.json`

Add the following configuration:

```json
{
  "mcpServers": {
    "remember-me": {
      "command": "/usr/local/bin/remember-me-mcp",
      "env": {
        "REMEMBER_ME_DATABASE_HOST": "localhost",
        "REMEMBER_ME_DATABASE_PORT": "5432",
        "REMEMBER_ME_DATABASE_USER": "postgres",
        "REMEMBER_ME_DATABASE_PASSWORD": "your-database-password",
        "REMEMBER_ME_DATABASE_DBNAME": "remember_me",
        "REMEMBER_ME_DATABASE_SSLMODE": "disable",
        "OPENAI_API_KEY": "your-openai-api-key-here",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

**Important**: After updating the configuration file, restart Claude Desktop for the changes to take effect.

## MCP Tools

The server provides three MCP tools:

### 1. store_memory

Store a new memory or update an existing one.

**Parameters:**
- `content` (required): The memory content
- `type` (required): Memory type (`fact`, `conversation`, `context`, `preference`)
- `category` (required): Memory category (`personal`, `project`, `business`)
- `tags` (optional): Array of tags
- `metadata` (optional): Additional metadata object

**Example:**
```json
{
  "content": "User prefers email communication over Slack",
  "type": "preference",
  "category": "business",
  "tags": ["communication", "preferences"],
  "metadata": {
    "source": "user_feedback",
    "priority": "high"
  }
}
```

### 2. search_memories

Search for memories using keyword or semantic search.

**Parameters:**
- `query` (optional): Search query
- `category` (optional): Filter by category
- `type` (optional): Filter by type
- `limit` (optional): Maximum results (default: 10)
- `use_semantic_search` (optional): Use vector search (default: false)

**Example:**
```json
{
  "query": "email preferences",
  "category": "business",
  "use_semantic_search": true,
  "limit": 5
}
```

### 3. delete_memory

Delete a memory by ID.

**Parameters:**
- `id` (required): Memory ID to delete

**Example:**
```json
{
  "id": 123
}
```

## Memory Types

- **fact**: Factual information about the user or context
- **conversation**: Important conversation history
- **context**: Contextual information for better understanding
- **preference**: User preferences and settings

## Memory Categories

- **personal**: Personal information and preferences
- **project**: Project-related memories
- **business**: Business and professional context

## API Examples

### Claude Desktop Usage

Once installed, you can use the memory system naturally in Claude Desktop:

```
User: "Remember that I prefer TypeScript over JavaScript for new projects"
Claude: I'll remember that you prefer TypeScript over JavaScript for new projects.
[Memory stored instantly, embedding generated in background]

User: "What are my programming preferences?"
Claude: Based on what I remember about your programming preferences:
- You prefer TypeScript over JavaScript for new projects
[Uses keyword search immediately, semantic search available after embedding is ready]
```

**How it works:**
- **Instant Storage**: Memories are stored immediately without waiting for OpenAI API
- **Background Processing**: Embeddings are generated asynchronously for semantic search
- **Dual Search**: Keyword search works instantly, semantic search enabled once embeddings are ready
- **No Timeouts**: Memory storage is never blocked by API timeouts

### Direct API Usage

```bash
# Store a memory
echo '{"content": "Meeting with John on Friday", "type": "context", "category": "business"}' | \
  ./remember-me-mcp

# Search memories
echo '{"query": "John", "use_semantic_search": true}' | \
  ./remember-me-mcp
```

## HTTP API Server

The Remember Me MCP server can also run as a standalone HTTP API server, allowing third-party applications to integrate with the memory system.

### Running the HTTP Server

```bash
# Using make
make run-http

# Or directly
go run cmd/http-server/main.go -config config.json

# With Docker
docker run -p 8082:8082 remember-me-mcp:latest http-server
```

### Features

- **User Registration & Authentication**: JWT-based authentication
- **API Key Management**: Generate and manage API keys for programmatic access
- **RESTful Endpoints**: Full CRUD operations for memories
- **Swagger Documentation**: Interactive API docs at `/swagger`

For detailed HTTP API documentation, see [docs/HTTP_API.md](docs/HTTP_API.md).

## Development

### Prerequisites

- Go 1.21+
- Docker (for PostgreSQL)
- OpenAI API key (optional, will use mock embeddings if not provided)

### Quick Development Setup

```bash
# Clone repository
git clone https://github.com/ksred/remember-me-mcp.git
cd remember-me-mcp

# One-command development setup
make dev-setup
```

### Manual Development Setup

```bash
# Install dependencies
make deps

# Start PostgreSQL in Docker
make docker-db

# Test database connection
make db-test

# Run tests
make test

# Start development server
make run
```

### Database-Only Docker Commands

```bash
# Start PostgreSQL container
make docker-db

# Stop PostgreSQL container
make docker-db-down

# View PostgreSQL logs
make docker-db-logs

# Connect to PostgreSQL
make docker-db-connect

# Clean up database container
make docker-db-clean

# Test database connection
make db-test
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with HTML coverage report
make test-coverage-html

# Run linter
make lint
```

### Docker Development

```bash
# Start with Docker
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down

# Clean up
make docker-clean
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Claude        │    │   MCP Server    │    │   PostgreSQL    │
│   Desktop       │◄──►│   (Go)          │◄──►│   + pgvector    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   OpenAI API    │
                       │   (Embeddings)  │
                       └─────────────────┘
```

### Components

- **MCP Server**: Go-based server implementing MCP protocol
- **Memory Service**: Core business logic for memory operations
- **Database Layer**: PostgreSQL with pgvector for vector storage
- **OpenAI Integration**: Text embeddings for semantic search
- **Configuration**: Flexible configuration management

## Deployment

### Production Setup

1. **Server Requirements**: 
   - 2+ CPU cores
   - 4GB+ RAM
   - 20GB+ storage
   - PostgreSQL 15+ with pgvector

2. **Environment Setup**:
   ```bash
   # Create production environment
   cp .env.example .env
   # Edit .env with production values
   
   # Deploy with Docker
   make docker-setup
   ```

3. **Database Optimization**:
   ```sql
   -- Optimize for production
   ALTER SYSTEM SET shared_buffers = '256MB';
   ALTER SYSTEM SET effective_cache_size = '1GB';
   ALTER SYSTEM SET maintenance_work_mem = '64MB';
   ```

### Monitoring

- **Logs**: View logs with `make docker-logs`
- **Health Check**: Built-in health checks in Docker
- **Metrics**: Application metrics available via logs

## Security

- **Database**: Use strong passwords and SSL connections
- **API Keys**: Store OpenAI API keys securely
- **Access Control**: Restrict database access to application only
- **Input Validation**: All inputs are validated and sanitized

## Troubleshooting

### Common Issues

1. **Database Connection Failed**:
   ```bash
   # Check PostgreSQL is running
   pg_isready -h localhost -p 5432
   
   # Check database exists
   psql -l | grep remember_me
   ```

2. **pgvector Extension Missing**:
   ```bash
   # Install pgvector
   # macOS: brew install pgvector
   # Ubuntu: apt-get install postgresql-15-pgvector
   
   # Enable extension
   psql -d remember_me -c "CREATE EXTENSION IF NOT EXISTS vector;"
   ```

3. **OpenAI API Issues**:
   ```bash
   # Check API key
   echo $OPENAI_API_KEY
   
   # Test API access
   curl -H "Authorization: Bearer $OPENAI_API_KEY" \
        https://api.openai.com/v1/models
   ```

4. **Claude Desktop Not Connecting**:
   ```bash
   # Check configuration
   cat ~/.config/claude-desktop/claude_desktop_config.json
   
   # Restart Claude Desktop
   # Check logs in Claude Desktop
   ```

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
export DEBUG=true
./remember-me-mcp
```

### Performance Tuning

1. **Database Indexes**: Automatically created by GORM
2. **Connection Pooling**: Configured via environment variables
3. **Memory Limits**: Set `MEMORY_MAX_MEMORIES` to limit storage
4. **Embedding Cache**: Consider caching for frequently accessed memories

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices
- Write comprehensive tests
- Update documentation
- Use conventional commits
- Ensure all tests pass

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [GitHub Wiki](https://github.com/ksred/remember-me-mcp/wiki)
- **Issues**: [GitHub Issues](https://github.com/ksred/remember-me-mcp/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ksred/remember-me-mcp/discussions)

## Roadmap

- [ ] Web dashboard for memory management
- [ ] Multi-user support
- [ ] Additional embedding providers
- [ ] Memory expiration policies
- [ ] Export/import functionality
- [ ] Advanced search filters
- [ ] Memory clustering and summarization

## Acknowledgments

- [Model Context Protocol](https://github.com/modelcontextprotocol/specification) for the MCP specification
- [pgvector](https://github.com/pgvector/pgvector) for PostgreSQL vector support
- [OpenAI](https://openai.com/) for embedding models
- [GORM](https://gorm.io/) for database ORM

---

**Made with ❤️ for the Claude Desktop community**