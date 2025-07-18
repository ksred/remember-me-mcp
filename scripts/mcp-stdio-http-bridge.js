#!/usr/bin/env node
/**
 * MCP stdio-to-HTTP Bridge
 * Compatible with older Node.js versions (v12+)
 * Forwards stdio MCP messages to HTTP MCP server
 */

const http = require('http');
const https = require('https');
const readline = require('readline');

// Configuration
const API_URL = process.env.REMEMBER_ME_API_URL || 'http://localhost:8082/api/v1/mcp';
const API_KEY = process.env.REMEMBER_ME_API_KEY;

// Simple debug function that writes to stderr
function debug(msg) {
  if (process.env.DEBUG === 'true') {
    process.stderr.write('[MCP Bridge] ' + new Date().toISOString() + ' - ' + msg + '\n');
  }
}

if (!API_KEY) {
  process.stderr.write('Error: REMEMBER_ME_API_KEY environment variable is required\n');
  process.exit(1);
}

debug('Starting MCP stdio-to-HTTP bridge');
debug('API URL: ' + API_URL);

// Parse URL manually for Node v12 compatibility
const urlParts = API_URL.match(/^(https?):\/\/([^:\/]+)(?::(\d+))?(\/.*)?$/);
if (!urlParts) {
  process.stderr.write('Error: Invalid API URL format\n');
  process.exit(1);
}

const isHttps = urlParts[1] === 'https';
const hostname = urlParts[2];
const port = urlParts[3] || (isHttps ? '443' : '80');
const pathname = urlParts[4] || '/';
const httpModule = isHttps ? https : http;

// Setup readline for reading JSON-RPC messages from stdin
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

// Buffer for incomplete JSON messages
let buffer = '';

// Process each line from stdin
rl.on('line', function(line) {
  buffer += line;
  
  // Try to parse as JSON
  let message;
  try {
    message = JSON.parse(buffer);
    buffer = ''; // Clear buffer on successful parse
  } catch (e) {
    // Not complete JSON yet, wait for more
    return;
  }
  
  debug('Received: ' + JSON.stringify(message));
  
  // Forward to HTTP server
  forwardToHttp(message);
});

// Forward message to HTTP server
function forwardToHttp(message) {
  const postData = JSON.stringify(message);
  
  const options = {
    hostname: hostname,
    port: parseInt(port),
    path: pathname,
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Content-Length': Buffer.byteLength(postData),
      'X-API-Key': API_KEY
    }
  };
  
  const req = httpModule.request(options, function(res) {
    let responseData = '';
    
    res.on('data', function(chunk) {
      responseData += chunk;
    });
    
    res.on('end', function() {
      if (res.statusCode !== 200) {
        debug('HTTP error: ' + res.statusCode + ' - ' + responseData);
        // Send error response
        const errorResponse = {
          jsonrpc: '2.0',
          error: {
            code: -32603,
            message: 'HTTP error: ' + res.statusCode,
            data: responseData
          },
          id: message.id || null
        };
        process.stdout.write(JSON.stringify(errorResponse) + '\n');
        return;
      }
      
      try {
        const response = JSON.parse(responseData);
        debug('Response: ' + JSON.stringify(response));
        // Forward response to stdout
        process.stdout.write(JSON.stringify(response) + '\n');
        
        // Mark as initialized after first successful response
        if (message.method === 'initialize') {
          markInitialized();
        }
      } catch (e) {
        debug('Failed to parse response: ' + e.message);
        const errorResponse = {
          jsonrpc: '2.0',
          error: {
            code: -32603,
            message: 'Invalid response from server',
            data: responseData
          },
          id: message.id || null
        };
        process.stdout.write(JSON.stringify(errorResponse) + '\n');
      }
    });
  });
  
  req.on('error', function(error) {
    debug('Request error: ' + error.message);
    const errorResponse = {
      jsonrpc: '2.0',
      error: {
        code: -32603,
        message: 'Request failed: ' + error.message
      },
      id: message.id || null
    };
    process.stdout.write(JSON.stringify(errorResponse) + '\n');
  });
  
  req.write(postData);
  req.end();
}

// Handle stdin close
process.stdin.on('end', function() {
  debug('stdin closed (initialized: ' + initialized + ')');
  if (initialized) {
    process.exit(0);
  } else {
    debug('Ignoring stdin close during initialization');
  }
});

// Track initialization state
let initialized = false;
let sigtermCount = 0;

// Mark as initialized after first successful response
function markInitialized() {
  if (!initialized) {
    initialized = true;
    debug('Marked as initialized');
  }
}

// Handle process termination
process.on('SIGINT', function() {
  debug('Received SIGINT');
  process.exit(0);
});

process.on('SIGTERM', function() {
  sigtermCount++;
  debug('Received SIGTERM (count: ' + sigtermCount + ', initialized: ' + initialized + ')');
  
  // Claude sends SIGTERM after initialization as a test
  // Only exit if we've received multiple SIGTERMs or if we're past initialization
  if (sigtermCount > 1 || (initialized && sigtermCount === 1)) {
    debug('Exiting due to SIGTERM');
    process.exit(0);
  } else {
    debug('Ignoring first SIGTERM during initialization phase');
  }
});

// Keep process alive
process.stdin.resume();

debug('Bridge ready');