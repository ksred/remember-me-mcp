package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/ksred/remember-me-mcp/internal/utils"
)

// MemoryService handles memory-related business logic
type MemoryService struct {
	db        *gorm.DB
	embedding EmbeddingService
	logger    zerolog.Logger
	config    map[string]interface{}
	userID    uint // User ID for scoping memories (0 means no scoping)
}

// NewMemoryService creates a new instance of MemoryService for local MCP mode
// This uses the system user (ID: 1) for all operations
func NewMemoryService(db *gorm.DB, embedding EmbeddingService, logger zerolog.Logger, config map[string]interface{}) *MemoryService {
	if config == nil {
		config = make(map[string]interface{})
	}
	return &MemoryService{
		db:        db,
		embedding: embedding,
		logger:    logger,
		config:    config,
		userID:    1, // System user for local MCP mode
	}
}

// NewMemoryServiceWithUser creates a new instance of MemoryService for HTTP mode
// This scopes all operations to the specified user
func NewMemoryServiceWithUser(db *gorm.DB, embedding EmbeddingService, logger zerolog.Logger, config map[string]interface{}, userID uint) *MemoryService {
	if config == nil {
		config = make(map[string]interface{})
	}
	if userID == 0 {
		panic("userID cannot be 0 for HTTP mode")
	}
	if userID == 1 {
		panic("system user (ID: 1) cannot be used in HTTP mode")
	}
	return &MemoryService{
		db:        db,
		embedding: embedding,
		logger:    logger,
		config:    config,
		userID:    userID,
	}
}

// StoreRequest represents a request to store a memory
type StoreRequest struct {
	Content  string
	Category string
	Type     string
	Priority string
	UpdateKey string
	Metadata map[string]interface{}
}

// SearchRequest represents a request to search memories
type SearchRequest struct {
	Query             string
	Category          string
	Type              string
	Limit             int
	UseSemanticSearch bool
}

// ProcessContentForMemory automatically detects and stores memories from content
func (s *MemoryService) ProcessContentForMemory(ctx context.Context, content string) ([]*models.Memory, error) {
	// Detect memory patterns
	detectedMemories := DetectMemoryPatterns(content)
	
	var storedMemories []*models.Memory
	
	for _, detected := range detectedMemories {
		// Skip if confidence is too low
		if detected.Confidence < 0.5 {
			continue
		}
		
		req := StoreRequest{
			Content:   detected.Content,
			Category:  detected.Category,
			Type:      detected.Type,
			Priority:  detected.Priority.String(),
			UpdateKey: detected.UpdateKey,
			Metadata:  map[string]interface{}{
				"auto_detected": true,
				"confidence":    detected.Confidence,
				"pattern_type":  detected.Type,
			},
		}
		
		memory, err := s.Store(ctx, req)
		if err != nil {
			s.logger.Warn().Err(err).Str("content", detected.Content).Msg("failed to store auto-detected memory")
			continue
		}
		
		storedMemories = append(storedMemories, memory)
	}
	
	return storedMemories, nil
}

