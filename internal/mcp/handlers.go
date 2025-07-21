package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/services"
	"github.com/ksred/remember-me-mcp/internal/utils"
)

// Handler manages MCP tool handlers
type Handler struct {
	memoryService *services.MemoryService
	logger        zerolog.Logger
}

// NewHandler creates a new MCP handler
func NewHandler(memoryService *services.MemoryService, logger zerolog.Logger) *Handler {
	return &Handler{
		memoryService: memoryService,
		logger:        logger,
	}
}

// StoreMemoryRequest represents the request structure for storing memory
type StoreMemoryRequest struct {
	Type     string                 `json:"type"`
	Category string                 `json:"category"`
	Content  string                 `json:"content"`
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchMemoriesRequest represents the request structure for searching memories
type SearchMemoriesRequest struct {
	Query             string `json:"query"`
	Category          string `json:"category,omitempty"`
	Type              string `json:"type,omitempty"`
	Limit             int    `json:"limit,omitempty"`
	UseSemanticSearch bool   `json:"useSemanticSearch,omitempty"`
}

// UpdateMemoryRequest represents the request structure for updating memory
type UpdateMemoryRequest struct {
	ID       uint                   `json:"id"`
	Type     string                 `json:"type,omitempty"`
	Category string                 `json:"category,omitempty"`
	Content  string                 `json:"content,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Priority string                 `json:"priority,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DeleteMemoryRequest represents the request structure for deleting memory
type DeleteMemoryRequest struct {
	ID uint `json:"id"`
}

// Response structures

// StoreMemoryResponse represents the response after storing a memory
type StoreMemoryResponse struct {
	Success bool           `json:"success"`
	Memory  *models.Memory `json:"memory,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// SearchMemoriesResponse represents the response after searching memories
type SearchMemoriesResponse struct {
	Memories []*models.Memory `json:"memories"`
	Count    int              `json:"count"`
	Error    string           `json:"error,omitempty"`
}

// UpdateMemoryResponse represents the response after updating a memory
type UpdateMemoryResponse struct {
	Success bool           `json:"success"`
	Memory  *models.Memory `json:"memory,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// DeleteMemoryResponse represents the response after deleting a memory
type DeleteMemoryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// HandleStoreMemory handles the store memory MCP tool call
func (h *Handler) HandleStoreMemory(ctx context.Context, params json.RawMessage) (interface{}, error) {
	h.logger.Debug().RawJSON("params", params).Msg("handleStoreMemory called")

	// Parse request
	var req StoreMemoryRequest
	if err := json.Unmarshal(params, &req); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse store memory request")
		return StoreMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request format: %v", err),
		}, nil
	}

	// Validate request
	if req.Content == "" {
		h.logger.Warn().Msg("store memory request missing content")
		return StoreMemoryResponse{
			Success: false,
			Error:   "content is required",
		}, nil
	}

	// Check if tags are provided in metadata for backward compatibility
	if len(req.Tags) == 0 && req.Metadata != nil {
		if tagsInterface, exists := req.Metadata["tags"]; exists {
			switch tags := tagsInterface.(type) {
			case []interface{}:
				// Convert []interface{} to []string
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						req.Tags = append(req.Tags, tagStr)
					}
				}
			case []string:
				// Direct assignment if already []string
				req.Tags = tags
			}
			// Remove tags from metadata to avoid duplication
			delete(req.Metadata, "tags")
		}
	}

	if !models.IsValidType(req.Type) {
		h.logger.Warn().Str("type", req.Type).Msg("invalid memory type")
		return StoreMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid memory type '%s': must be one of fact, conversation, context, or preference", req.Type),
		}, nil
	}

	if !models.IsValidCategory(req.Category) {
		h.logger.Warn().Str("category", req.Category).Msg("invalid memory category")
		return StoreMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid memory category '%s': must be one of personal, project, or business", req.Category),
		}, nil
	}

	// First try automatic pattern detection
	autoMemories, err := h.memoryService.ProcessContentForMemory(ctx, req.Content)
	if err != nil {
		h.logger.Warn().Err(err).Msg("automatic pattern detection failed")
	}
	
	// If automatic detection found memories, use the first one as base
	var storeReq services.StoreRequest
	if len(autoMemories) > 0 {
		// Use detected memory as base but allow manual override
		detected := autoMemories[0]
		storeReq = services.StoreRequest{
			Content:   req.Content,
			Category:  req.Category,  // Manual override
			Type:      req.Type,      // Manual override
			Priority:  detected.Priority,
			UpdateKey: detected.UpdateKey,
			Tags:      req.Tags,
			Metadata:  req.Metadata,
		}
		
		h.logger.Info().
			Str("auto_priority", detected.Priority).
			Str("auto_update_key", detected.UpdateKey).
			Msg("using automatic pattern detection")
	} else {
		// No automatic detection, use manual input
		storeReq = services.StoreRequest{
			Content:   req.Content,
			Category:  req.Category,
			Type:      req.Type,
			Priority:  "medium", // Default priority
			UpdateKey: "",       // No update key
			Tags:      req.Tags,
			Metadata:  req.Metadata,
		}
	}

	// Call memory service
	memory, err := h.memoryService.Store(ctx, storeReq)

	if err != nil {
		h.logger.Error().Err(err).Msg("failed to store memory")
		return StoreMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to store memory: %v", err),
		}, nil
	}

	h.logger.Info().
		Uint("id", memory.ID).
		Str("type", memory.Type).
		Str("category", memory.Category).
		Msg("successfully stored memory")

	// Create a response without the embedding field to keep response size manageable
	responseMemory := &models.Memory{
		ID:        memory.ID,
		Type:      memory.Type,
		Category:  memory.Category,
		Content:   memory.Content,
		Priority:  memory.Priority,
		UpdateKey: memory.UpdateKey,
		Tags:      memory.Tags,
		Metadata:  memory.Metadata,
		CreatedAt: memory.CreatedAt,
		UpdatedAt: memory.UpdatedAt,
	}
	
	return StoreMemoryResponse{
		Success: true,
		Memory:  responseMemory,
	}, nil
}

