package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ksred/remember-me-mcp/internal/models"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

type CreateAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required" example:"Production API Key"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" example:"2024-12-31T23:59:59Z"`
}

type APIKeyResponse struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	Permissions []string   `json:"permissions"`
}

// registerHandler godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} UserInfo
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /auth/register [post]
func (s *Server) registerHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.authService.RegisterUser(req.Email, req.Password)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, UserInfo{
		ID:    user.ID,
		Email: user.Email,
	})
}

// loginHandler godoc
// @Summary Login user
// @Description Authenticate user and get JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (s *Server) loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.authService.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Generate JWT token
	expiresAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     expiresAt.Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to generate JWT token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Log the login activity
	details := map[string]interface{}{
		"email": user.Email,
	}
	go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityLogin, details, c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusOK, LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:    user.ID,
			Email: user.Email,
		},
	})
}

// listAPIKeysHandler godoc
// @Summary List API keys
// @Description Get all API keys for the authenticated user
// @Tags keys
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} APIKeyResponse
// @Failure 401 {object} ErrorResponse
// @Router /keys [get]
func (s *Server) listAPIKeysHandler(c *gin.Context) {
	user, ok := getUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	keys, err := s.authService.ListUserAPIKeys(user.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list API keys")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	response := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = APIKeyResponse{
			ID:          key.ID,
			Name:        key.Name,
			CreatedAt:   key.CreatedAt,
			ExpiresAt:   key.ExpiresAt,
			IsActive:    key.IsActive,
			Permissions: key.GetPermissions(),
		}
	}

	c.JSON(http.StatusOK, response)
}

// createAPIKeyHandler godoc
// @Summary Create API key
// @Description Create a new API key for authentication
// @Tags keys
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body CreateAPIKeyRequest true "API key details"
// @Success 201 {object} APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /keys [post]
func (s *Server) createAPIKeyHandler(c *gin.Context) {
	user, ok := getUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := s.authService.GenerateAPIKey(user.ID, req.Name, req.ExpiresAt)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	// Log the API key creation activity
	details := map[string]interface{}{
		"api_key_id": apiKey.ID,
		"name":       apiKey.Name,
	}
	go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityAPIKeyCreated, details, c.ClientIP(), c.GetHeader("User-Agent"))

	c.JSON(http.StatusCreated, APIKeyResponse{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		Key:         apiKey.Key, // Only shown once during creation
		CreatedAt:   apiKey.CreatedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		Permissions: apiKey.GetPermissions(),
	})
}

// deleteAPIKeyHandler godoc
// @Summary Delete API key
// @Description Delete an API key
// @Tags keys
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "API Key ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /keys/{id} [delete]
func (s *Server) deleteAPIKeyHandler(c *gin.Context) {
	user, ok := getUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key ID"})
		return
	}

	// Get the API key details before deletion for logging
	keys, err := s.authService.ListUserAPIKeys(user.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list API keys for deletion logging")
	}
	
	var keyName string
	for _, key := range keys {
		if key.ID == uint(keyID) {
			keyName = key.Name
			break
		}
	}

	if err := s.authService.DeleteAPIKey(user.ID, uint(keyID)); err != nil {
		if err.Error() == "API key not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		s.logger.Error().Err(err).Msg("Failed to delete API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	// Log the API key deletion activity
	details := map[string]interface{}{
		"api_key_id": uint(keyID),
		"name":       keyName,
	}
	go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityAPIKeyDeleted, details, c.ClientIP(), c.GetHeader("User-Agent"))

	c.Status(http.StatusNoContent)
}