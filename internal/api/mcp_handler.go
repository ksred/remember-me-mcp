package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ksred/remember-me-mcp/internal/mcp"
	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/services"
	mcpTypes "github.com/mark3labs/mcp-go/mcp"
)

// MCPRequest represents a JSON-RPC 2.0 request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// MCPResponse represents a JSON-RPC 2.0 response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// MCPError represents a JSON-RPC 2.0 error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// HandleMCP processes MCP protocol requests over HTTP
func (s *Server) HandleMCP(c *gin.Context) {
	// Debug: Log raw request body
	bodyBytes, _ := c.GetRawData()
	s.logger.Debug().
		Int("body_length", len(bodyBytes)).
		Str("body_raw", string(bodyBytes)).
		Msg("HandleMCP received raw request")

	// Restore body for ShouldBindJSON
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req MCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error().
			Err(err).
			Str("body", string(bodyBytes)).
			Msg("failed to bind JSON request")
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			Error: &MCPError{
				Code:    ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
			ID: nil,
		})
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			Error: &MCPError{
				Code:    InvalidRequest,
				Message: "Invalid Request",
				Data:    "jsonrpc must be 2.0",
			},
			ID: req.ID,
		})
		return
	}

	// Get user from context (set by auth middleware)
	user, exists := getUserFromContext(c)
	if !exists || user == nil {
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			Error: &MCPError{
				Code:    InternalError,
				Message: "Authentication required",
			},
			ID: req.ID,
		})
		return
	}

	// Create a scoped memory service for this user
	scopedMemoryService := s.createScopedMemoryService(user.ID)

	// Route the request based on method
	var result interface{}
	var err error

	switch req.Method {
	case "initialize":
		result, err = s.handleMCPInitialize(req.Params)
	case "tools/list":
		result, err = s.handleMCPListTools()
	case "tools/call":
		result, err = s.handleMCPCallTool(c.Request.Context(), req.Params, scopedMemoryService, user, c)
	case "resources/list":
		result, err = s.handleMCPListResources()
	case "resources/read":
		result, err = s.handleMCPReadResource(c.Request.Context(), req.Params, scopedMemoryService)
	default:
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			Error: &MCPError{
				Code:    MethodNotFound,
				Message: "Method not found",
				Data:    fmt.Sprintf("Unknown method: %s", req.Method),
			},
			ID: req.ID,
		})
		return
	}

	if err != nil {
		s.logger.Error().Err(err).Str("method", req.Method).Msg("MCP method error")
		c.JSON(http.StatusOK, MCPResponse{
			JSONRPC: "2.0",
			Error: &MCPError{
				Code:    InternalError,
				Message: "Internal error",
				Data:    err.Error(),
			},
			ID: req.ID,
		})
		return
	}

	c.JSON(http.StatusOK, MCPResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

// handleMCPInitialize handles the initialize method
func (s *Server) handleMCPInitialize(params json.RawMessage) (interface{}, error) {
	// Parse initialize params if needed
	var initParams struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	
	if err := json.Unmarshal(params, &initParams); err != nil {
		return nil, fmt.Errorf("invalid initialize params: %w", err)
	}

	return map[string]interface{}{
		"protocolVersion": "0.1.0",
		"serverInfo": map[string]interface{}{
			"name":    "remember-me-mcp",
			"version": "1.0.0",
		},
		"capabilities": map[string]interface{}{
			"tools":     true,
			"resources": true,
		},
	}, nil
}

// handleMCPListTools returns the list of available tools
func (s *Server) handleMCPListTools() (interface{}, error) {
	tools := []mcpTypes.Tool{
		{
			Name:        "store_memory",
			Description: "Store important information that the user wants remembered. Use when user says 'remember that...', shares personal preferences ('I prefer...', 'I like...'), provides personal information ('I work at...', 'I live in...'), mentions ongoing projects ('I'm working on...'), or shares important facts they'll need later.",
			InputSchema: mcpTypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Type of memory: fact, conversation, context, or preference",
						"enum":        []string{"fact", "conversation", "context", "preference"},
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category of memory: personal, project, or business",
						"enum":        []string{"personal", "project", "business"},
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content of the memory to store",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "Optional tags to categorize the memory",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Optional metadata for the memory",
					},
				},
				Required: []string{"type", "category", "content"},
			},
		},
		{
			Name:        "store_memories_bulk",
			Description: "Store multiple memories at once. Use when the user wants to remember multiple things in a single request.",
			InputSchema: mcpTypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"memories": map[string]interface{}{
						"type":        "array",
						"description": "Array of memories to store",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "Type of memory: fact, conversation, context, or preference",
									"enum":        []string{"fact", "conversation", "context", "preference"},
								},
								"category": map[string]interface{}{
									"type":        "string",
									"description": "Category of memory: personal, project, or business",
									"enum":        []string{"personal", "project", "business"},
								},
								"content": map[string]interface{}{
									"type":        "string",
									"description": "The content of the memory to store",
								},
								"tags": map[string]interface{}{
									"type":        "array",
									"description": "Optional tags to categorize the memory",
									"items": map[string]interface{}{
										"type": "string",
									},
								},
								"metadata": map[string]interface{}{
									"type":        "object",
									"description": "Optional metadata for the memory",
								},
							},
							"required": []string{"type", "category", "content"},
						},
					},
				},
				Required: []string{"memories"},
			},
		},
		{
			Name:        "search_memories",
			Description: "Search for previously stored memories. Use when user asks 'what do you remember about...', 'what did I say about...', 'what are my preferences for...', 'what projects am I working on...', or needs to recall any previously shared information.",
			InputSchema: mcpTypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category: personal, project, or business",
						"enum":        []string{"personal", "project", "business"},
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Filter by type: fact, conversation, context, or preference",
						"enum":        []string{"fact", "conversation", "context", "preference"},
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return (default: 100)",
						"minimum":     1,
						"maximum":     1000,
					},
					"useSemanticSearch": map[string]interface{}{
						"type":        "boolean",
						"description": "Use semantic search (default: true)",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "update_memory",
			Description: "Update an existing memory by ID. Provide only the fields you want to update.",
			InputSchema: mcpTypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "integer",
						"description": "ID of the memory to update",
						"minimum":     1,
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Type of memory: fact, conversation, context, or preference",
						"enum":        []string{"fact", "conversation", "context", "preference"},
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category of memory: personal, project, or business",
						"enum":        []string{"personal", "project", "business"},
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The new content of the memory",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "Priority level: low, medium, or high",
						"enum":        []string{"low", "medium", "high"},
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "Tags to categorize the memory",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Metadata for the memory",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "delete_memory",
			Description: "Delete a memory by ID",
			InputSchema: mcpTypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "integer",
						"description": "ID of the memory to delete",
						"minimum":     1,
					},
				},
				Required: []string{"id"},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleMCPCallTool handles tool invocations
