package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// Memory represents a stored memory item in the database
type Memory struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	UserID    uint              `gorm:"not null;index;default:1" json:"user_id"`
	Type      string            `gorm:"index;not null" json:"type"`
	Category  string            `gorm:"index;not null" json:"category"`
	Content   string            `gorm:"type:text;not null" json:"content"`
	Priority  string            `gorm:"index;default:'medium'" json:"priority"`
	UpdateKey string            `gorm:"index" json:"update_key,omitempty"`
	Embedding pgvector.Vector   `gorm:"type:vector(1536);default:null" json:"-" swaggerignore:"true"`
	Tags      pq.StringArray    `gorm:"type:text[]" json:"tags" swaggertype:"array,string"`
	Metadata  json.RawMessage   `gorm:"type:jsonb" json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	
	// Associations
	User      *User             `gorm:"foreignKey:UserID" json:"-" swaggerignore:"true"`
}

// Valid memory types
const (
	TypeFact         = "fact"
	TypeConversation = "conversation"
	TypeContext      = "context"
	TypePreference   = "preference"
)

// Valid memory categories
const (
	CategoryPersonal = "personal"
	CategoryProject  = "project"
	CategoryBusiness = "business"
)

// TableName ensures consistent table naming
func (Memory) TableName() string {
	return "memories"
}

// Validate checks if the memory has valid Type and Category values
func (m *Memory) Validate() error {
	// Validate Type
	switch m.Type {
	case TypeFact, TypeConversation, TypeContext, TypePreference:
		// Valid type
	default:
		return errors.New("invalid memory type: must be one of fact, conversation, context, or preference")
	}

	// Validate Category
	switch m.Category {
	case CategoryPersonal, CategoryProject, CategoryBusiness:
		// Valid category
	default:
		return errors.New("invalid memory category: must be one of personal, project, or business")
	}

	// Validate required fields
	if m.Content == "" {
		return errors.New("content cannot be empty")
	}

	return nil
}

// BeforeCreate runs validation before saving a new memory
func (m *Memory) BeforeCreate(tx *gorm.DB) error {
	return m.Validate()
}

// BeforeUpdate runs validation before updating an existing memory
func (m *Memory) BeforeUpdate(tx *gorm.DB) error {
	return m.Validate()
}

// IsValidType checks if a given type string is valid
func IsValidType(t string) bool {
	switch t {
	case TypeFact, TypeConversation, TypeContext, TypePreference:
		return true
	default:
		return false
	}
}

// IsValidCategory checks if a given category string is valid
func IsValidCategory(c string) bool {
	switch c {
	case CategoryPersonal, CategoryProject, CategoryBusiness:
		return true
	default:
		return false
	}
}