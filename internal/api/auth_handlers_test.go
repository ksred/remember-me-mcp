package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/services"
	"github.com/ksred/remember-me-mcp/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	gin.SetMode(gin.TestMode)

	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&models.Memory{}, &models.User{}, &models.APIKey{})
	require.NoError(t, err)

	// Create test config
	cfg := &config.Config{
		JWT: config.JWT{
			Secret: "test-secret",
		},
		HTTP: config.HTTP{
			Port: 8080,
		},
		Memory: config.Memory{
			MaxMemories: 1000,
		},
	}

	// Create database wrapper
	testDB := &database.Database{}
	testDB.SetDB(db)

	// Create services
	logger := utils.NewLogger(utils.LoggerConfig{
		Level:  "error",
		Pretty: false,
	})
	embeddingService := services.NewMockEmbeddingService()
	memoryService := services.NewMemoryService(db, embeddingService, logger, map[string]interface{}{
		"memory_limit": cfg.Memory.MaxMemories,
	})

	// Create server
	server, err := NewServer(cfg, testDB, memoryService, logger)
	require.NoError(t, err)

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return server, cleanup
}

func TestRegisterHandler(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name       string
		body       RegisterRequest
		wantStatus int
		wantError  bool
	}{
		{
			name: "successful registration",
			body: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantStatus: http.StatusCreated,
			wantError:  false,
		},
		{
			name: "duplicate email",
			body: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantStatus: http.StatusConflict,
			wantError:  true,
		},
		{
			name: "short password",
			body: RegisterRequest{
				Email:    "test2@example.com",
				Password: "short",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "invalid email",
			body: RegisterRequest{
				Email:    "invalid-email",
				Password: "password123",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantError {
				var response ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Error)
			} else {
				var response UserInfo
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.body.Email, response.Email)
				assert.NotZero(t, response.ID)
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Register a user first
	_, err := server.authService.RegisterUser("test@example.com", "password123")
	require.NoError(t, err)

	tests := []struct {
		name       string
		body       LoginRequest
		wantStatus int
		wantError  bool
	}{
		{
			name: "successful login",
			body: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name: "wrong password",
			body: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  true,
		},
		{
			name: "non-existent user",
			body: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantError {
				var response ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Error)
			} else {
				var response LoginResponse
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, tt.body.Email, response.User.Email)
			}
		})
	}
}

func TestAPIKeyManagement(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Register and login a user
	user, err := server.authService.RegisterUser("test@example.com", "password123")
	require.NoError(t, err)

	// Login to get JWT token
	loginBody, _ := json.Marshal(LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	server.router.ServeHTTP(loginRec, loginReq)

	var loginResp LoginResponse
	err = json.Unmarshal(loginRec.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	jwtToken := loginResp.Token

	t.Run("create API key", func(t *testing.T) {
		body, _ := json.Marshal(CreateAPIKeyRequest{
			Name: "Test API Key",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/keys", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response APIKeyResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Test API Key", response.Name)
		assert.NotEmpty(t, response.Key)
		assert.True(t, response.IsActive)
	})

	t.Run("list API keys", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/keys", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response []APIKeyResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Empty(t, response[0].Key) // Key should not be returned in list
	})

	t.Run("delete API key", func(t *testing.T) {
		// First create an API key
		apiKey, err := server.authService.GenerateAPIKey(user.ID, "Test Delete", nil)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/keys/"+strconv.Itoa(int(apiKey.ID)), nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		rec := httptest.NewRecorder()

		server.router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestAuthMiddleware(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Register a user and create an API key
	user, err := server.authService.RegisterUser("test@example.com", "password123")
	require.NoError(t, err)

	apiKey, err := server.authService.GenerateAPIKey(user.ID, "Test Key", nil)
	require.NoError(t, err)

	tests := []struct {
		name       string
		authHeader string
		apiKey     string
		wantStatus int
	}{
		{
			name:       "no auth",
			authHeader: "",
			apiKey:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "valid API key",
			authHeader: "",
			apiKey:     apiKey.Key,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid API key",
			authHeader: "",
			apiKey:     "invalid-key",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid bearer format",
			authHeader: "InvalidFormat",
			apiKey:     "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/memories/stats", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()

			server.router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}