func (s *Server) handleMCPCallTool(ctx context.Context, params json.RawMessage, memoryService *services.MemoryService, user *models.User, c *gin.Context) (interface{}, error) {
	// Debug logging for tool call params
	s.logger.Debug().
		Int("params_length", len(params)).
		Str("params_raw", string(params)).
		Msg("handleMCPCallTool received params")

	var callParams struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(params, &callParams); err != nil {
		s.logger.Error().
			Err(err).
			Str("params_string", string(params)).
			Msg("failed to unmarshal tool call params")
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}

	// Log the parsed tool call details
	s.logger.Debug().
		Str("tool_name", callParams.Name).
		Int("arguments_length", len(callParams.Arguments)).
		Str("arguments_raw", string(callParams.Arguments)).
		Msg("parsed tool call params")

	// Check if arguments are missing or empty
	if len(callParams.Arguments) == 0 || string(callParams.Arguments) == "null" {
		errMsg := fmt.Sprintf("tool '%s' called without arguments. Arguments are required for all tool calls.", callParams.Name)
		s.logger.Error().Str("tool", callParams.Name).Msg(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	// Create a handler with the scoped memory service
	handler := mcp.NewHandler(memoryService, s.logger)

	var result interface{}
	var err error

	switch callParams.Name {
	case "store_memory":
		s.logger.Debug().Msg("routing to HandleStoreMemory")
		result, err = handler.HandleStoreMemory(ctx, callParams.Arguments)
	case "store_memories_bulk":
		s.logger.Debug().Msg("routing to HandleStoreMemoriesBulk")
		result, err = handler.HandleStoreMemoriesBulk(ctx, callParams.Arguments)
	case "search_memories":
		result, err = handler.HandleSearchMemories(ctx, callParams.Arguments)
		// Log search activity if successful (but not for wildcard queries)
		if err == nil && result != nil && user != nil {
			go func() {
				// Parse the request to get search details
				var searchReq mcp.SearchMemoriesRequest
				if unmarshalErr := json.Unmarshal(callParams.Arguments, &searchReq); unmarshalErr == nil {
					// Skip logging wildcard searches
					if searchReq.Query == "*" || searchReq.Query == "" {
						return
					}
					
					// Get result count
					resultCount := 0
					if searchResp, ok := result.(mcp.SearchMemoriesResponse); ok {
						resultCount = searchResp.Count
					}
					
					details := map[string]interface{}{
						"query":               searchReq.Query,
						"category":            searchReq.Category,
						"type":                searchReq.Type,
						"limit":               searchReq.Limit,
						"use_semantic_search": searchReq.Query != "", // MCP uses semantic search when query is present
						"results_count":       resultCount,
						"source":              "mcp", // Mark as MCP search
					}
					
					// Use background context for async logging
					if logErr := s.activityService.LogActivity(
						context.Background(),
						user.ID,
						models.ActivityMemorySearch,
						details,
						c.ClientIP(),
						c.GetHeader("User-Agent"),
					); logErr != nil {
						s.logger.Error().
							Err(logErr).
							Uint("user_id", user.ID).
							Msg("Failed to log MCP search activity")
					} else {
						s.logger.Debug().
							Uint("user_id", user.ID).
							Str("query", searchReq.Query).
							Int("results_count", resultCount).
							Msg("MCP search activity logged")
					}
				}
			}()
		}
	case "update_memory":
		result, err = handler.HandleUpdateMemory(ctx, callParams.Arguments)
	case "delete_memory":
		result, err = handler.HandleDeleteMemory(ctx, callParams.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", callParams.Name)
	}

	if err != nil {
		return nil, err
	}

	// Convert result to the expected format
	var content []mcpTypes.Content
	
	// Marshal result to JSON for text content
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	content = append(content, mcpTypes.TextContent{
		Type: "text",
		Text: string(resultJSON),
	})

	return map[string]interface{}{
		"content": content,
	}, nil
}

// handleMCPListResources returns the list of available resources
func (s *Server) handleMCPListResources() (interface{}, error) {
	resources := []mcpTypes.Resource{
		{
			URI:         "memory://stats",
			Name:        "Memory Statistics",
			Description: "Get statistics about stored memories",
			MIMEType:    "application/json",
		},
	}

	return map[string]interface{}{
		"resources": resources,
	}, nil
}

// handleMCPReadResource handles resource reads
func (s *Server) handleMCPReadResource(ctx context.Context, params json.RawMessage, memoryService *services.MemoryService) (interface{}, error) {
	var readParams struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(params, &readParams); err != nil {
		return nil, fmt.Errorf("invalid resource read params: %w", err)
	}

	if readParams.URI != "memory://stats" {
		return nil, fmt.Errorf("unknown resource: %s", readParams.URI)
	}

	stats, err := memoryService.GetMemoryStats(ctx)
	if err != nil {
		return nil, err
	}

	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      readParams.URI,
				"mimeType": "application/json",
				"text":     string(statsJSON),
			},
		},
	}, nil
}

// createScopedMemoryService creates a memory service scoped to a specific user
func (s *Server) createScopedMemoryService(userID uint) *services.MemoryService {
	// Build config with memory limit and encryption service
	serviceConfig := map[string]interface{}{
		"memory_limit": s.config.Memory.MaxMemories,
		"similarity_threshold": s.config.Memory.SimilarityThreshold,
	}
	
	// Pass encryption service if available
	if encSvc := s.memoryService.GetEncryptionService(); encSvc != nil {
		serviceConfig["encryption_service"] = encSvc
	}
	
	// Create a user-scoped memory service for this request
	return services.NewMemoryServiceWithUser(
		s.db.DB(),
		s.memoryService.GetEmbeddingService(),
		s.logger,
		serviceConfig,
		userID,
	)
}