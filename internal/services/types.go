package services

import (
	"encoding/json"
)

// StoreMemoryRequest represents a request to store a new memory
type StoreMemoryRequest struct {
	Type     string                 `json:"type" validate:"required,oneof=fact conversation context preference"`
	Category string                 `json:"category" validate:"required,oneof=personal project business"`
	Content  string                 `json:"content" validate:"required,min=1"`
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchMemoriesRequest represents a request to search memories
type SearchMemoriesRequest struct {
	Query             string `json:"query" validate:"required,min=1"`
	Category          string `json:"category,omitempty" validate:"omitempty,oneof=personal project business"`
	Type              string `json:"type,omitempty" validate:"omitempty,oneof=fact conversation context preference"`
	Limit             int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	UseSemanticSearch bool   `json:"use_semantic_search"`
}

// SetDefaults sets default values for SearchMemoriesRequest
func (r *SearchMemoriesRequest) SetDefaults() {
	if r.Limit == 0 {
		r.Limit = 10
	}
	if !r.UseSemanticSearch {
		r.UseSemanticSearch = true
	}
}

// DeleteMemoryRequest represents a request to delete a memory
type DeleteMemoryRequest struct {
	ID uint `json:"id" validate:"required,min=1"`
}

// MemoryResponse represents a standard response for memory operations
type MemoryResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    interface{}     `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Meta    *ResponseMeta   `json:"meta,omitempty"`
}

// ResponseMeta contains metadata about the response
type ResponseMeta struct {
	Count      int    `json:"count,omitempty"`
	TotalCount int    `json:"total_count,omitempty"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	SearchType string `json:"search_type,omitempty"`
}

// NewSuccessResponse creates a successful memory response
func NewSuccessResponse(message string, data interface{}) *MemoryResponse {
	return &MemoryResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates an error memory response
func NewErrorResponse(error string) *MemoryResponse {
	return &MemoryResponse{
		Success: false,
		Error:   error,
	}
}

// ToJSON converts the request to JSON
func (r *StoreMemoryRequest) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToJSON converts the request to JSON
func (r *SearchMemoriesRequest) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToJSON converts the request to JSON
func (r *DeleteMemoryRequest) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToJSON converts the response to JSON
func (r *MemoryResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}