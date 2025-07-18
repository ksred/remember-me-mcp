package models

import (
	"strings"
	"time"
	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	APIKeys   []APIKey       `gorm:"foreignKey:UserID" json:"-"`
}

type APIKey struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	User        User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Key         string         `gorm:"uniqueIndex;not null" json:"key"`
	Name        string         `gorm:"not null" json:"name"`
	LastUsedAt  *time.Time     `json:"last_used_at"`
	ExpiresAt   *time.Time     `json:"expires_at"`
	IsActive    bool           `gorm:"default:true;index" json:"is_active"`
	Permissions string         `gorm:"type:text" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// GetPermissions returns the permissions as a slice
func (a *APIKey) GetPermissions() []string {
	if a.Permissions == "" {
		return []string{}
	}
	return strings.Split(a.Permissions, ",")
}

// SetPermissions sets the permissions from a slice
func (a *APIKey) SetPermissions(perms []string) {
	a.Permissions = strings.Join(perms, ",")
}