// HandleSearchMemories handles the search memories MCP tool call
func (h *Handler) HandleSearchMemories(ctx context.Context, params json.RawMessage) (interface{}, error) {
	h.logger.Debug().RawJSON("params", params).Msg("handleSearchMemories called")

	// Parse request
	var req SearchMemoriesRequest
	if err := json.Unmarshal(params, &req); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse search memories request")
		return SearchMemoriesResponse{
			Memories: []*models.Memory{},
			Count:    0,
			Error:    fmt.Sprintf("invalid request format: %v", err),
		}, nil
	}

	// Validate request
	if req.Type != "" && !models.IsValidType(req.Type) {
		h.logger.Warn().Str("type", req.Type).Msg("invalid memory type")
		return SearchMemoriesResponse{
			Memories: []*models.Memory{},
			Count:    0,
			Error:    fmt.Sprintf("invalid memory type '%s': must be one of fact, conversation, context, or preference", req.Type),
		}, nil
	}

	if req.Category != "" && !models.IsValidCategory(req.Category) {
		h.logger.Warn().Str("category", req.Category).Msg("invalid memory category")
		return SearchMemoriesResponse{
			Memories: []*models.Memory{},
			Count:    0,
			Error:    fmt.Sprintf("invalid memory category '%s': must be one of personal, project, or business", req.Category),
		}, nil
	}

	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 100
	}

	// Default to semantic search when we have a query (this is why we have embeddings!)
	// This is the entire point of having vector search
	useSemanticSearch := req.Query != ""

	// Call memory service
	memories, err := h.memoryService.Search(ctx, services.SearchRequest{
		Query:             req.Query,
		Category:          req.Category,
		Type:              req.Type,
		Limit:             req.Limit,
		UseSemanticSearch: useSemanticSearch,
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("failed to search memories")
		return SearchMemoriesResponse{
			Memories: []*models.Memory{},
			Count:    0,
			Error:    fmt.Sprintf("failed to search memories: %v", err),
		}, nil
	}

	// Ensure we return an empty array instead of nil
	if memories == nil {
		memories = []*models.Memory{}
	}

	// Create response memories without embedding field to keep response size manageable
	responseMemories := make([]*models.Memory, len(memories))
	for i, memory := range memories {
		responseMemories[i] = &models.Memory{
			ID:        memory.ID,
			Type:      memory.Type,
			Category:  memory.Category,
			Content:   memory.Content,
			Priority:  memory.Priority,
			UpdateKey: memory.UpdateKey,
			Tags:      memory.Tags,
			Metadata:  memory.Metadata,
			CreatedAt: memory.CreatedAt,
			UpdatedAt: memory.UpdatedAt,
		}
	}

	h.logger.Info().
		Int("count", len(memories)).
		Str("query", req.Query).
		Str("category", req.Category).
		Str("type", req.Type).
		Bool("semantic", useSemanticSearch).
		Msg("successfully searched memories")

	return SearchMemoriesResponse{
		Memories: responseMemories,
		Count:    len(responseMemories),
	}, nil
}

