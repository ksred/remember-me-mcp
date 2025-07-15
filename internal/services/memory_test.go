package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ksred/remember-me-mcp/internal/models"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Create table manually without pgvector fields for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			category TEXT NOT NULL,
			content TEXT NOT NULL,
			embedding BLOB,
			tags TEXT,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Create indexes
	err = db.Exec(`CREATE INDEX idx_memories_type ON memories(type)`).Error
	require.NoError(t, err)
	
	err = db.Exec(`CREATE INDEX idx_memories_category ON memories(category)`).Error
	require.NoError(t, err)

	return db
}

// setupMemoryService creates a test memory service with an in-memory database
func setupMemoryService(t *testing.T, config map[string]interface{}) *MemoryService {
	db := setupTestDB(t)
	logger := zerolog.New(nil).Level(zerolog.Disabled)
	if config == nil {
		config = make(map[string]interface{})
	}
	// Pass nil for embedding service in tests
	return NewMemoryService(db, nil, logger, config)
}

func TestMemoryService_Store(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful creation", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		req := StoreRequest{
			Content:  "Test memory content",
			Category: models.CategoryPersonal,
			Type:     models.TypeFact,
			Metadata: map[string]interface{}{
				"source": "test",
			},
		}

		memory, err := service.Store(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, memory)
		assert.Equal(t, req.Content, memory.Content)
		assert.Equal(t, req.Category, memory.Category)
		assert.Equal(t, req.Type, memory.Type)
		assert.NotZero(t, memory.ID)
		assert.NotZero(t, memory.CreatedAt)
		assert.NotZero(t, memory.UpdatedAt)

		// Verify metadata
		var metadata map[string]interface{}
		err = json.Unmarshal(memory.Metadata, &metadata)
		assert.NoError(t, err)
		assert.Equal(t, "test", metadata["source"])
	})

	t.Run("Duplicate content handling - updates existing", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		// Create first memory
		req1 := StoreRequest{
			Content:  "Duplicate test content",
			Category: models.CategoryPersonal,
			Type:     models.TypeFact,
		}
		memory1, err := service.Store(ctx, req1)
		require.NoError(t, err)

		// Try to create duplicate with different category and type
		req2 := StoreRequest{
			Content:  "Duplicate test content",
			Category: models.CategoryBusiness,
			Type:     models.TypeContext,
		}
		memory2, err := service.Store(ctx, req2)
		assert.NoError(t, err)
		assert.NotNil(t, memory2)
		
		// Should have same ID (updated, not created new)
		assert.Equal(t, memory1.ID, memory2.ID)
		// But updated fields
		assert.Equal(t, models.CategoryBusiness, memory2.Category)
		assert.Equal(t, models.TypeContext, memory2.Type)
	})

	t.Run("Validation error - empty content", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		req := StoreRequest{
			Content:  "",
			Category: models.CategoryPersonal,
			Type:     models.TypeFact,
		}

		memory, err := service.Store(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, memory)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})

	t.Run("Validation error - invalid type", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		req := StoreRequest{
			Content:  "Test content",
			Category: models.CategoryPersonal,
			Type:     "invalid_type",
		}

		memory, err := service.Store(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, memory)
		assert.Contains(t, err.Error(), "invalid memory type")
	})

	t.Run("Validation error - invalid category", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		req := StoreRequest{
			Content:  "Test content",
			Category: "invalid_category",
			Type:     models.TypeFact,
		}

		memory, err := service.Store(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, memory)
		assert.Contains(t, err.Error(), "invalid memory category")
	})

	t.Run("Memory limit enforcement", func(t *testing.T) {
		config := map[string]interface{}{
			"memory_limit": 3,
		}
		service := setupMemoryService(t, config)

		// Create 4 memories (exceeding limit of 3)
		for i := 1; i <= 4; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		// Count should be 3 (limit enforced)
		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)

		// Verify oldest memory was deleted
		memories, err := service.Search(ctx, SearchRequest{})
		assert.NoError(t, err)
		assert.Len(t, memories, 3)
		
		// Check that "Memory 1" was deleted (oldest)
		for _, mem := range memories {
			assert.NotEqual(t, "Memory 1", mem.Content)
		}
	})

	t.Run("Memory limit with float64 config", func(t *testing.T) {
		config := map[string]interface{}{
			"memory_limit": float64(2), // JSON often decodes numbers as float64
		}
		service := setupMemoryService(t, config)

		// Create 3 memories
		for i := 1; i <= 3; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		// Count should be 2
		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func TestMemoryService_Search(t *testing.T) {
	ctx := context.Background()

	// Setup test data
	setupTestData := func(service *MemoryService) {
		testMemories := []StoreRequest{
			{
				Content:  "I love programming in Go",
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			},
			{
				Content:  "The project deadline is next Friday",
				Category: models.CategoryProject,
				Type:     models.TypeContext,
			},
			{
				Content:  "Client prefers email communication",
				Category: models.CategoryBusiness,
				Type:     models.TypePreference,
			},
			{
				Content:  "Go is a statically typed language",
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			},
		}

		for _, req := range testMemories {
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}
	}

	t.Run("Basic keyword search", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Query: "Go",
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 2)
		
		// Both memories containing "Go" should be returned
		for _, mem := range memories {
			assert.Contains(t, mem.Content, "Go")
		}
	})

	t.Run("Case insensitive search", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Query: "go",
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 2)
	})

	t.Run("Category filtering", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Category: models.CategoryPersonal,
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 2)
		
		for _, mem := range memories {
			assert.Equal(t, models.CategoryPersonal, mem.Category)
		}
	})

	t.Run("Type filtering", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Type: models.TypeFact,
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 2)
		
		for _, mem := range memories {
			assert.Equal(t, models.TypeFact, mem.Type)
		}
	})

	t.Run("Combined query and filters", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Query:    "Go",
			Category: models.CategoryPersonal,
			Type:     models.TypeFact,
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 2)
		
		for _, mem := range memories {
			assert.Contains(t, mem.Content, "Go")
			assert.Equal(t, models.CategoryPersonal, mem.Category)
			assert.Equal(t, models.TypeFact, mem.Type)
		}
	})

	t.Run("Limit application", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Limit: 2,
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 2)
	})

	t.Run("Default limit", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		
		// Create 150 memories
		for i := 0; i < 150; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		req := SearchRequest{}
		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 100) // Default limit
	})

	t.Run("Empty results", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		setupTestData(service)

		req := SearchRequest{
			Query: "nonexistent",
		}

		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Empty(t, memories)
	})

	t.Run("Order by created_at descending", func(t *testing.T) {
		service := setupMemoryService(t, nil)
		
		// Create memories with slight delays to ensure different timestamps
		for i := 1; i <= 3; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		req := SearchRequest{}
		memories, err := service.Search(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, memories, 3)
		
		// Verify newest first
		assert.Equal(t, "Memory 3", memories[0].Content)
		assert.Equal(t, "Memory 2", memories[1].Content)
		assert.Equal(t, "Memory 1", memories[2].Content)
	})
}

