package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ksred/remember-me-mcp/internal/models"
)

const (
	authTypeBearer = "bearer"
	authTypeAPIKey = "apikey"
	userContextKey = "user"
	authTypeKey    = "auth_type"
)

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API Key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			apiKeyObj, err := s.authService.ValidateAPIKey(apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
				c.Abort()
				return
			}
			
			c.Set(userContextKey, &apiKeyObj.User)
			c.Set(authTypeKey, authTypeAPIKey)
			c.Set("api_key", apiKeyObj)
			c.Next()
			return
		}

		// Check for Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(s.config.JWT.Secret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["user_id"].(float64)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				c.Abort()
				return
			}

			// Get user from database
			var user models.User
			if err := s.db.DB().First(&user, uint(userID)).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
				c.Abort()
				return
			}

			c.Set(userContextKey, &user)
			c.Set(authTypeKey, authTypeBearer)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
	}
}

func getUserFromContext(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get(userContextKey)
	if !exists {
		return nil, false
	}
	
	u, ok := user.(*models.User)
	return u, ok
}

func getAuthType(c *gin.Context) string {
	authType, _ := c.Get(authTypeKey)
	t, _ := authType.(string)
	return t
}