#!/usr/bin/env node
// Test script to verify the proxy works correctly

const http = require('http');

const API_URL = 'http://localhost:8082/api/v1/mcp';
const API_KEY = process.env.REMEMBER_ME_API_KEY || 'test-key';

// Test initialize request
const testRequest = {
  jsonrpc: "2.0",
  method: "initialize",
  params: {
    protocolVersion: "2025-06-18",
    capabilities: {},
    clientInfo: {
      name: "test-client",
      version: "1.0.0"
    }
  },
  id: 1
};

const postData = JSON.stringify(testRequest);

const url = new URL(API_URL);
const options = {
  hostname: url.hostname,
  port: url.port || 80,
  path: url.pathname,
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(postData),
    'X-API-Key': API_KEY
  }
};

const req = http.request(options, (res) => {
  let data = '';
  
  res.on('data', (chunk) => {
    data += chunk;
  });
  
  res.on('end', () => {
    console.log('Status:', res.statusCode);
    console.log('Response:', data);
  });
});

req.on('error', (error) => {
  console.error('Error:', error);
});

req.write(postData);
req.end();