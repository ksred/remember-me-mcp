#!/usr/bin/env node
/**
 * MCP HTTP Proxy
 * 
 * This script acts as a bridge between Claude Desktop (which expects stdio communication)
 * and the Remember Me HTTP API server that implements the MCP protocol over HTTP.
 * 
 * Usage: Set this script as the command in Claude Desktop configuration
 */

const http = require('http');
const https = require('https');
const readline = require('readline');

// Configuration from environment variables
const API_URL = process.env.REMEMBER_ME_API_URL || 'http://localhost:8082/api/v1/mcp';
const API_KEY = process.env.REMEMBER_ME_API_KEY;

// Debug logging to stderr (visible in Claude logs)
const debug = (msg) => {
  if (process.env.DEBUG === 'true') {
    console.error(`[MCP-HTTP-Proxy] ${new Date().toISOString()} - ${msg}`);
  }
};

debug('Starting MCP HTTP Proxy');
debug(`API URL: ${API_URL}`);
debug(`API Key: ${API_KEY ? 'Set' : 'Not set'}`);

if (!API_KEY) {
  console.error('Error: REMEMBER_ME_API_KEY environment variable is required');
  process.exit(1);
}

// Parse URL
const url = new URL(API_URL);
const isHttps = url.protocol === 'https:';
const httpModule = isHttps ? https : http;

// Setup readline interface for stdio communication
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

// Keep track of active state
let isActive = true;

// Process incoming JSON-RPC requests from Claude
rl.on('line', async (line) => {
  if (!isActive) return;
  
  debug(`Received from Claude: ${line}`);
  
  try {
    const request = JSON.parse(line);
    
    // Forward the request to the HTTP API
    const response = await forwardRequest(request);
    
    // Send response back to Claude
    const responseStr = JSON.stringify(response);
    debug(`Sending to Claude: ${responseStr}`);
    console.log(responseStr);
  } catch (error) {
    debug(`Error processing request: ${error.message}`);
    // Send error response
    console.log(JSON.stringify({
      jsonrpc: '2.0',
      error: {
        code: -32603,
        message: 'Internal error',
        data: error.message
      },
      id: null
    }));
  }
});

// Handle readline close
rl.on('close', () => {
  debug('Readline interface closed');
  isActive = false;
  // Don't exit on close during initialization phase
  if (initialized) {
    setTimeout(() => {
      process.exit(0);
    }, 100);
  } else {
    debug('Ignoring readline close during initialization');
  }
});

// Forward request to HTTP API
function forwardRequest(request) {
  return new Promise((resolve, reject) => {
    const postData = JSON.stringify(request);
    
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        'X-API-Key': API_KEY
      },
      timeout: 30000 // 30 second timeout
    };
    
    debug(`Forwarding to ${options.hostname}:${options.port}${options.path}`);
    
    const req = httpModule.request(options, (res) => {
      let data = '';
      
      res.on('data', (chunk) => {
        data += chunk;
      });
      
      res.on('end', () => {
        debug(`Received response: ${res.statusCode} - ${data.substring(0, 200)}...`);
        
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode}: ${data}`));
          return;
        }
        
        try {
          const response = JSON.parse(data);
          resolve(response);
        } catch (error) {
          reject(new Error(`Invalid JSON response: ${data}`));
        }
      });
    });
    
    req.on('error', (error) => {
      debug(`Request error: ${error.message}`);
      reject(error);
    });
    
    req.on('timeout', () => {
      debug('Request timeout');
      req.destroy();
      reject(new Error('Request timeout'));
    });
    
    req.write(postData);
    req.end();
  });
}

// Keep track of initialization state
let initialized = false;
let sigtermCount = 0;

// Handle process termination gracefully
process.on('SIGINT', () => {
  debug('Received SIGINT');
  isActive = false;
  rl.close();
  process.exit(0);
});

process.on('SIGTERM', () => {
  sigtermCount++;
  debug(`Received SIGTERM (count: ${sigtermCount})`);
  
  // Claude seems to send SIGTERM after initialization as a test
  // Only exit if we've received multiple SIGTERMs or if we're past initialization
  if (sigtermCount > 1 || (initialized && sigtermCount === 1)) {
    debug('Exiting due to SIGTERM');
    isActive = false;
    rl.close();
    setTimeout(() => {
      process.exit(0);
    }, 100);
  } else {
    debug('Ignoring first SIGTERM during initialization phase');
  }
});

// Prevent process from exiting immediately
process.stdin.resume();

// Handle stdin end
process.stdin.on('end', () => {
  debug('stdin ended');
  if (initialized) {
    isActive = false;
    setTimeout(() => {
      process.exit(0);
    }, 100);
  } else {
    debug('Ignoring stdin end during initialization');
  }
});

// Mark as initialized after a delay
setTimeout(() => {
  initialized = true;
  debug('Initialization phase complete');
}, 2000);

debug('MCP HTTP Proxy ready and waiting for input');