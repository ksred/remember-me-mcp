package models

import (
	"time"
	"gorm.io/gorm"
)

// ActivityLog represents user activity tracking
type ActivityLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Type      string         `gorm:"not null;index" json:"type"` // memory_stored, memory_search, memory_deleted, api_key_created, login
	Details   map[string]interface{} `gorm:"type:jsonb" json:"details,omitempty" swaggertype:"object"`
	IPAddress string         `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent string         `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt time.Time      `gorm:"index" json:"timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// PerformanceMetric represents system performance tracking
type PerformanceMetric struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Endpoint     string    `gorm:"not null;index" json:"endpoint"`
	Method       string    `gorm:"not null" json:"method"`
	ResponseTime int       `gorm:"not null" json:"response_time_ms"` // in milliseconds
	StatusCode   int       `gorm:"not null" json:"status_code"`
	CreatedAt    time.Time `gorm:"index" json:"timestamp"`
}

// Activity type constants
const (
	ActivityMemoryStored  = "memory_stored"
	ActivityMemorySearch  = "memory_search"
	ActivityMemoryDeleted = "memory_deleted"
	ActivityAPIKeyCreated = "api_key_created"
	ActivityAPIKeyDeleted = "api_key_deleted"
	ActivityLogin         = "login"
)