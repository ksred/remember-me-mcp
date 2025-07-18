# Remember Me MCP HTTP API

The Remember Me MCP server can be run as an HTTP API server, allowing third-party applications to integrate with the memory storage system.

## Quick Start

1. **Create a configuration file** based on `config/example.http.json`
2. **Set up the database** (see main README)
3. **Run the HTTP server**:
   ```bash
   go run cmd/http-server/main.go -config your-config.json
   # or
   make run-http
   ```

## Authentication

The API supports two authentication methods:

### 1. JWT Bearer Tokens
Used for user authentication after login.

```http
Authorization: Bearer <jwt-token>
```

### 2. API Keys
Used for programmatic access and MCP integration.

```http
X-API-Key: <api-key>
```

## API Endpoints

### Authentication

#### Register a new user
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

Response:
```json
{
  "token": "eyJhbGc...",
  "expires_at": "2024-01-01T00:00:00Z",
  "user": {
    "id": 1,
    "email": "user@example.com"
  }
}
```

### API Key Management

#### Create API Key
```http
POST /api/v1/keys
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "name": "Production API Key",
  "expires_at": "2024-12-31T23:59:59Z"  // optional
}
```

Response:
```json
{
  "id": 1,
  "name": "Production API Key",
  "key": "your-api-key-shown-only-once",
  "created_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-12-31T23:59:59Z",
  "is_active": true,
  "permissions": ["memory:read", "memory:write", "memory:delete"]
}
```

#### List API Keys
```http
GET /api/v1/keys
Authorization: Bearer <jwt-token>
```

#### Delete API Key
```http
DELETE /api/v1/keys/{id}
Authorization: Bearer <jwt-token>
```

### Memory Operations

All memory endpoints require authentication via API key or JWT token.

#### Store Memory
```http
POST /api/v1/memories
X-API-Key: <api-key>
Content-Type: application/json

{
  "type": "fact",  // fact, conversation, context, preference
  "category": "personal",  // personal, project, business
  "content": "Important information to remember",
  "metadata": {
    "source": "meeting-notes",
    "tags": ["important", "project-x"]
  }
}
```

#### Search Memories
```http
GET /api/v1/memories?query=search-term&category=personal&type=fact&limit=100&useSemanticSearch=true
X-API-Key: <api-key>
```

Parameters:
- `query` (required): Search query
- `category` (optional): Filter by category
- `type` (optional): Filter by type
- `limit` (optional): Max results (default: 100, max: 1000)
- `useSemanticSearch` (optional): Use AI-powered semantic search (default: true)

#### Delete Memory
```http
DELETE /api/v1/memories/{id}
X-API-Key: <api-key>
```

#### Get Memory Statistics
```http
GET /api/v1/memories/stats
X-API-Key: <api-key>
```

## Swagger Documentation

When the server is running, you can access the interactive API documentation at:
```
http://localhost:8082/swagger/index.html
```

To regenerate the Swagger documentation:
```bash
make swagger
```

## Integration with MCP Clients

To use the HTTP API as an MCP server backend:

1. Create a user account and generate an API key
2. Configure your MCP client to use the HTTP server URL and API key
3. The client can then use the standard MCP protocol over HTTP

Example MCP client configuration:
```json
{
  "mcpServers": {
    "remember-me": {
      "url": "https://your-api-server.com",
      "apiKey": "your-api-key"
    }
  }
}
```

## Security Considerations

1. **Always use HTTPS in production** to protect API keys and user credentials
2. **Use strong JWT secrets** - never use the default secret in production
3. **Set appropriate expiration times** for API keys
4. **Implement rate limiting** in production environments
5. **Use environment variables** for sensitive configuration

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": "Error message here"
}
```

Common HTTP status codes:
- `200 OK`: Success
- `201 Created`: Resource created
- `400 Bad Request`: Invalid request
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `500 Internal Server Error`: Server error