// Store creates or updates a memory
func (s *MemoryService) Store(ctx context.Context, req StoreRequest) (*models.Memory, error) {
	// Validate input
	if req.Content == "" {
		return nil, utils.WrapValidationError("", "content cannot be empty")
	}

	var existing *models.Memory
	var err error

	// Check for existing memory using UpdateKey first (for intelligent updates)
	if req.UpdateKey != "" {
		existing, err = s.findByUpdateKey(ctx, req.UpdateKey)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error().Err(err).Msg("failed to check for existing memory by update key")
			return nil, utils.WrapDatabaseError("check for existing memory", err)
		}
	}

	// If no UpdateKey match, check for duplicate content
	if existing == nil {
		existing, err = s.findByContent(ctx, req.Content)
		if err != nil && err != gorm.ErrRecordNotFound {
			s.logger.Error().Err(err).Msg("failed to check for duplicate memory")
			return nil, utils.WrapDatabaseError("check for duplicate memory", err)
		}
	}

	// If memory exists, update it
	if existing != nil {
		s.logger.Info().
			Uint("id", existing.ID).
			Str("update_key", req.UpdateKey).
			Msg("updating existing memory")
			
		existing.Content = req.Content
		existing.Category = req.Category
		existing.Type = req.Type
		existing.Priority = req.Priority
		existing.UpdateKey = req.UpdateKey
		
		if req.Metadata != nil {
			metadataJSON, err := json.Marshal(req.Metadata)
			if err != nil {
				return nil, utils.WrapValidationError("metadata", "invalid metadata format")
			}
			existing.Metadata = json.RawMessage(metadataJSON)
		}
		
		// Skip embedding generation for updates too - do it asynchronously
		// This prevents MCP timeout issues from affecting memory updates
		
		// Create a new context with a longer timeout to avoid cancellation
		dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Update memory without touching embedding field
		updateErr := s.db.WithContext(dbCtx).Omit("embedding").Save(existing).Error
		
		if updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("failed to update memory")
			return nil, utils.WrapDatabaseError("update memory", updateErr)
		}
		
		// Generate embedding asynchronously after updating the memory
		if s.embedding != nil {
			go s.generateEmbeddingAsync(existing.ID, req.Content)
		}
		
		return existing, nil
	}

	// Create new memory
	memory := &models.Memory{
		UserID:    s.userID,
		Content:   req.Content,
		Category:  req.Category,
		Type:      req.Type,
		Priority:  req.Priority,
		UpdateKey: req.UpdateKey,
	}
	
	s.logger.Debug().Msg("Creating new memory - will generate embedding asynchronously")
	
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, utils.WrapValidationError("metadata", "invalid metadata format")
		}
		memory.Metadata = json.RawMessage(metadataJSON)
	}

	// Skip embedding generation for now - we'll do it asynchronously after storing
	// This prevents MCP timeout issues from affecting memory storage

	// Create the memory record
	// Create a new context with a longer timeout to avoid cancellation
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create memory without embedding first
	createErr := s.db.WithContext(dbCtx).Omit("embedding").Create(memory).Error
	
	if createErr != nil {
		s.logger.Error().Err(createErr).Msg("failed to create memory")
		return nil, utils.WrapDatabaseError("create memory", createErr)
	}

	// Enforce memory limit if configured
	if err := s.enforceMemoryLimit(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("failed to enforce memory limit")
		// Don't fail the operation, just log the warning
	}

	s.logger.Info().
		Uint("id", memory.ID).
		Str("type", memory.Type).
		Str("category", memory.Category).
		Str("priority", memory.Priority).
		Str("update_key", memory.UpdateKey).
		Msg("successfully stored new memory")

	// Generate embedding asynchronously after storing the memory
	if s.embedding != nil {
		go s.generateEmbeddingAsync(memory.ID, req.Content)
	}

	return memory, nil
}

// generateEmbeddingAsync generates embedding for a memory asynchronously
func (s *MemoryService) generateEmbeddingAsync(memoryID uint, content string) {
	s.logger.Debug().Uint("memory_id", memoryID).Msg("starting async embedding generation")
	
	// Use the same approach as the successful startup validation
	// Don't pass any context from the caller - create completely fresh one
	embedding, err := s.embedding.GenerateEmbedding(context.Background(), content)
	if err != nil {
		s.logger.Warn().Err(err).Uint("memory_id", memoryID).Msg("failed to generate embedding asynchronously")
		return
	}
	
	// Update the memory with the embedding
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer updateCancel()
	
	err = s.db.WithContext(updateCtx).
		Model(&models.Memory{}).
		Where("id = ?", memoryID).
		UpdateColumn("embedding", pgvector.NewVector(embedding)).Error
	
	if err != nil {
		s.logger.Error().Err(err).Uint("memory_id", memoryID).Msg("failed to update memory with embedding")
		return
	}
	
	s.logger.Info().Uint("memory_id", memoryID).Int("dimensions", len(embedding)).Msg("successfully updated memory with embedding")
}

// Search searches memories based on the provided criteria
func (s *MemoryService) Search(ctx context.Context, req SearchRequest) ([]*models.Memory, error) {
	// Use semantic search if requested and embedding service is available
	if req.UseSemanticSearch && s.embedding != nil && req.Query != "" {
		return s.SearchSemantic(ctx, req)
	}

	// Fall back to keyword search
	query := s.db.WithContext(ctx).Model(&models.Memory{}).Where("user_id = ?", s.userID)

	// Apply keyword search if query is provided
	if req.Query != "" {
		searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(req.Query))
		query = query.Where("LOWER(content) LIKE ?", searchTerm)
	}

	// Filter by category if provided
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	// Filter by type if provided
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	// Apply limit
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	} else {
		// Default limit to prevent returning too many results
		query = query.Limit(100)
	}

	// Order by priority (high to low) then by created_at descending (newest first)
	query = query.Order("CASE WHEN priority = 'critical' THEN 1 WHEN priority = 'high' THEN 2 WHEN priority = 'medium' THEN 3 WHEN priority = 'low' THEN 4 ELSE 3 END, created_at DESC")

	var memories []*models.Memory
	if err := query.Omit("embedding", "tags").Find(&memories).Error; err != nil {
		s.logger.Error().Err(err).Msg("failed to search memories")
		return nil, utils.WrapDatabaseError("search memories", err)
	}

	return memories, nil
}

