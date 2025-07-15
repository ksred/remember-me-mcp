package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreMemoryRequest_Structure(t *testing.T) {
	req := StoreMemoryRequest{
		Type:     "fact",
		Category: "personal",
		Content:  "Test content",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}
	
	assert.Equal(t, "fact", req.Type)
	assert.Equal(t, "personal", req.Category)
	assert.Equal(t, "Test content", req.Content)
	assert.Equal(t, "value", req.Metadata["key"])
}

func TestSearchMemoriesRequest_Structure(t *testing.T) {
	req := SearchMemoriesRequest{
		Query:             "test query",
		Category:          "personal",
		Type:              "fact",
		Limit:             10,
		UseSemanticSearch: true,
	}
	
	assert.Equal(t, "test query", req.Query)
	assert.Equal(t, "personal", req.Category)
	assert.Equal(t, "fact", req.Type)
	assert.Equal(t, 10, req.Limit)
	assert.True(t, req.UseSemanticSearch)
}

func TestDeleteMemoryRequest_Structure(t *testing.T) {
	req := DeleteMemoryRequest{
		ID: 42,
	}
	
	assert.Equal(t, uint(42), req.ID)
}

func TestMemoryResponse_NewSuccessResponse(t *testing.T) {
	data := map[string]interface{}{
		"id":      1,
		"content": "test",
	}
	
	response := NewSuccessResponse("Memory stored successfully", data)
	
	assert.True(t, response.Success)
	assert.Equal(t, "Memory stored successfully", response.Message)
	assert.Equal(t, data, response.Data)
	assert.Empty(t, response.Error)
}

func TestMemoryResponse_NewErrorResponse(t *testing.T) {
	response := NewErrorResponse("Database error")
	
	assert.False(t, response.Success)
	assert.Equal(t, "Database error", response.Error)
	assert.Nil(t, response.Data)
}

func TestMemoryResponse_ToJSON(t *testing.T) {
	response := NewSuccessResponse("Success", map[string]interface{}{
		"id": 1,
	})
	
	jsonBytes, err := response.ToJSON()
	require.NoError(t, err)
	
	jsonString := string(jsonBytes)
	assert.Contains(t, jsonString, "\"success\":true")
	assert.Contains(t, jsonString, "\"message\":\"Success\"")
	assert.Contains(t, jsonString, "\"id\":1")
}