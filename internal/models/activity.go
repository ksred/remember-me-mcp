package models

import (
	"encoding/json"
	"time"
	"gorm.io/gorm"
)

// ActivityLog represents user activity tracking
type ActivityLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Type      string         `gorm:"not null;index" json:"type"` // memory_stored, memory_search, memory_deleted, api_key_created, login
	Details   json.RawMessage `gorm:"type:jsonb" json:"details,omitempty" swaggertype:"object"`
	IPAddress string         `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent string         `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt time.Time      `gorm:"index" json:"timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// GetDetailsMap unmarshals the Details JSON into a map
func (a *ActivityLog) GetDetailsMap() (map[string]interface{}, error) {
	if a.Details == nil || len(a.Details) == 0 {
		return nil, nil
	}
	
	var details map[string]interface{}
	if err := json.Unmarshal(a.Details, &details); err != nil {
		return nil, err
	}
	return details, nil
}

// SetDetailsFromMap marshals a map into the Details JSON
func (a *ActivityLog) SetDetailsFromMap(details map[string]interface{}) error {
	if details == nil {
		a.Details = nil
		return nil
	}
	
	data, err := json.Marshal(details)
	if err != nil {
		return err
	}
	a.Details = data
	return nil
}

// PerformanceMetric represents system performance tracking
type PerformanceMetric struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Endpoint     string    `gorm:"not null;index" json:"endpoint"`
	Method       string    `gorm:"not null" json:"method"`
	DurationMs   int       `gorm:"column:duration_ms;not null" json:"response_time_ms"` // in milliseconds
	ResponseTime int       `gorm:"column:response_time;not null;-:migration" json:"-"`  // Legacy column, kept for compatibility
	StatusCode   int       `gorm:"not null" json:"status_code"`
	UserID       *uint     `gorm:"index" json:"user_id,omitempty"`
	User         *User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Error        *string   `gorm:"type:text" json:"error,omitempty"`
	CreatedAt    time.Time `gorm:"index" json:"timestamp"`
}

// TableName specifies the table name for PerformanceMetric
func (PerformanceMetric) TableName() string {
	return "performance_metrics"
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