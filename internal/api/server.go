package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ksred/remember-me-mcp/internal/config"
	"github.com/ksred/remember-me-mcp/internal/database"
	"github.com/ksred/remember-me-mcp/internal/services"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	router         *gin.Engine
	config         *config.Config
	db             *database.Database
	memoryService  *services.MemoryService
	authService    *AuthService
	activityService *services.ActivityService
	logger         zerolog.Logger
	httpServer     *http.Server
}

func NewServer(cfg *config.Config, db *database.Database, memoryService *services.MemoryService, activityService *services.ActivityService, logger zerolog.Logger) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware(logger))
	
	// Configure CORS
	corsConfig := cors.DefaultConfig()
	if len(cfg.HTTP.AllowOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.HTTP.AllowOrigins
	} else {
		// Default origins for development
		corsConfig.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:5174", "http://127.0.0.1:3000", "http://127.0.0.1:5173", "http://127.0.0.1:5174"}
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Requested-With"}
	corsConfig.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour
	
	router.Use(cors.New(corsConfig))

	authService := NewAuthService(db, logger)

	server := &Server{
		router:         router,
		config:         cfg,
		db:             db,
		memoryService:  memoryService,
		authService:    authService,
		activityService: activityService,
		logger:         logger,
	}

	server.setupRoutes()

	return server, nil
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthHandler)

	// Swagger documentation
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// Authentication endpoints
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.registerHandler)
			auth.POST("/login", s.loginHandler)
		}

		// Protected endpoints
		protected := v1.Group("")
		protected.Use(s.authMiddleware())
		{
			// API Key management
			keys := protected.Group("/keys")
			{
				keys.GET("", s.listAPIKeysHandler)
				keys.POST("", s.createAPIKeyHandler)
				keys.DELETE("/:id", s.deleteAPIKeyHandler)
			}

			// Memory endpoints (MCP functionality)
			memories := protected.Group("/memories")
			{
				memories.POST("", s.storeMemoryHandler)
				memories.GET("", s.searchMemoriesHandler)
				memories.DELETE("/:id", s.deleteMemoryHandler)
				memories.GET("/stats", s.enhancedMemoryStatsHandler)
			}

			// User activity statistics
			users := protected.Group("/users")
			{
				users.GET("/activity-stats", s.userActivityStatsHandler)
			}

			// System performance statistics
			system := protected.Group("/system")
			{
				system.GET("/performance", s.systemPerformanceStatsHandler)
			}
		}
		
		// MCP protocol endpoint (for Claude Desktop)
		protected.POST("/mcp", s.HandleMCP)
	}
}

func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.logger.Info().Str("address", addr).Msg("Starting HTTP server")
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func LoggerMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info().
			Str("client_ip", clientIP).
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("error", errorMessage).
			Msg("HTTP request")
	}
}

// @title Remember Me MCP API
// @version 1.0
// @description API for Remember Me MCP Server - A persistent memory system for Claude
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8082
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// healthHandler godoc
// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /health [get]
func (s *Server) healthHandler(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Check database health
	dbHealthy := true
	var dbError string
	if err := s.db.Health(ctx); err != nil {
		dbHealthy = false
		dbError = err.Error()
	}

	status := "healthy"
	if !dbHealthy {
		status = "unhealthy"
	}

	response := gin.H{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"database": gin.H{
			"healthy": dbHealthy,
			"error":   dbError,
		},
	}

	if !dbHealthy {
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}