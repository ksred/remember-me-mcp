#!/usr/bin/env node
/**
 * Test script for MCP stdio-to-HTTP bridge
 * Tests basic JSON-RPC communication
 */

const spawn = require('child_process').spawn;
const path = require('path');

// Configuration
const bridgeScript = path.join(__dirname, 'mcp-stdio-http-bridge.js');
const apiUrl = process.env.REMEMBER_ME_API_URL || 'http://localhost:8082/api/v1/mcp';
const apiKey = process.env.REMEMBER_ME_API_KEY || '80b6d70a35a56addc0b2208aa5186f6692f45ae90549e446cb42cb6c9a263f39';

console.log('Testing MCP stdio-to-HTTP bridge...');
console.log('Bridge script:', bridgeScript);
console.log('API URL:', apiUrl);
console.log('API Key:', apiKey ? 'Set' : 'Not set');

// Spawn the bridge process
const bridge = spawn('node', [bridgeScript], {
  env: Object.assign({}, process.env, {
    REMEMBER_ME_API_URL: apiUrl,
    REMEMBER_ME_API_KEY: apiKey,
    DEBUG: 'true'
  })
});

// Handle bridge output
bridge.stdout.on('data', function(data) {
  console.log('Bridge response:', data.toString().trim());
});

bridge.stderr.on('data', function(data) {
  console.error('Bridge debug:', data.toString().trim());
});

bridge.on('error', function(error) {
  console.error('Failed to start bridge:', error.message);
  process.exit(1);
});

bridge.on('exit', function(code) {
  console.log('Bridge exited with code:', code);
});

// Send test requests
setTimeout(function() {
  console.log('\nSending initialize request...');
  const initRequest = {
    jsonrpc: '2.0',
    method: 'initialize',
    params: {
      protocolVersion: '0.1.0',
      capabilities: {}
    },
    id: 1
  };
  bridge.stdin.write(JSON.stringify(initRequest) + '\n');
}, 500);

setTimeout(function() {
  console.log('\nSending tools/list request...');
  const toolsRequest = {
    jsonrpc: '2.0',
    method: 'tools/list',
    params: {},
    id: 2
  };
  bridge.stdin.write(JSON.stringify(toolsRequest) + '\n');
}, 1500);

// Exit after tests
setTimeout(function() {
  console.log('\nTest complete, ending bridge...');
  bridge.stdin.end();
}, 3000);