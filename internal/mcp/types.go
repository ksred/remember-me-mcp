package mcp

import (
	"encoding/json"
)

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


// ToJSON converts the response to JSON
func (r *MemoryResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}