// HandleUpdateMemory handles the update memory MCP tool call
func (h *Handler) HandleUpdateMemory(ctx context.Context, params json.RawMessage) (interface{}, error) {
	h.logger.Debug().RawJSON("params", params).Msg("handleUpdateMemory called")

	// Parse request
	var req UpdateMemoryRequest
	if err := json.Unmarshal(params, &req); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse update memory request")
		return UpdateMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request format: %v", err),
		}, nil
	}

	// Validate request
	if req.ID == 0 {
		h.logger.Warn().Msg("update memory request missing ID")
		return UpdateMemoryResponse{
			Success: false,
			Error:   "memory ID is required",
		}, nil
	}

	// Check if tags are provided in metadata for backward compatibility
	if len(req.Tags) == 0 && req.Metadata != nil {
		if tagsInterface, exists := req.Metadata["tags"]; exists {
			switch tags := tagsInterface.(type) {
			case []interface{}:
				// Convert []interface{} to []string
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						req.Tags = append(req.Tags, tagStr)
					}
				}
			case []string:
				// Direct assignment if already []string
				req.Tags = tags
			}
			// Remove tags from metadata to avoid duplication
			delete(req.Metadata, "tags")
		}
	}

	// Validate fields if provided
	if req.Type != "" && !models.IsValidType(req.Type) {
		h.logger.Warn().Str("type", req.Type).Msg("invalid memory type")
		return UpdateMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid memory type '%s': must be one of fact, conversation, context, or preference", req.Type),
		}, nil
	}

	if req.Category != "" && !models.IsValidCategory(req.Category) {
		h.logger.Warn().Str("category", req.Category).Msg("invalid memory category")
		return UpdateMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid memory category '%s': must be one of personal, project, or business", req.Category),
		}, nil
	}

	// Call memory service
	memory, err := h.memoryService.Update(ctx, req.ID, services.UpdateRequest{
		Content:  req.Content,
		Category: req.Category,
		Type:     req.Type,
		Priority: req.Priority,
		Tags:     req.Tags,
		Metadata: req.Metadata,
	})

	if err != nil {
		// Check if it's a not found error
		if utils.IsNotFoundError(err) {
			h.logger.Warn().Uint("id", req.ID).Msg("memory not found")
			return UpdateMemoryResponse{
				Success: false,
				Error:   fmt.Sprintf("memory with ID %d not found", req.ID),
			}, nil
		}

		h.logger.Error().Err(err).Uint("id", req.ID).Msg("failed to update memory")
		return UpdateMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to update memory: %v", err),
		}, nil
	}

	h.logger.Info().
		Uint("id", memory.ID).
		Msg("successfully updated memory")

	// Create a response without the embedding field to keep response size manageable
	responseMemory := &models.Memory{
		ID:        memory.ID,
		Type:      memory.Type,
		Category:  memory.Category,
		Content:   memory.Content,
		Priority:  memory.Priority,
		UpdateKey: memory.UpdateKey,
		Tags:      memory.Tags,
		Metadata:  memory.Metadata,
		CreatedAt: memory.CreatedAt,
		UpdatedAt: memory.UpdatedAt,
	}

	return UpdateMemoryResponse{
		Success: true,
		Memory:  responseMemory,
	}, nil
}

// HandleDeleteMemory handles the delete memory MCP tool call
func (h *Handler) HandleDeleteMemory(ctx context.Context, params json.RawMessage) (interface{}, error) {
	h.logger.Debug().RawJSON("params", params).Msg("handleDeleteMemory called")

	// Parse request
	var req DeleteMemoryRequest
	if err := json.Unmarshal(params, &req); err != nil {
		h.logger.Error().Err(err).Msg("failed to parse delete memory request")
		return DeleteMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid request format: %v", err),
		}, nil
	}

	// Validate request
	if req.ID == 0 {
		h.logger.Warn().Msg("delete memory request missing ID")
		return DeleteMemoryResponse{
			Success: false,
			Error:   "memory ID is required",
		}, nil
	}

	// Call memory service
	err := h.memoryService.Delete(ctx, req.ID)
	if err != nil {
		// Check if it's a not found error
		if utils.IsNotFoundError(err) {
			h.logger.Warn().Uint("id", req.ID).Msg("memory not found")
			return DeleteMemoryResponse{
				Success: false,
				Error:   fmt.Sprintf("memory with ID %d not found", req.ID),
			}, nil
		}

		h.logger.Error().Err(err).Uint("id", req.ID).Msg("failed to delete memory")
		return DeleteMemoryResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to delete memory: %v", err),
		}, nil
	}

	h.logger.Info().
		Uint("id", req.ID).
		Msg("successfully deleted memory")

	return DeleteMemoryResponse{
		Success: true,
		Message: fmt.Sprintf("Memory with ID %d successfully deleted", req.ID),
	}, nil
}

// ToJSON methods for request types

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

// ToJSON methods for response types

// ToJSON converts the response to JSON
func (r *StoreMemoryResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToJSON converts the response to JSON
func (r *SearchMemoriesResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToJSON converts the response to JSON
func (r *DeleteMemoryResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}