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
	db         *gorm.DB
	embedding  EmbeddingService
	encryption *utils.EncryptionService
	logger     zerolog.Logger
	config     map[string]interface{}
	userID     uint // User ID for scoping memories (0 means no scoping)
}

// NewMemoryService creates a new instance of MemoryService for local MCP mode
// This uses the system user (ID: 1) for all operations
func NewMemoryService(db *gorm.DB, embedding EmbeddingService, logger zerolog.Logger, config map[string]interface{}) *MemoryService {
	if config == nil {
		config = make(map[string]interface{})
	}
	
	// Extract encryption service from config if available
	var encryption *utils.EncryptionService
	if encSvc, ok := config["encryption_service"].(*utils.EncryptionService); ok {
		encryption = encSvc
	}
	
	return &MemoryService{
		db:         db,
		embedding:  embedding,
		encryption: encryption,
		logger:     logger,
		config:     config,
		userID:     1, // System user for local MCP mode
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
	
	// Extract encryption service from config if available
	var encryption *utils.EncryptionService
	if encSvc, ok := config["encryption_service"].(*utils.EncryptionService); ok {
		encryption = encSvc
	}
	
	return &MemoryService{
		db:         db,
		embedding:  embedding,
		encryption: encryption,
		logger:     logger,
		config:     config,
		userID:     userID,
	}
}

// StoreRequest represents a request to store a memory
type StoreRequest struct {
	Content  string
	Category string
	Type     string
	Priority string
	UpdateKey string
	Tags     []string
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

// UpdateRequest represents a request to update a memory
type UpdateRequest struct {
	Content  string
	Category string
	Type     string
	Priority string
	Tags     []string
	Metadata map[string]interface{}
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
			
		// Store original content for embedding generation
		originalContent := req.Content
		
		existing.Content = req.Content
		existing.Category = req.Category
		existing.Type = req.Type
		existing.Priority = req.Priority
		existing.UpdateKey = req.UpdateKey
		existing.Tags = req.Tags
		
		if req.Metadata != nil {
			metadataJSON, err := json.Marshal(req.Metadata)
			if err != nil {
				return nil, utils.WrapValidationError("metadata", "invalid metadata format")
			}
			existing.Metadata = json.RawMessage(metadataJSON)
		}
		
		// Encrypt content if encryption is enabled
		if err := s.encryptContent(existing); err != nil {
			s.logger.Error().Err(err).Msg("failed to encrypt content")
			return nil, utils.WrapDatabaseError("encrypt content", err)
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
		// Use original content for embedding, not encrypted content
		if s.embedding != nil {
			go s.generateEmbeddingAsync(existing.ID, originalContent)
		}
		
		// Decrypt content before returning if it was encrypted
		if err := s.decryptContent(existing); err != nil {
			s.logger.Warn().Err(err).Msg("failed to decrypt content for response")
			// Don't fail the operation, just return with encrypted marker
		}
		
		return existing, nil
	}

	// Store original content for embedding generation
	originalContent := req.Content
	
	// Create new memory
	memory := &models.Memory{
		UserID:    s.userID,
		Content:   req.Content,
		Category:  req.Category,
		Type:      req.Type,
		Priority:  req.Priority,
		UpdateKey: req.UpdateKey,
		Tags:      req.Tags,
	}
	
	s.logger.Debug().Msg("Creating new memory - will generate embedding asynchronously")
	
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, utils.WrapValidationError("metadata", "invalid metadata format")
		}
		memory.Metadata = json.RawMessage(metadataJSON)
	}
	
	// Encrypt content if encryption is enabled
	if err := s.encryptContent(memory); err != nil {
		s.logger.Error().Err(err).Msg("failed to encrypt content")
		return nil, utils.WrapDatabaseError("encrypt content", err)
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
	// Use original content for embedding, not encrypted content
	if s.embedding != nil {
		go s.generateEmbeddingAsync(memory.ID, originalContent)
	}
	
	// Decrypt content before returning if it was encrypted
	if err := s.decryptContent(memory); err != nil {
		s.logger.Warn().Err(err).Msg("failed to decrypt content for response")
		// Don't fail the operation, just return with encrypted marker
	}

	return memory, nil
}

// Update updates an existing memory by ID
func (s *MemoryService) Update(ctx context.Context, id uint, req UpdateRequest) (*models.Memory, error) {
	// Create a new context with a longer timeout to avoid cancellation
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find the memory by ID
	var memory models.Memory
	if err := s.db.WithContext(dbCtx).Where("id = ? AND user_id = ?", id, s.userID).First(&memory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, utils.WrapNotFoundError("memory", fmt.Sprintf("%d", id))
		}
		return nil, utils.WrapDatabaseError("find memory", err)
	}

	// Store original content for embedding generation
	originalContent := memory.Content

	// Update fields if provided (only update non-empty values)
	if req.Content != "" {
		memory.Content = req.Content
		originalContent = req.Content // Use new content for embedding
	}
	if req.Category != "" {
		memory.Category = req.Category
	}
	if req.Type != "" {
		memory.Type = req.Type
	}
	if req.Priority != "" {
		memory.Priority = req.Priority
	}
	if req.Tags != nil {
		memory.Tags = req.Tags
	}

	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, utils.WrapValidationError("metadata", "invalid metadata format")
		}
		memory.Metadata = json.RawMessage(metadataJSON)
	}

	// Encrypt content if encryption is enabled
	if err := s.encryptContent(&memory); err != nil {
		s.logger.Error().Err(err).Msg("failed to encrypt content")
		return nil, utils.WrapDatabaseError("encrypt content", err)
	}

	// Update memory without touching embedding field initially
	updateErr := s.db.WithContext(dbCtx).Omit("embedding").Save(&memory).Error
	if updateErr != nil {
		s.logger.Error().Err(updateErr).Msg("failed to update memory")
		return nil, utils.WrapDatabaseError("update memory", updateErr)
	}

	// Generate new embedding asynchronously if content changed
	if req.Content != "" && s.embedding != nil {
		go s.generateEmbeddingAsync(memory.ID, originalContent)
	}

	s.logger.Info().
		Uint("id", memory.ID).
		Msg("successfully updated memory")

	// Decrypt content before returning if it was encrypted
	if err := s.decryptContent(&memory); err != nil {
		s.logger.Warn().Err(err).Msg("failed to decrypt content for response")
		// Don't fail the operation, just return with encrypted marker
	}

	return &memory, nil
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
	// Handle wildcard query - return all memories
	if req.Query == "*" || req.Query == "" {
		req.Query = ""
		req.UseSemanticSearch = false
	}
	
	// Use semantic search if requested and embedding service is available
	if req.UseSemanticSearch && s.embedding != nil && req.Query != "" {
		return s.SearchSemantic(ctx, req)
	}

	// Fall back to keyword search
	query := s.db.WithContext(ctx).Model(&models.Memory{}).Where("user_id = ?", s.userID)

	// Apply keyword search if query is provided (and not wildcard)
	if req.Query != "" && req.Query != "*" {
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

	// Order by created_at descending (newest first)
	query = query.Order("created_at DESC")

	var memories []*models.Memory
	if err := query.Omit("embedding", "tags").Find(&memories).Error; err != nil {
		s.logger.Error().Err(err).Msg("failed to search memories")
		return nil, utils.WrapDatabaseError("search memories", err)
	}
	
	// Decrypt content for each memory
	for _, memory := range memories {
		if err := s.decryptContent(memory); err != nil {
			s.logger.Warn().Err(err).Uint("id", memory.ID).Msg("failed to decrypt memory content")
			// Continue with other memories, don't fail the entire search
		}
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
	var memories []*models.Memory
	
	// For SQLite in tests, fall back to regular search
	if s.db.Dialector.Name() == "sqlite" {
		req.UseSemanticSearch = false
		return s.Search(ctx, req)
	}

	// Get similarity threshold from config - use a lower default for now
	similarityThreshold := 0.3 // lowered significantly from 0.7
	if threshold, ok := s.config["similarity_threshold"].(float64); ok && threshold > 0 {
		similarityThreshold = threshold
	}
	
	s.logger.Info().
		Float64("similarity_threshold", similarityThreshold).
		Str("query", req.Query).
		Int("limit", limit).
		Msg("Performing semantic search")

	// First, check if we have any memories with embeddings
	var totalCount int64
	s.db.WithContext(ctx).Model(&models.Memory{}).
		Where("user_id = ? AND embedding IS NOT NULL", s.userID).
		Count(&totalCount)
	
	s.logger.Info().
		Int64("memories_with_embeddings", totalCount).
		Msg("Total memories available for semantic search")

	if totalCount == 0 {
		s.logger.Warn().Msg("No memories with embeddings found")
		return []*models.Memory{}, nil
	}

	// Simple semantic search query using pgvector
	// Calculate similarity and order by it
	// Using raw SQL for the order clause to ensure proper syntax
	sql := fmt.Sprintf(`
		SELECT *, (1 - (embedding <=> $1)) as similarity 
		FROM memories 
		WHERE user_id = $2 AND embedding IS NOT NULL
		%s %s
		ORDER BY embedding <=> $1
		LIMIT $3
	`, 
		func() string {
			if req.Category != "" {
				return "AND category = $4"
			}
			return ""
		}(),
		func() string {
			if req.Type != "" {
				if req.Category != "" {
					return "AND type = $5"
				}
				return "AND type = $4"
			}
			return ""
		}(),
	)
	
	args := []interface{}{pgvector.NewVector(queryEmbedding), s.userID, limit}
	if req.Category != "" {
		args = append(args, req.Category)
	}
	if req.Type != "" {
		args = append(args, req.Type)
	}
	
	err = s.db.WithContext(ctx).Raw(sql, args...).Scan(&memories).Error

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("query", req.Query).
			Msg("failed to perform semantic search")
		return nil, utils.WrapDatabaseError("semantic search", err)
	}
	
	s.logger.Info().
		Int("results_count", len(memories)).
		Msg("Semantic search completed")
	
	// Decrypt content for each memory
	for _, memory := range memories {
		if err := s.decryptContent(memory); err != nil {
			s.logger.Warn().Err(err).Uint("id", memory.ID).Msg("failed to decrypt memory content")
			// Continue with other memories, don't fail the entire search
		}
	}

	return memories, nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
	
	// Decrypt content if encrypted
	if err := s.decryptContent(&memory); err != nil {
		s.logger.Warn().Err(err).Uint("id", memory.ID).Msg("failed to decrypt memory content")
		// Don't fail the operation, return with encrypted marker
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

// GetEncryptionService returns the encryption service
func (s *MemoryService) GetEncryptionService() *utils.EncryptionService {
	return s.encryption
}

// encryptContent encrypts the content field if encryption is enabled
func (s *MemoryService) encryptContent(memory *models.Memory) error {
	if s.encryption == nil || memory.Content == "" {
		return nil
	}
	
	// Encrypt the content
	encryptedData, err := s.encryption.EncryptField(memory.Content)
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}
	
	// Store encrypted data as JSON
	encryptedJSON, err := json.Marshal(encryptedData)
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted data: %w", err)
	}
	
	memory.EncryptedContent = encryptedJSON
	memory.IsEncrypted = true
	// Clear the plain text content
	memory.Content = "[encrypted]"
	
	return nil
}

// decryptContent decrypts the content field if it's encrypted
func (s *MemoryService) decryptContent(memory *models.Memory) error {
	if !memory.IsEncrypted || len(memory.EncryptedContent) == 0 {
		return nil
	}
	
	if s.encryption == nil {
		return fmt.Errorf("content is encrypted but encryption service is not available")
	}
	
	// Unmarshal encrypted data
	var encryptedData utils.EncryptedData
	if err := json.Unmarshal(memory.EncryptedContent, &encryptedData); err != nil {
		return fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}
	
	// Decrypt the content
	decrypted, err := s.encryption.DecryptField(&encryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt content: %w", err)
	}
	
	memory.Content = decrypted
	
	return nil
}