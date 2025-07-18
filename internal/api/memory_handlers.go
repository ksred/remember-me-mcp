package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ksred/remember-me-mcp/internal/mcp"
	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/services"
	"github.com/ksred/remember-me-mcp/internal/utils"
)

// storeMemoryHandler godoc
// @Summary Store a memory
// @Description Store important information that can be recalled later
// @Tags memories
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body mcp.StoreMemoryRequest true "Memory to store"
// @Success 201 {object} mcp.StoreMemoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /memories [post]
func (s *Server) storeMemoryHandler(c *gin.Context) {
	// Get user from context
	user, exists := getUserFromContext(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var req mcp.StoreMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user-scoped memory service
	userMemoryService := s.createScopedMemoryService(user.ID)

	// Store memory using the memory service
	storeReq := &services.StoreMemoryRequest{
		Type:     req.Type,
		Category: req.Category,
		Content:  req.Content,
		Metadata: req.Metadata,
	}
	memory, err := userMemoryService.StoreMemory(c.Request.Context(), storeReq)
	
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to store memory")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store memory"})
		return
	}

	// Log the activity
	details := map[string]interface{}{
		"memory_id": memory.ID,
		"category":  memory.Category,
		"type":      memory.Type,
	}
	go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityMemoryStored, details, c.ClientIP(), c.GetHeader("User-Agent"))

	response := mcp.StoreMemoryResponse{
		Success: true,
		Memory:  memory,
	}

	c.JSON(http.StatusCreated, response)
}

// searchMemoriesHandler godoc
// @Summary Search memories
// @Description Search through stored memories using keywords or semantic search
// @Tags memories
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param query query string true "Search query"
// @Param category query string false "Filter by category (personal, project, business)"
// @Param type query string false "Filter by type (fact, conversation, context, preference)"
// @Param limit query int false "Maximum number of results (default: 100, max: 1000)"
// @Param useSemanticSearch query bool false "Use semantic search (default: true)"
// @Success 200 {object} mcp.SearchMemoriesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /memories [get]
func (s *Server) searchMemoriesHandler(c *gin.Context) {
	// Get user from context
	user, exists := getUserFromContext(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter is required"})
		return
	}

	category := c.Query("category")
	memoryType := c.Query("type")
	
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit > 0 && parsedLimit <= 1000 {
				limit = parsedLimit
			}
		}
	}

	useSemanticSearch := true
	if semanticStr := c.Query("useSemanticSearch"); semanticStr == "false" {
		useSemanticSearch = false
	}

	// Create user-scoped memory service
	userMemoryService := s.createScopedMemoryService(user.ID)

	// Search memories
	searchReq := &services.SearchMemoriesRequest{
		Query:             query,
		Category:          category,
		Type:              memoryType,
		Limit:             limit,
		UseSemanticSearch: useSemanticSearch,
	}
	memories, err := userMemoryService.SearchMemories(c.Request.Context(), searchReq)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to search memories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search memories"})
		return
	}

	// Log the search activity
	details := map[string]interface{}{
		"query":                query,
		"category":             category,
		"type":                 memoryType,
		"limit":                limit,
		"use_semantic_search":  useSemanticSearch,
		"results_count":        len(memories),
	}
	go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityMemorySearch, details, c.ClientIP(), c.GetHeader("User-Agent"))

	response := mcp.SearchMemoriesResponse{
		Memories: memories,
		Count:    len(memories),
	}

	c.JSON(http.StatusOK, response)
}

// deleteMemoryHandler godoc
// @Summary Delete a memory
// @Description Delete a memory by its ID
// @Tags memories
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Memory ID"
// @Success 200 {object} mcp.DeleteMemoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /memories/{id} [delete]
func (s *Server) deleteMemoryHandler(c *gin.Context) {
	// Get user from context
	user, exists := getUserFromContext(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memory ID"})
		return
	}

	// Create user-scoped memory service
	userMemoryService := s.createScopedMemoryService(user.ID)

	delReq := &services.DeleteMemoryRequest{
		ID: uint(id),
	}
	err = userMemoryService.DeleteMemory(c.Request.Context(), delReq)
	if err != nil {
		// Check if it's a NotFoundError
		var notFoundErr *utils.NotFoundError
		if errors.As(err, &notFoundErr) {
			c.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
			return
		}
		s.logger.Error().Err(err).Msg("Failed to delete memory")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete memory"})
		return
	}

	// Log the deletion activity
	details := map[string]interface{}{
		"memory_id": uint(id),
	}
	go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityMemoryDeleted, details, c.ClientIP(), c.GetHeader("User-Agent"))

	response := mcp.DeleteMemoryResponse{
		Success: true,
		Message: "Memory deleted successfully",
	}

	c.JSON(http.StatusOK, response)
}

// basicMemoryStatsHandler - deprecated, kept for compatibility
func (s *Server) basicMemoryStatsHandler(c *gin.Context) {
	stats, err := s.memoryService.GetMemoryStats(c.Request.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get memory stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get memory statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// enhancedMemoryStatsHandler godoc
// @Summary Get enhanced memory statistics
// @Description Get comprehensive statistics about stored memories including search patterns and growth trends
// @Tags memories
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /memories/stats [get]
func (s *Server) enhancedMemoryStatsHandler(c *gin.Context) {
	// Get user from context
	user, exists := getUserFromContext(c)
	if !exists || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	ctx := c.Request.Context()
	
	// Create user-scoped memory service
	userMemoryService := s.createScopedMemoryService(user.ID)

	// Get basic memory stats
	basicStats, err := userMemoryService.GetMemoryStats(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get basic memory stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get memory statistics"})
		return
	}
	
	// Get search statistics for this user
	userIDPtr := &user.ID
	searchStats, err := s.activityService.GetSearchStats(ctx, userIDPtr)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get search stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get search statistics"})
		return
	}
	
	// Get memory growth stats for the last 7 days
	growthStats, err := s.activityService.GetMemoryGrowthStats(ctx, userIDPtr)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get memory growth stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get memory growth statistics"})
		return
	}
	
	// Combine all statistics
	enhancedStats := map[string]interface{}{
		"basic_stats":    basicStats,
		"search_stats":   searchStats,
		"growth_stats":   growthStats,
	}
	
	c.JSON(http.StatusOK, enhancedStats)
}

// userActivityStatsHandler godoc
// @Summary Get user activity statistics
// @Description Get comprehensive activity statistics for the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/activity-stats [get]
func (s *Server) userActivityStatsHandler(c *gin.Context) {
	// Get user from context (set by auth middleware)
	user, exists := getUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	stats, err := s.activityService.GetUserActivityStats(c.Request.Context(), user.ID)
	if err != nil {
		s.logger.Error().Err(err).Uint("user_id", user.ID).Msg("Failed to get user activity stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user activity statistics"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}

// systemPerformanceStatsHandler godoc
// @Summary Get system performance statistics
// @Description Get system-wide performance metrics and health indicators
// @Tags system
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /system/performance [get]
func (s *Server) systemPerformanceStatsHandler(c *gin.Context) {
	stats, err := s.activityService.GetPerformanceStats(c.Request.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get system performance stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system performance statistics"})
		return
	}
	
	c.JSON(http.StatusOK, stats)
}