// SearchSemantic performs semantic search using vector embeddings
func (s *MemoryService) SearchSemantic(ctx context.Context, req SearchRequest) ([]*models.Memory, error) {
	if s.embedding == nil {
		return nil, fmt.Errorf("embedding service not available")
	}

	// Generate embedding for the search query
	queryEmbedding, err := s.embedding.GenerateEmbedding(ctx, req.Query)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate query embedding")
		// Fall back to keyword search
		req.UseSemanticSearch = false
		return s.Search(ctx, req)
	}

	// Build the query
	query := s.db.WithContext(ctx).Model(&models.Memory{}).Where("user_id = ?", s.userID)

	// Apply category filter if provided
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	// Apply type filter if provided
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	// Apply limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	// Perform vector similarity search
	// Using cosine similarity (1 - cosine_distance)
	var memories []*models.Memory
	
	// For SQLite in tests, fall back to regular search
	if s.db.Dialector.Name() == "sqlite" {
		req.UseSemanticSearch = false
		return s.Search(ctx, req)
	}

	// PostgreSQL with pgvector
	err = query.
		Select("*, (1 - (embedding <=> ?)) as similarity", pgvector.NewVector(queryEmbedding)).
		Where("embedding IS NOT NULL").
		Order("similarity DESC, created_at DESC").
		Limit(limit).
		Find(&memories).Error

	if err != nil {
		s.logger.Error().Err(err).Msg("failed to perform semantic search")
		return nil, utils.WrapDatabaseError("semantic search", err)
	}

	return memories, nil
}

// Delete deletes a memory by ID
func (s *MemoryService) Delete(ctx context.Context, id uint) error {
	// Check if memory exists and belongs to the user
	var memory models.Memory
	query := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, s.userID)
	
	// For SQLite, omit fields that cause issues
	if s.db.Dialector.Name() == "sqlite" {
		query = query.Omit("embedding", "tags")
	}
	
	if err := query.First(&memory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return utils.WrapNotFoundError("memory", fmt.Sprintf("%d", id))
		}
		s.logger.Error().Err(err).Msg("failed to find memory")
		return utils.WrapDatabaseError("find memory", err)
	}

	// Delete the memory
	if err := s.db.WithContext(ctx).Delete(&memory).Error; err != nil {
		s.logger.Error().Err(err).Msg("failed to delete memory")
		return utils.WrapDatabaseError("delete memory", err)
	}

	return nil
}

// Count returns the total number of memories for the user
func (s *MemoryService) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Memory{}).Where("user_id = ?", s.userID).Count(&count).Error; err != nil {
		s.logger.Error().Err(err).Msg("failed to count memories")
		return 0, utils.WrapDatabaseError("count memories", err)
	}

	return count, nil
}

// GetByID retrieves a memory by its ID for the user
func (s *MemoryService) GetByID(ctx context.Context, id uint) (*models.Memory, error) {
	var memory models.Memory
	query := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, s.userID)
	
	// For SQLite, omit fields that cause issues
	if s.db.Dialector.Name() == "sqlite" {
		query = query.Omit("embedding", "tags")
	}
	
	if err := query.First(&memory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.WrapNotFoundError("memory", fmt.Sprintf("%d", id))
		}
		s.logger.Error().Err(err).Msg("failed to get memory by id")
		return nil, utils.WrapDatabaseError("get memory by id", err)
	}

	return &memory, nil
}