func TestMemoryService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful deletion", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		// Create a memory
		req := StoreRequest{
			Content:  "Test memory to delete",
			Category: models.CategoryPersonal,
			Type:     models.TypeFact,
		}
		memory, err := service.Store(ctx, req)
		require.NoError(t, err)

		// Delete it
		err = service.Delete(ctx, memory.ID)
		assert.NoError(t, err)

		// Verify it's gone
		_, err = service.GetByID(ctx, memory.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Not found error", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		err := service.Delete(ctx, 9999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory with ID '9999' not found")
	})

	t.Run("Invalid ID - zero", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		err := service.Delete(ctx, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMemoryService_Count(t *testing.T) {
	ctx := context.Background()

	t.Run("Correct count", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		// Initially empty
		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Add memories
		for i := 0; i < 5; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		// Check count
		count, err = service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)

		// Delete one
		memories, err := service.Search(ctx, SearchRequest{Limit: 1})
		require.NoError(t, err)
		require.Len(t, memories, 1)

		err = service.Delete(ctx, memories[0].ID)
		require.NoError(t, err)

		// Check count again
		count, err = service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), count)
	})

	t.Run("Empty database", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestMemoryService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("Found", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		// Create a memory
		req := StoreRequest{
			Content:  "Test memory content",
			Category: models.CategoryPersonal,
			Type:     models.TypeFact,
			Metadata: map[string]interface{}{
				"test": true,
			},
		}
		created, err := service.Store(ctx, req)
		require.NoError(t, err)

		// Get it by ID
		found, err := service.GetByID(ctx, created.ID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Content, found.Content)
		assert.Equal(t, created.Category, found.Category)
		assert.Equal(t, created.Type, found.Type)
	})

	t.Run("Not found", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		memory, err := service.GetByID(ctx, 9999)
		assert.Error(t, err)
		assert.Nil(t, memory)
		assert.Contains(t, err.Error(), "memory with ID '9999' not found")
	})

	t.Run("Invalid ID - zero", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		memory, err := service.GetByID(ctx, 0)
		assert.Error(t, err)
		assert.Nil(t, memory)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMemoryService_EnforceMemoryLimit(t *testing.T) {
	ctx := context.Background()

	t.Run("No limit configured", func(t *testing.T) {
		service := setupMemoryService(t, nil)

		// Create many memories
		for i := 0; i < 10; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		// All should remain
		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})

	t.Run("Invalid limit type", func(t *testing.T) {
		config := map[string]interface{}{
			"memory_limit": "invalid", // String instead of number
		}
		service := setupMemoryService(t, config)

		// Should not crash, just ignore the limit
		for i := 0; i < 5; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("Zero limit", func(t *testing.T) {
		config := map[string]interface{}{
			"memory_limit": 0,
		}
		service := setupMemoryService(t, config)

		// Should not enforce limit
		for i := 0; i < 5; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("Negative limit", func(t *testing.T) {
		config := map[string]interface{}{
			"memory_limit": -1,
		}
		service := setupMemoryService(t, config)

		// Should not enforce limit
		for i := 0; i < 5; i++ {
			req := StoreRequest{
				Content:  fmt.Sprintf("Memory %d", i),
				Category: models.CategoryPersonal,
				Type:     models.TypeFact,
			}
			_, err := service.Store(ctx, req)
			require.NoError(t, err)
		}

		count, err := service.Count(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestMemoryService_ComplexMetadata(t *testing.T) {
	ctx := context.Background()
	service := setupMemoryService(t, nil)

	// Test with complex nested metadata
	complexMetadata := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   123,
			"name": "Test User",
			"tags": []string{"tag1", "tag2", "tag3"},
		},
		"context": map[string]interface{}{
			"session_id": "abc-123",
			"timestamp":  "2024-01-01T10:00:00Z",
		},
		"scores": []float64{0.95, 0.87, 0.92},
	}

	req := StoreRequest{
		Content:  "Memory with complex metadata",
		Category: models.CategoryPersonal,
		Type:     models.TypeContext,
		Metadata: complexMetadata,
	}

	memory, err := service.Store(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, memory)

	// Retrieve and verify metadata
	retrieved, err := service.GetByID(ctx, memory.ID)
	assert.NoError(t, err)

	var retrievedMetadata map[string]interface{}
	err = json.Unmarshal(retrieved.Metadata, &retrievedMetadata)
	assert.NoError(t, err)

	// Verify nested structure
	user, ok := retrievedMetadata["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(123), user["id"]) // JSON numbers decode as float64
	assert.Equal(t, "Test User", user["name"])

	context, ok := retrievedMetadata["context"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "abc-123", context["session_id"])
}