package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ksred/remember-me-mcp/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryEndpoints(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Register a user and create an API key
	user, err := server.authService.RegisterUser("test@example.com", "password123")
	require.NoError(t, err)

	apiKey, err := server.authService.GenerateAPIKey(user.ID, "Test Key", nil)
	require.NoError(t, err)

	var createdMemoryID uint

	t.Run("store memory", func(t *testing.T) {
		metadata := map[string]interface{}{"source": "test"}
		body, _ := json.Marshal(mcp.StoreMemoryRequest{
			Type:     "fact",
			Category: "personal",
			Content:  "Test memory content",
			Metadata: metadata,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/memories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response mcp.StoreMemoryResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.NotNil(t, response.Memory)
		assert.NotZero(t, response.Memory.ID)

		createdMemoryID = response.Memory.ID
	})

	t.Run("store memory with invalid type", func(t *testing.T) {
		body, _ := json.Marshal(mcp.StoreMemoryRequest{
			Type:     "invalid-type",
			Category: "personal",
			Content:  "Test memory content",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/memories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		var response ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.Error)
	})

	t.Run("search memories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memories?query=test", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response mcp.SearchMemoriesResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, response.Count, 1)
		assert.NotEmpty(t, response.Memories)
	})

	t.Run("search memories with filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memories?query=test&category=personal&type=fact&limit=10", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response mcp.SearchMemoriesResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(response.Memories), 0)
	})

	t.Run("search memories without query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memories", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "query parameter is required", response.Error)
	})

	t.Run("get memory stats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memories/stats", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "total_count")
		assert.Contains(t, response, "by_type")
		assert.Contains(t, response, "by_category")
	})

	t.Run("delete memory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/memories/"+strconv.Itoa(int(createdMemoryID)), nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response mcp.DeleteMemoryResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Equal(t, "Memory deleted successfully", response.Message)
	})

	t.Run("delete non-existent memory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/memories/999999", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "memory not found", response.Error)
	})

	t.Run("delete memory with invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/memories/invalid-id", nil)
		req.Header.Set("X-API-Key", apiKey.Key)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid memory ID", response.Error)
	})
}

func TestMemoryEndpointsWithoutAuth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{
			name:   "store memory",
			method: http.MethodPost,
			path:   "/api/v1/memories",
			body: mcp.StoreMemoryRequest{
				Type:     "fact",
				Category: "personal",
				Content:  "Test",
			},
		},
		{
			name:   "search memories",
			method: http.MethodGet,
			path:   "/api/v1/memories?query=test",
			body:   nil,
		},
		{
			name:   "delete memory",
			method: http.MethodDelete,
			path:   "/api/v1/memories/1",
			body:   nil,
		},
		{
			name:   "memory stats",
			method: http.MethodGet,
			path:   "/api/v1/memories/stats",
			body:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)

			var response ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.NotEmpty(t, response.Error)
		})
	}
}