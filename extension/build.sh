#!/bin/bash

# Build script for Remember Me Claude Desktop Extension

echo "Building Remember Me extension..."

# Clean up any existing build
rm -f remember-me.dxt

# Ensure dependencies are installed
cd server
npm install --production
cd ..

# Create the extension archive
zip -r remember-me.dxt manifest.json server/ README.md -x "server/node_modules/.bin/*" "*.DS_Store" "*/.*"

echo "Extension built successfully: remember-me.dxt"
echo ""
echo "To install:"
echo "1. Open Claude Desktop"
echo "2. Go to Extensions"
echo "3. Click 'Add Extension' and select remember-me.dxt"