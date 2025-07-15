package models

import (
	"encoding/json"
	"testing"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

func TestMemory_Validate(t *testing.T) {
	tests := []struct {
		name    string
		memory  Memory
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid memory",
			memory: Memory{
				Type:     TypeFact,
				Category: CategoryPersonal,
				Content:  "Test content",
			},
			wantErr: false,
		},
		{
			name: "Invalid type",
			memory: Memory{
				Type:     "invalid_type",
				Category: CategoryPersonal,
				Content:  "Test content",
			},
			wantErr: true,
			errMsg:  "invalid memory type",
		},
		{
			name: "Invalid category",
			memory: Memory{
				Type:     TypeFact,
				Category: "invalid_category",
				Content:  "Test content",
			},
			wantErr: true,
			errMsg:  "invalid memory category",
		},
		{
			name: "Empty content",
			memory: Memory{
				Type:     TypeFact,
				Category: CategoryPersonal,
				Content:  "",
			},
			wantErr: true,
			errMsg:  "content cannot be empty",
		},
		{
			name: "Empty type",
			memory: Memory{
				Type:     "",
				Category: CategoryPersonal,
				Content:  "Test content",
			},
			wantErr: true,
			errMsg:  "invalid memory type",
		},
		{
			name: "Empty category",
			memory: Memory{
				Type:     TypeFact,
				Category: "",
				Content:  "Test content",
			},
			wantErr: true,
			errMsg:  "invalid memory category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.memory.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemory_TableName(t *testing.T) {
	m := &Memory{}
	assert.Equal(t, "memories", m.TableName())
}

func TestMemory_IsValidType(t *testing.T) {
	tests := []struct {
		typ    string
		valid  bool
	}{
		{TypeFact, true},
		{TypeConversation, true},
		{TypeContext, true},
		{TypePreference, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidType(tt.typ))
		})
	}
}

func TestMemory_IsValidCategory(t *testing.T) {
	tests := []struct {
		category string
		valid    bool
	}{
		{CategoryPersonal, true},
		{CategoryProject, true},
		{CategoryBusiness, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidCategory(tt.category))
		})
	}
}

func TestMemory_WithEmbedding(t *testing.T) {
	// Create a test embedding
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) / 1536.0
	}

	memory := Memory{
		Type:      TypeFact,
		Category:  CategoryPersonal,
		Content:   "Test with embedding",
		Embedding: pgvector.NewVector(embedding),
		Tags:      pq.StringArray{"test", "embedding"},
	}

	// Test validation passes with embedding
	err := memory.Validate()
	assert.NoError(t, err)

	// Test embedding dimension
	assert.Equal(t, 1536, len(memory.Embedding.Slice()))
}

func TestMemory_WithMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"source": "test",
		"version": 1,
		"tags": []string{"test", "metadata"},
	}

	metadataJSON, err := json.Marshal(metadata)
	assert.NoError(t, err)

	memory := Memory{
		Type:     TypeFact,
		Category: CategoryPersonal,
		Content:  "Test with metadata",
		Metadata: json.RawMessage(metadataJSON),
	}

	// Test validation passes with metadata
	err = memory.Validate()
	assert.NoError(t, err)

	// Test metadata can be unmarshaled
	var result map[string]interface{}
	err = json.Unmarshal(memory.Metadata, &result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result["source"])
}