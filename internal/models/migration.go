package models

import (
	"time"
)

// Migration represents a database migration that has been applied
type Migration struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Version   string    `gorm:"uniqueIndex;not null" json:"version"`
	Name      string    `gorm:"not null" json:"name"`
	AppliedAt time.Time `json:"applied_at"`
}

// TableName ensures consistent table naming
func (Migration) TableName() string {
	return "schema_migrations"
}