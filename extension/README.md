# Remember Me - Claude Desktop Extension

A powerful Model Context Protocol (MCP) server that provides persistent memory capabilities for Claude Desktop, allowing it to store and retrieve information across conversations.

## Installation

1. Download the `remember-me.dxt` extension file
2. In Claude Desktop, go to Extensions
3. Click "Add Extension" and select the downloaded file
4. Configure your API settings:
   - **API URL**: The URL of your Remember Me HTTP API server (default: `http://localhost:8082/api/v1/mcp`)
   - **API Key**: Your Remember Me API key for authentication

## Features

- **Semantic Search**: Find memories using AI-powered semantic similarity
- **Memory Management**: Store, update, and delete memories
- **Categorization**: Organize memories by type and category
- **Priority System**: Set priorities for important information
- **Cross-Conversation Persistence**: Access memories across all your Claude conversations

## Prerequisites

Before using this extension, you need to have the Remember Me HTTP API server running. See the [main repository](https://github.com/ksred/remember-me-mcp) for setup instructions.

## Configuration

The extension requires two configuration parameters:

1. **API URL**: The endpoint of your Remember Me HTTP API server
2. **API Key**: Your authentication key (obtain this from the Remember Me dashboard)

## Usage

Once installed and configured, Claude will automatically have access to the Remember Me tools:

- `remember_memory`: Store new information
- `search_memories`: Search through stored memories
- `update_memory`: Update existing memories
- `delete_memory`: Remove memories
- `list_recent_memories`: View recently stored memories

## Support

For issues or questions, please visit: https://github.com/ksred/remember-me-mcp/issues