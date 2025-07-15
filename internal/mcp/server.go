package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/ksred/remember-me-mcp/internal/services"
)

// Server wraps the MCP server with our application logic
type Server struct {
	mcpServer *server.MCPServer
	handler   *Handler
	logger    zerolog.Logger
}

// NewServer creates a new MCP server instance
func NewServer(memoryService *services.MemoryService, logger zerolog.Logger) (*Server, error) {
	// Create the MCP server
	mcpServer := server.NewMCPServer(
		"remember-me",
		"1.0.0",
		server.WithLogging(),
	)

	// Create handler
	handler := NewHandler(memoryService, logger)

	s := &Server{
		mcpServer: mcpServer,
		handler:   handler,
		logger:    logger,
	}

	// Register handlers
	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s, nil
}

// Serve starts the MCP server
func (s *Server) Serve(ctx context.Context) error {
	s.logger.Debug().Msg("Starting MCP server ServeStdio")
	err := server.ServeStdio(s.mcpServer)
	if err != nil {
		s.logger.Error().Err(err).Msg("MCP server ServeStdio error")
	}
	return err
}

// registerTools registers MCP tools
func (s *Server) registerTools() {
	// Store memory tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "store_memory",
		Description: "Store important information that the user wants remembered. Use when user says 'remember that...', shares personal preferences ('I prefer...', 'I like...'), provides personal information ('I work at...', 'I live in...'), mentions ongoing projects ('I'm working on...'), or shares important facts they'll need later.",
		InputSchema: mcp.ToolInputSchema{
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
				"metadata": map[string]interface{}{
					"type":        "object",
					"description": "Optional metadata for the memory",
				},
			},
			Required: []string{"type", "category", "content"},
		},
	}, s.createStoreMemoryHandler())

	// Search memories tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "search_memories",
		Description: "Search for previously stored memories. Use when user asks 'what do you remember about...', 'what did I say about...', 'what are my preferences for...', 'what projects am I working on...', or needs to recall any previously shared information.",
		InputSchema: mcp.ToolInputSchema{
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
	}, s.createSearchMemoriesHandler())

	// Delete memory tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "delete_memory",
		Description: "Delete a memory by ID",
		InputSchema: mcp.ToolInputSchema{
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
	}, s.createDeleteMemoryHandler())

	s.logger.Info().Int("count", 3).Msg("Registered MCP tools")
}

// registerResources registers MCP resources
func (s *Server) registerResources() {
	// Memory statistics resource
	s.mcpServer.AddResource(mcp.Resource{
		URI:         "memory://stats",
		Name:        "Memory Statistics",
		Description: "Get statistics about stored memories",
		MIMEType:    "application/json",
	}, s.createMemoryStatsHandler())

	s.logger.Info().Int("count", 1).Msg("Registered MCP resources")
}

// registerPrompts registers MCP prompts
func (s *Server) registerPrompts() {
	// Example prompt for memory storage
	s.mcpServer.AddPrompt(mcp.Prompt{
		Name:        "store_fact",
		Description: "Template for storing a factual memory",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "fact",
				Description: "The fact to store",
				Required:    true,
			},
			{
				Name:        "category",
				Description: "Category for the fact",
				Required:    false,
			},
		},
	}, s.createStoreFactHandler())

	s.logger.Info().Int("count", 1).Msg("Registered MCP prompts")
}

// Handler creation functions for MCP tools

func (s *Server) createStoreMemoryHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.logger.Debug().Msg("Store memory tool handler called")
		
		// Convert arguments to JSON for the handler
		jsonData, err := json.Marshal(request.GetArguments())
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to parse arguments: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Call the existing handler
		result, err := s.handler.handleStoreMemory(ctx, jsonData)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Convert result to JSON string
		response := result.(StoreMemoryResponse)
		resultJSON, err := response.ToJSON()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to marshal result: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		}, nil
	}
}

func (s *Server) createSearchMemoriesHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert arguments to JSON for the handler
		jsonData, err := json.Marshal(request.GetArguments())
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to parse arguments: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Call the existing handler
		result, err := s.handler.handleSearchMemories(ctx, jsonData)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Convert result to JSON string
		response := result.(SearchMemoriesResponse)
		resultJSON, err := response.ToJSON()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to marshal result: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		}, nil
	}
}

func (s *Server) createDeleteMemoryHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert arguments to JSON for the handler
		jsonData, err := json.Marshal(request.GetArguments())
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to parse arguments: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Call the existing handler
		result, err := s.handler.handleDeleteMemory(ctx, jsonData)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// Convert result to JSON string
		response := result.(DeleteMemoryResponse)
		resultJSON, err := response.ToJSON()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to marshal result: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		}, nil
	}
}

func (s *Server) createMemoryStatsHandler() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		stats, err := s.handler.memoryService.GetMemoryStats(ctx)
		if err != nil {
			return nil, err
		}

		statsJSON, err := json.Marshal(stats)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(statsJSON),
			},
		}, nil
	}
}

func (s *Server) createStoreFactHandler() server.PromptHandlerFunc {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		fact := ""
		category := "personal"

		if f, ok := request.Params.Arguments["fact"]; ok {
			fact = f
		}
		if c, ok := request.Params.Arguments["category"]; ok {
			category = c
		}

		return &mcp.GetPromptResult{
			Messages: []mcp.PromptMessage{
				{
					Role: "user",
					Content: mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Store this fact in the %s category: %s", category, fact),
					},
				},
			},
		}, nil
	}
}