// findByContent finds a memory with the exact same content for the user
func (s *MemoryService) findByContent(ctx context.Context, content string) (*models.Memory, error) {
	var memory models.Memory
	// Create a new context with a longer timeout to avoid cancellation
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	query := s.db.WithContext(dbCtx).Where("content = ? AND user_id = ?", content, s.userID)
	
	// For SQLite, omit fields that cause issues
	if s.db.Dialector.Name() == "sqlite" {
		query = query.Omit("embedding", "tags")
	}
	
	err := query.First(&memory).Error
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// findByUpdateKey finds a memory with the same update key (for intelligent updates) for the user
func (s *MemoryService) findByUpdateKey(ctx context.Context, updateKey string) (*models.Memory, error) {
	var memory models.Memory
	// Create a new context with a longer timeout to avoid cancellation
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	query := s.db.WithContext(dbCtx).Where("update_key = ? AND user_id = ?", updateKey, s.userID)
	
	// For SQLite, omit fields that cause issues
	if s.db.Dialector.Name() == "sqlite" {
		query = query.Omit("embedding", "tags")
	}
	
	err := query.First(&memory).Error
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

// enforceMemoryLimit deletes oldest memories if over the configured limit
func (s *MemoryService) enforceMemoryLimit(ctx context.Context) error {
	// Get memory limit from config
	limitInterface, exists := s.config["memory_limit"]
	if !exists {
		// No limit configured
		return nil
	}

	limit, ok := limitInterface.(int)
	if !ok {
		// Try to convert from float64 (common in JSON)
		if limitFloat, ok := limitInterface.(float64); ok {
			limit = int(limitFloat)
		} else {
			s.logger.Warn().Interface("memory_limit", limitInterface).Msg("invalid memory_limit configuration")
			return nil
		}
	}

	if limit <= 0 {
		// No limit or invalid limit
		return nil
	}

	// Count current memories
	count, err := s.Count(ctx)
	if err != nil {
		return err
	}

	if count <= int64(limit) {
		// Within limit
		return nil
	}

	// Calculate how many to delete
	toDelete := int(count) - limit

	// Find and delete oldest memories
	var oldestMemories []models.Memory
	query := s.db.WithContext(ctx).Order("created_at ASC").Limit(toDelete)
	
	// For SQLite, omit fields that cause issues
	if s.db.Dialector.Name() == "sqlite" {
		query = query.Omit("embedding", "tags")
	}
	
	if err := query.Find(&oldestMemories).Error; err != nil {
		return fmt.Errorf("failed to find oldest memories: %w", err)
	}

	// Delete the oldest memories
	for _, memory := range oldestMemories {
		if err := s.db.WithContext(ctx).Delete(&memory).Error; err != nil {
			s.logger.Error().Err(err).Uint("id", memory.ID).Msg("failed to delete old memory")
			// Continue deleting others
		}
	}

	s.logger.Info().
		Int("deleted", toDelete).
		Int("limit", limit).
		Msg("enforced memory limit")

	return nil
}

// StoreMemory stores a memory using the standard request/response types
func (s *MemoryService) StoreMemory(ctx context.Context, req *StoreMemoryRequest) (*models.Memory, error) {
	storeReq := StoreRequest{
		Content:  req.Content,
		Category: req.Category,
		Type:     req.Type,
		Metadata: req.Metadata,
	}
	
	memory, err := s.Store(ctx, storeReq)
	if err != nil {
		return nil, err
	}
	
	// Set tags if provided
	if len(req.Tags) > 0 {
		memory.Tags = req.Tags
		if err := s.db.WithContext(ctx).Save(memory).Error; err != nil {
			s.logger.Error().Err(err).Msg("failed to save memory tags")
			return nil, utils.WrapDatabaseError("save memory tags", err)
		}
	}
	
	return memory, nil
}

// SearchMemories searches memories using the standard request/response types
func (s *MemoryService) SearchMemories(ctx context.Context, req *SearchMemoriesRequest) ([]*models.Memory, error) {
	searchReq := SearchRequest{
		Query:             req.Query,
		Category:          req.Category,
		Type:              req.Type,
		Limit:             req.Limit,
		UseSemanticSearch: req.UseSemanticSearch,
	}
	
	return s.Search(ctx, searchReq)
}

// DeleteMemory deletes a memory using the standard request/response types
func (s *MemoryService) DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) error {
	return s.Delete(ctx, req.ID)
}

// GetMemoryStats returns statistics about stored memories
func (s *MemoryService) GetMemoryStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get total count
	totalCount, err := s.Count(ctx)
	if err != nil {
		return nil, err
	}
	stats["total_count"] = totalCount
	
	// Get count by category
	categoryStats := make(map[string]int64)
	for _, category := range []string{models.CategoryPersonal, models.CategoryProject, models.CategoryBusiness} {
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Memory{}).Where("category = ? AND user_id = ?", category, s.userID).Count(&count).Error; err != nil {
			s.logger.Error().Err(err).Str("category", category).Msg("failed to count memories by category")
			continue
		}
		categoryStats[category] = count
	}
	stats["by_category"] = categoryStats
	
	// Get count by type
	typeStats := make(map[string]int64)
	for _, memType := range []string{models.TypeFact, models.TypeConversation, models.TypeContext, models.TypePreference} {
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Memory{}).Where("type = ? AND user_id = ?", memType, s.userID).Count(&count).Error; err != nil {
			s.logger.Error().Err(err).Str("type", memType).Msg("failed to count memories by type")
			continue
		}
		typeStats[memType] = count
	}
	stats["by_type"] = typeStats
	
	// Get embedding stats
	var embeddingCount int64
	if err := s.db.WithContext(ctx).Model(&models.Memory{}).Where("embedding IS NOT NULL AND user_id = ?", s.userID).Count(&embeddingCount).Error; err != nil {
		s.logger.Error().Err(err).Msg("failed to count memories with embeddings")
	} else {
		stats["with_embeddings"] = embeddingCount
		stats["without_embeddings"] = totalCount - embeddingCount
	}
	
	return stats, nil
}

// GetEmbeddingService returns the embedding service
func (s *MemoryService) GetEmbeddingService() EmbeddingService {
	return s.embedding
}