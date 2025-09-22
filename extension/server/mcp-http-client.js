#!/usr/bin/env node
/**
 * MCP HTTP Client
 * A proper MCP server that communicates with the Remember Me HTTP API
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { 
  ListResourcesRequestSchema, 
  ReadResourceRequestSchema,
  ListToolsRequestSchema,
  CallToolRequestSchema
} from '@modelcontextprotocol/sdk/types.js';
import axios from 'axios';

// Configuration
const API_URL = process.env.REMEMBER_ME_API_URL || 'http://localhost:8082/api/v1/mcp';
const API_KEY = process.env.REMEMBER_ME_API_KEY;

// Log startup configuration
process.stderr.write(`[MCP-HTTP-CLIENT] Starting up...\n`);
process.stderr.write(`[MCP-HTTP-CLIENT] API URL: ${API_URL}\n`);
process.stderr.write(`[MCP-HTTP-CLIENT] API Key: ${API_KEY ? '***' + API_KEY.slice(-4) : 'NOT SET'}\n`);

if (!API_KEY) {
  process.stderr.write('[MCP-HTTP-CLIENT] Error: REMEMBER_ME_API_KEY environment variable is required\n');
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

// Note: Initialize is handled automatically by the Server constructor

// List tools handler
server.setRequestHandler(ListToolsRequestSchema, async () => {
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

// Track last tool call time for rate limiting workaround
let lastToolCallTime = 0;

// Call tool handler
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  try {
    // Add a small delay between tool calls to work around Claude Desktop bug
    const now = Date.now();
    const timeSinceLastCall = now - lastToolCallTime;
    if (timeSinceLastCall < 100) {
      const delay = 100 - timeSinceLastCall;
      process.stderr.write(`[MCP-HTTP-CLIENT] Delaying ${delay}ms to avoid Claude Desktop bug\n`);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
    lastToolCallTime = Date.now();
    
    // Log the full request from Claude Desktop
    process.stderr.write(`\n[MCP-HTTP-CLIENT] === TOOL CALL RECEIVED ===\n`);
    process.stderr.write(`[MCP-HTTP-CLIENT] Full request: ${JSON.stringify(request, null, 2)}\n`);
    process.stderr.write(`[MCP-HTTP-CLIENT] Tool name: ${request.params.name}\n`);
    process.stderr.write(`[MCP-HTTP-CLIENT] Has arguments: ${request.params.arguments ? 'YES' : 'NO'}\n`);
    
    // Workaround for Claude Desktop bug with multiple tool calls
    // Sometimes arguments come as null or undefined
    let args = request.params.arguments;
    
    if (args === null || args === undefined) {
      process.stderr.write(`[MCP-HTTP-CLIENT] WARNING: Arguments are null/undefined!\n`);
      args = {};
    } else if (typeof args === 'string') {
      // Sometimes arguments might come as a string that needs parsing
      try {
        args = JSON.parse(args);
        process.stderr.write(`[MCP-HTTP-CLIENT] Parsed string arguments: ${JSON.stringify(args, null, 2)}\n`);
      } catch (e) {
        process.stderr.write(`[MCP-HTTP-CLIENT] Failed to parse string arguments: ${e.message}\n`);
        args = {};
      }
    }
    
    if (args && Object.keys(args).length > 0) {
      process.stderr.write(`[MCP-HTTP-CLIENT] Arguments: ${JSON.stringify(args, null, 2)}\n`);
    } else {
      process.stderr.write(`[MCP-HTTP-CLIENT] WARNING: No valid arguments provided!\n`);
    }
    
    // Check if arguments are missing - this is a common issue
    if (!args || Object.keys(args).length === 0) {
      process.stderr.write(`[MCP-HTTP-CLIENT] Returning error for missing arguments\n`);
      
      // Include the full request in the error response for debugging
      const errorResponse = {
        success: false,
        error: 'Tool called without valid arguments',
        debug: {
          receivedRequest: request,
          receivedParams: request.params,
          processedArgs: args,
          hasArguments: false,
          errorDetails: 'This may be due to a Claude Desktop bug with multiple sequential tool calls',
          expectedFormat: {
            params: {
              name: "store_memory",
              arguments: {
                type: "fact/conversation/context/preference",
                category: "personal/project/business",
                content: "the text to remember",
                tags: ["optional", "tags"],
                metadata: {}
              }
            }
          }
        }
      };
      
      return {
        content: [{
          type: 'text',
          text: JSON.stringify(errorResponse)
        }]
      };
    }
    
    // The MCP SDK passes the tool name and arguments in request.params
    const toolCallParams = {
      name: request.params.name,
      arguments: args  // Use the normalized args
    };
    
    // Log what we're sending to the HTTP server
    process.stderr.write(`[MCP-HTTP-CLIENT] Sending to HTTP server: ${JSON.stringify(toolCallParams, null, 2)}\n`);
    
    const response = await api.post('', {
      jsonrpc: '2.0',
      method: 'tools/call',
      params: toolCallParams,
      id: 1
    });
    
    // Log the response
    process.stderr.write(`[MCP-HTTP-CLIENT] Response from server: ${JSON.stringify(response.data, null, 2)}\n`);
    process.stderr.write(`[MCP-HTTP-CLIENT] === TOOL CALL COMPLETE ===\n\n`);
    
    return response.data.result;
  } catch (error) {
    process.stderr.write(`[MCP-HTTP-CLIENT] Error: ${error.message}\n`);
    if (error.response) {
      process.stderr.write(`[MCP-HTTP-CLIENT] Error response: ${JSON.stringify(error.response.data, null, 2)}\n`);
    }
    
    // Return error in MCP format
    return {
      content: [{
        type: 'text',
        text: JSON.stringify({
          success: false,
          error: error.message
        })
      }]
    };
  }
});

// List resources handler
server.setRequestHandler(ListResourcesRequestSchema, async () => {
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
server.setRequestHandler(ReadResourceRequestSchema, async (request) => {
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