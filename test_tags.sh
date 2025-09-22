#!/bin/bash

# Test script for memory tags functionality

echo "Testing memory tags functionality..."
echo "===================================="

# Test 1: Store memory with tags directly
echo -e "\n1. Testing store_memory with tags array:"
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "store_memory",
      "arguments": {
        "type": "fact",
        "category": "project",
        "content": "Testing tags functionality in remember-me-mcp",
        "tags": ["test", "mcp", "tags-feature"]
      }
    }
  }' | jq '.'

# Test 2: Store memory with tags in metadata (backward compatibility)
echo -e "\n2. Testing store_memory with tags in metadata:"
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "store_memory",
      "arguments": {
        "type": "fact",
        "category": "project",
        "content": "Testing backward compatibility for tags in metadata",
        "metadata": {
          "tags": ["metadata-test", "backward-compat"],
          "source": "test-script"
        }
      }
    }
  }' | jq '.'

# Test 3: Search for memories and check if tags are returned
echo -e "\n3. Testing search_memories to verify tags are returned:"
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "search_memories",
      "arguments": {
        "query": "tags functionality",
        "limit": 5
      }
    }
  }' | jq '.'

# Test 4: Update memory with new tags
echo -e "\n4. Testing update_memory to add/modify tags:"
echo "First, let's search to get a memory ID..."
MEMORY_ID=$(curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "search_memories",
      "arguments": {
        "query": "Testing tags functionality",
        "limit": 1
      }
    }
  }' | jq -r '.result[0].memories[0].id // empty')

if [ -n "$MEMORY_ID" ]; then
  echo "Found memory ID: $MEMORY_ID"
  echo "Updating memory with new tags..."
  curl -s -X POST http://localhost:8080/mcp \
    -H "Content-Type: application/json" \
    -d "{
      \"jsonrpc\": \"2.0\",
      \"id\": 5,
      \"method\": \"tools/call\",
      \"params\": {
        \"name\": \"update_memory\",
        \"arguments\": {
          \"id\": $MEMORY_ID,
          \"tags\": [\"updated\", \"test-complete\", \"mcp-tags\"],
          \"priority\": \"high\"
        }
      }
    }" | jq '.'
else
  echo "No memory found to update"
fi

echo -e "\n===================================="
echo "Test completed!"