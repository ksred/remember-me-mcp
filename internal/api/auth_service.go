package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db     *database.Database
	logger zerolog.Logger
}

func NewAuthService(db *database.Database, logger zerolog.Logger) *AuthService {
	return &AuthService{
		db:     db,
		logger: logger,
	}
}

func (s *AuthService) RegisterUser(email, password string) (*models.User, error) {
	// Validate email and password
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters long")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
	}

	// Save to database
	if err := s.db.DB().Create(user).Error; err != nil {
		// Check for unique constraint violation
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return nil, errors.New("email already exists")
		}
		return nil, err
	}

	return user, nil
}

func (s *AuthService) AuthenticateUser(email, password string) (*models.User, error) {
	var user models.User
	
	if err := s.db.DB().Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user, nil
}

func (s *AuthService) GenerateAPIKey(userID uint, name string, expiresAt *time.Time) (*models.APIKey, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, err
	}
	
	keyString := hex.EncodeToString(keyBytes)

	apiKey := &models.APIKey{
		UserID:    userID,
		Key:       keyString,
		Name:      name,
		ExpiresAt: expiresAt,
		IsActive:  true,
	}
	apiKey.SetPermissions([]string{"memory:read", "memory:write", "memory:delete"})

	if err := s.db.DB().Create(apiKey).Error; err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (s *AuthService) ValidateAPIKey(key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	
	// First find the API key
	err := s.db.DB().
		Where("key = ? AND is_active = ?", key, true).
		First(&apiKey).Error
		
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid API key")
		}
		return nil, err
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("API key expired")
	}

	// Load the associated user
	if err := s.db.DB().First(&apiKey.User, apiKey.UserID).Error; err != nil {
		return nil, err
	}

	// Update last used timestamp
	now := time.Now()
	apiKey.LastUsedAt = &now
	s.db.DB().Model(&apiKey).Update("last_used_at", now)

	return &apiKey, nil
}

func (s *AuthService) ListUserAPIKeys(userID uint) ([]models.APIKey, error) {
	var keys []models.APIKey
	
	err := s.db.DB().
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&keys).Error
		
	return keys, err
}

func (s *AuthService) DeleteAPIKey(userID uint, keyID uint) error {
	result := s.db.DB().
		Where("id = ? AND user_id = ?", keyID, userID).
		Delete(&models.APIKey{})
		
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	
	return nil
}