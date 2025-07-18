#!/usr/bin/env node
/**
 * MCP HTTP Client
 * A proper MCP server that communicates with the Remember Me HTTP API
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import axios from 'axios';

// Configuration
const API_URL = process.env.REMEMBER_ME_API_URL || 'http://localhost:8082/api/v1/mcp';
const API_KEY = process.env.REMEMBER_ME_API_KEY;

if (!API_KEY) {
  console.error('Error: REMEMBER_ME_API_KEY environment variable is required');
  process.exit(1);
}

// Create axios instance with default config
const api = axios.create({
  baseURL: API_URL,
  headers: {
    'X-API-Key': API_KEY,
    'Content-Type': 'application/json'
  },
  timeout: 30000
});

// Create MCP server
const server = new Server(
  {
    name: 'remember-me',
    version: '1.0.0',
  },
  {
    capabilities: {
      resources: {},
      tools: {},
    },
  }
);

// Error handler
server.onerror = (error) => {
  console.error('[MCP HTTP Client Error]', error);
};

// Initialize handler - fetch capabilities from HTTP server
server.setRequestHandler('initialize', async (request) => {
  try {
    // Forward initialize request to HTTP server
    const response = await api.post('', {
      jsonrpc: '2.0',
      method: 'initialize',
      params: request.params,
      id: 1
    });

    // Return the capabilities from the HTTP server
    return response.data.result;
  } catch (error) {
    console.error('Initialize error:', error.message);
    // Return default capabilities if HTTP server is unreachable
    return {
      protocolVersion: '0.1.0',
      serverInfo: {
        name: 'remember-me',
        version: '1.0.0'
      },
      capabilities: {
        resources: true,
        tools: true
      }
    };
  }
});

// List tools handler
server.setRequestHandler('tools/list', async () => {
  try {
    const response = await api.post('', {
      jsonrpc: '2.0',
      method: 'tools/list',
      id: 1
    });
    
    return response.data.result;
  } catch (error) {
    console.error('List tools error:', error.message);
    throw error;
  }
});

// Call tool handler
server.setRequestHandler('tools/call', async (request) => {
  try {
    const response = await api.post('', {
      jsonrpc: '2.0',
      method: 'tools/call',
      params: request.params,
      id: 1
    });
    
    return response.data.result;
  } catch (error) {
    console.error('Call tool error:', error.message);
    throw error;
  }
});

// List resources handler
server.setRequestHandler('resources/list', async () => {
  try {
    const response = await api.post('', {
      jsonrpc: '2.0',
      method: 'resources/list',
      id: 1
    });
    
    return response.data.result;
  } catch (error) {
    console.error('List resources error:', error.message);
    throw error;
  }
});

// Read resource handler
server.setRequestHandler('resources/read', async (request) => {
  try {
    const response = await api.post('', {
      jsonrpc: '2.0',
      method: 'resources/read',
      params: request.params,
      id: 1
    });
    
    return response.data.result;
  } catch (error) {
    console.error('Read resource error:', error.message);
    throw error;
  }
});

// Start the server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  
  // Keep the process alive
  process.stdin.resume();
}

main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});