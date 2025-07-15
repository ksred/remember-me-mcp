# Testing Guide - Remember Me MCP Server

This guide walks you through testing the Remember Me MCP server from manual testing to Claude Desktop integration.

## 🧪 Complete Testing Strategy

### **1. Manual Testing (Direct MCP Protocol)**

This tests the MCP server directly using JSON-RPC:

```bash
# Make sure your server is built
make build

# Test the MCP server with automated tests
make test-mcp
```

The `test-mcp.go` script will:
- Initialize the MCP connection
- List available tools
- Store memories
- Search memories
- Show you the JSON-RPC protocol in action
- Explain how MCP servers work

### **2. Database Testing**

```bash
# Check database is running
make docker-db-status

# Test database connection
make db-test

# Connect to database directly
make docker-db-connect
```

### **3. Claude Desktop Integration**

This is where the magic happens! Here's the step-by-step process:

#### **Step 1: Configure Claude Desktop**
```bash
# Auto-configure Claude Desktop for development
make configure-claude
```

This creates `~/.config/claude-desktop/claude_desktop_config.json` with:
```json
{
  "mcpServers": {
    "remember-me": {
      "command": "/path/to/your/remember-me-mcp",
      "env": {
        "REMEMBER_ME_DATABASE_HOST": "localhost",
        "REMEMBER_ME_DATABASE_PORT": "5432",
        "REMEMBER_ME_DATABASE_USER": "postgres",
        "REMEMBER_ME_DATABASE_PASSWORD": "devpassword",
        "REMEMBER_ME_DATABASE_DBNAME": "remember_me",
        "OPENAI_API_KEY": "your-key-here",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

#### **Step 2: Start Your Services**
```bash
# Start PostgreSQL
make docker-db

# Start MCP server (in another terminal)
make run
```

#### **Step 3: Restart Claude Desktop**
- Quit Claude Desktop completely
- Restart the application
- Claude Desktop will automatically connect to your MCP server

### **4. Claude Desktop Testing Commands**

Once Claude Desktop is connected, you can test with natural language:

#### **Basic Memory Storage:**
```
"Remember that I prefer TypeScript over JavaScript for new projects"
"Remember that I have a meeting with John on Friday at 2 PM"
"Remember that I'm working on the user authentication feature"
```

#### **Memory Retrieval:**
```
"What do you remember about my programming preferences?"
"What meetings do I have this week?"
"What projects am I working on?"
"Search for memories about authentication"
```

#### **Advanced Testing:**
```
"Remember that I prefer email communication over Slack for business"
"What do you remember about my communication preferences?"
"Search for memories about John"
"What do you remember about meetings?"
```

### **5. How Claude Desktop Integration Works**

1. **Protocol**: Claude Desktop communicates with your MCP server via stdio (standard input/output)
2. **JSON-RPC**: All communication uses JSON-RPC 2.0 protocol
3. **Tools**: Claude Desktop sees your three tools:
   - `store_memory` - When you ask Claude to remember something
   - `search_memories` - When you ask Claude what it remembers
   - `delete_memory` - When you ask Claude to forget something

4. **Natural Language**: Claude translates your natural language into tool calls:
   ```
   User: "Remember that I prefer TypeScript"
   Claude: [Calls store_memory with appropriate parameters]
   
   User: "What do you remember about my preferences?"
   Claude: [Calls search_memories to find relevant memories]
   ```

### **6. Debugging & Troubleshooting**

#### **Check if MCP server is running:**
```bash
ps aux | grep remember-me-mcp
```

#### **Check Claude Desktop logs:**
- macOS: `~/Library/Logs/Claude/`
- Look for MCP-related errors

#### **Test MCP server manually:**
```bash
# Direct test
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0.0"}}}' | ./remember-me-mcp

# Use the test script
make test-mcp
```

#### **Debug mode:**
```bash
# Set debug logging
export LOG_LEVEL=debug
export DEBUG=true
make run
```

### **7. What You Should See**

**In your MCP server logs:**
```
[INFO] Starting Remember Me MCP server
[INFO] Successfully connected to database
[INFO] MCP server listening on stdio
[DEBUG] Received tool call: store_memory
[DEBUG] Storing memory: content="I prefer TypeScript..."
```

**In Claude Desktop:**
```
User: Remember that I prefer TypeScript over JavaScript
Claude: I'll remember that you prefer TypeScript over JavaScript for new projects. This preference has been stored.

User: What do you remember about my programming preferences?
Claude: Based on what I remember:
- You prefer TypeScript over JavaScript for new projects
```

### **8. Common Issues & Solutions**

1. **Claude Desktop doesn't connect**: 
   - Check config file syntax with `jq . ~/.config/claude-desktop/claude_desktop_config.json`
   - Restart Claude Desktop completely

2. **Database connection fails**:
   - Check `make docker-db-status`
   - Verify environment variables in config

3. **No OpenAI API key**:
   - Server will use mock embeddings (still works!)
   - Semantic search will fall back to keyword search

4. **Binary not found**:
   - Run `make build` first
   - Check path in Claude Desktop config

5. **Tools not appearing in Claude Desktop**:
   - Check MCP server logs for errors
   - Verify JSON-RPC communication with `make test-mcp`
   - Ensure Claude Desktop config is valid JSON

6. **Database permissions**:
   - Check PostgreSQL is running: `make docker-db-status`
   - Test connection: `make db-test`

## 🔄 Complete Testing Workflow

Here's a complete end-to-end testing workflow:

```bash
# 1. Setup development environment
make dev-setup

# 2. Build and test the server
make build
make test

# 3. Test database connection
make db-test

# 4. Test MCP server manually
make test-mcp

# 5. Configure Claude Desktop
make configure-claude

# 6. Start services
make docker-db        # Terminal 1
make run             # Terminal 2

# 7. Restart Claude Desktop and test with natural language
```

## 🧰 Testing Tools Summary

| Command | Purpose |
|---------|---------|
| `make test-mcp` | Test MCP server with JSON-RPC |
| `make configure-claude` | Configure Claude Desktop |
| `make db-test` | Test database connection |
| `make docker-db-status` | Check PostgreSQL status |
| `make docker-db-connect` | Connect to database |
| `make run` | Start MCP server |
| `make test` | Run Go unit tests |

## 📝 Manual Testing Examples

### Direct JSON-RPC Testing

```bash
# Initialize connection
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0.0"}}}' | ./remember-me-mcp

# Store memory
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"store_memory","arguments":{"content":"I prefer TypeScript","type":"preference","category":"personal"}}}' | ./remember-me-mcp

# Search memories
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_memories","arguments":{"query":"TypeScript","limit":5}}}' | ./remember-me-mcp
```

### Database Testing

```bash
# Connect to database
make docker-db-connect

# In psql:
SELECT * FROM memories;
SELECT COUNT(*) FROM memories;
SELECT * FROM pg_extension WHERE extname = 'vector';
```

This testing guide provides a complete pathway from development to production testing of your Remember Me MCP server.