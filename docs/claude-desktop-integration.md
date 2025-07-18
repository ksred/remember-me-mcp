# Claude Desktop Integration

This document explains how to connect Claude Desktop to the Remember Me HTTP API server.

## Prerequisites

1. Remember Me HTTP server running (default: http://localhost:8082)
2. Valid API key from user registration
3. Node.js v12 or higher installed

## Setup Options

### Option 1: Simple stdio-to-HTTP Bridge (Recommended)

This option uses a simple bridge script that's compatible with older Node.js versions.

1. Update your Claude Desktop configuration file:

```json
{
  "mcpServers": {
    "remember-me": {
      "command": "node",
      "args": ["/path/to/remember-me-mcp/scripts/mcp-stdio-http-bridge.js"],
      "env": {
        "REMEMBER_ME_API_URL": "http://localhost:8082/api/v1/mcp",
        "REMEMBER_ME_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

### Option 2: MCP SDK Client

This option uses the official MCP SDK but requires Node.js v16 or higher.

1. Install dependencies:
```bash
cd /path/to/remember-me-mcp
npm install
```

2. Update Claude Desktop configuration:

```json
{
  "mcpServers": {
    "remember-me": {
      "command": "node",
      "args": ["/path/to/remember-me-mcp/scripts/mcp-http-client.js"],
      "env": {
        "REMEMBER_ME_API_URL": "http://localhost:8082/api/v1/mcp",
        "REMEMBER_ME_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

### Option 3: Advanced Proxy with Error Handling

This option includes better error handling and connection management.

```json
{
  "mcpServers": {
    "remember-me": {
      "command": "node",
      "args": ["/path/to/remember-me-mcp/scripts/mcp-http-proxy.js"],
      "env": {
        "REMEMBER_ME_API_URL": "http://localhost:8082/api/v1/mcp",
        "REMEMBER_ME_API_KEY": "your-api-key-here",
        "DEBUG": "true"
      }
    }
  }
}
```

## Troubleshooting

### Enable Debug Logging

Add `"DEBUG": "true"` to the environment variables to see detailed logs:

```json
"env": {
  "REMEMBER_ME_API_URL": "http://localhost:8082/api/v1/mcp",
  "REMEMBER_ME_API_KEY": "your-api-key-here",
  "DEBUG": "true"
}
```

### Check Server Connectivity

Test the HTTP server directly:

```bash
curl -X POST http://localhost:8082/api/v1/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-here" \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}'
```

### Node.js Version Issues

If you encounter errors related to JavaScript syntax:

1. Check your Node.js version: `node --version`
2. Use the stdio-to-HTTP bridge (Option 1) for maximum compatibility
3. The bridge script is compatible with Node.js v12 and higher

### Connection Issues

If Claude Desktop can't connect:

1. Ensure the HTTP server is running
2. Verify the API key is correct
3. Check firewall settings
4. Look for error messages in Claude Desktop logs