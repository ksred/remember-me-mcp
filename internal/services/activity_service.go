package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ksred/remember-me-mcp/internal/models"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type ActivityService struct {
	db     *gorm.DB
	logger zerolog.Logger
}

func NewActivityService(db *gorm.DB, logger zerolog.Logger) *ActivityService {
	return &ActivityService{
		db:     db,
		logger: logger,
	}
}

// LogActivity logs user activity
func (s *ActivityService) LogActivity(ctx context.Context, userID uint, activityType string, details map[string]interface{}, ipAddress, userAgent string) error {
	activity := &models.ActivityLog{
		UserID:    userID,
		Type:      activityType,
		Details:   details,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(activity).Error; err != nil {
		s.logger.Error().Err(err).Msg("Failed to log activity")
		return err
	}

	return nil
}

// LogPerformance logs performance metrics
func (s *ActivityService) LogPerformance(ctx context.Context, endpoint, method string, responseTime, statusCode int) error {
	metric := &models.PerformanceMetric{
		Endpoint:     endpoint,
		Method:       method,
		ResponseTime: responseTime,
		StatusCode:   statusCode,
		CreatedAt:    time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(metric).Error; err != nil {
		s.logger.Error().Err(err).Msg("Failed to log performance metric")
		return err
	}

	return nil
}

// GetSearchStats returns search statistics for different time periods
func (s *ActivityService) GetSearchStats(ctx context.Context, userID *uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	now := time.Now()

	query := s.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("type = ?", models.ActivityMemorySearch)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// Today
	var todayCount int64
	if err := query.Where("created_at >= ?", now.Truncate(24*time.Hour)).Count(&todayCount).Error; err != nil {
		return nil, err
	}
	stats["searches_today"] = todayCount

	// This week
	var weekCount int64
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	if err := query.Where("created_at >= ?", weekStart.Truncate(24*time.Hour)).Count(&weekCount).Error; err != nil {
		return nil, err
	}
	stats["searches_this_week"] = weekCount

	// This month
	var monthCount int64
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := query.Where("created_at >= ?", monthStart).Count(&monthCount).Error; err != nil {
		return nil, err
	}
	stats["searches_this_month"] = monthCount

	return stats, nil
}

// GetMemoryGrowthStats returns memory growth for the last 7 days
func (s *ActivityService) GetMemoryGrowthStats(ctx context.Context, userID *uint) ([]map[string]interface{}, error) {
	now := time.Now()
	var results []map[string]interface{}

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		
		// Count memories directly from the memories table instead of activity logs
		query := s.db.WithContext(ctx).Model(&models.Memory{}).
			Where("DATE(created_at) = ?", dateStr)

		if userID != nil {
			query = query.Where("user_id = ?", *userID)
		}

		var count int64
		if err := query.Count(&count).Error; err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"date":  dateStr,
			"count": count,
		})
	}

	return results, nil
}

// GetUserActivityStats returns user-specific activity statistics
func (s *ActivityService) GetUserActivityStats(ctx context.Context, userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// API Keys count
	var totalAPIKeys, activeAPIKeys int64
	if err := s.db.WithContext(ctx).Model(&models.APIKey{}).Where("user_id = ?", userID).Count(&totalAPIKeys).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&models.APIKey{}).Where("user_id = ? AND is_active = ?", userID, true).Count(&activeAPIKeys).Error; err != nil {
		return nil, err
	}

	stats["total_api_keys"] = totalAPIKeys
	stats["active_api_keys"] = activeAPIKeys

	// API calls stats
	searchStats, err := s.GetSearchStats(ctx, &userID)
	if err != nil {
		return nil, err
	}
	stats["api_calls_today"] = searchStats["searches_today"]
	stats["api_calls_this_week"] = searchStats["searches_this_week"]

	// User info
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}
	stats["account_created"] = user.CreatedAt

	// Last login - get latest login activity
	var lastLogin models.ActivityLog
	if err := s.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, models.ActivityLogin).
		Order("created_at DESC").First(&lastLogin).Error; err == nil {
		stats["last_login"] = lastLogin.CreatedAt
	}

	// Most used categories
	categories, err := s.getMostUsedCategories(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats["most_used_categories"] = categories

	// Recent activity
	recentActivity, err := s.getRecentActivity(ctx, userID, 10)
	if err != nil {
		return nil, err
	}
	stats["recent_activity"] = recentActivity

	return stats, nil
}

func (s *ActivityService) getMostUsedCategories(ctx context.Context, userID uint) ([]string, error) {
	type CategoryCount struct {
		Category string
		Count    int64
	}

	var results []CategoryCount
	err := s.db.WithContext(ctx).Raw(`
		SELECT 
			details->>'category' as category,
			COUNT(*) as count
		FROM activity_logs 
		WHERE user_id = ? AND type = ? AND details->>'category' IS NOT NULL
		GROUP BY details->>'category'
		ORDER BY count DESC
		LIMIT 3
	`, userID, models.ActivityMemoryStored).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	categories := make([]string, len(results))
	for i, result := range results {
		categories[i] = result.Category
	}

	return categories, nil
}

func (s *ActivityService) getRecentActivity(ctx context.Context, userID uint, limit int) ([]map[string]interface{}, error) {
	var activities []models.ActivityLog
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&activities).Error; err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, len(activities))
	for i, activity := range activities {
		result := map[string]interface{}{
			"type":        activity.Type,
			"timestamp":   activity.CreatedAt,
			"description": s.getActivityDescription(activity),
		}

		// Add type-specific details
		if activity.Details != nil {
			result["details"] = activity.Details
		}

		// Add IP and user agent if available
		if activity.IPAddress != "" {
			result["ip_address"] = activity.IPAddress
		}
		if activity.UserAgent != "" {
			result["user_agent"] = activity.UserAgent
		}

		results[i] = result
	}

	return results, nil
}

// getActivityDescription provides user-friendly descriptions for activities
func (s *ActivityService) getActivityDescription(activity models.ActivityLog) string {
	switch activity.Type {
	case models.ActivityMemoryStored:
		if activity.Details != nil {
			if category, ok := activity.Details["category"].(string); ok {
				return fmt.Sprintf("Stored memory in %s category", category)
			}
		}
		return "Stored a new memory"
	
	case models.ActivityMemorySearch:
		if activity.Details != nil {
			if query, ok := activity.Details["query"].(string); ok {
				if len(query) > 50 {
					query = query[:50] + "..."
				}
				return fmt.Sprintf("Searched for: %s", query)
			}
		}
		return "Performed memory search"
	
	case models.ActivityMemoryDeleted:
		if activity.Details != nil {
			if memoryID, ok := activity.Details["memory_id"]; ok {
				return fmt.Sprintf("Deleted memory (ID: %v)", memoryID)
			}
		}
		return "Deleted a memory"
	
	case models.ActivityAPIKeyCreated:
		if activity.Details != nil {
			if name, ok := activity.Details["name"].(string); ok {
				return fmt.Sprintf("Created API key: %s", name)
			}
		}
		return "Created new API key"
	
	case models.ActivityAPIKeyDeleted:
		if activity.Details != nil {
			if name, ok := activity.Details["name"].(string); ok {
				return fmt.Sprintf("Deleted API key: %s", name)
			}
		}
		return "Deleted API key"
	
	case models.ActivityLogin:
		return "Logged in"
	
	default:
		return fmt.Sprintf("Performed %s action", activity.Type)
	}
}

// GetPerformanceStats returns system performance statistics
func (s *ActivityService) GetPerformanceStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	now := time.Now()

	// Average response time today
	var avgResponseTime float64
	if err := s.db.WithContext(ctx).Model(&models.PerformanceMetric{}).
		Where("created_at >= ?", now.Truncate(24*time.Hour)).
		Select("AVG(response_time)").Scan(&avgResponseTime).Error; err != nil {
		avgResponseTime = 0
	}
	stats["average_response_time_ms"] = int(avgResponseTime)

	// P95 response time today
	var p95ResponseTime int
	if err := s.db.WithContext(ctx).Raw(`
		SELECT PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time) 
		FROM performance_metrics 
		WHERE created_at >= ?
	`, now.Truncate(24*time.Hour)).Scan(&p95ResponseTime).Error; err != nil {
		p95ResponseTime = 0
	}
	stats["p95_response_time_ms"] = p95ResponseTime

	// Total requests today
	var totalRequests int64
	if err := s.db.WithContext(ctx).Model(&models.PerformanceMetric{}).
		Where("created_at >= ?", now.Truncate(24*time.Hour)).
		Count(&totalRequests).Error; err != nil {
		totalRequests = 0
	}
	stats["total_requests_today"] = totalRequests

	// Database connections (simplified)
	var dbConnections int
	sqlDB, err := s.db.DB()
	if err == nil {
		dbStats := sqlDB.Stats()
		dbConnections = dbStats.OpenConnections
	}
	stats["database_connections"] = dbConnections

	// Simplified metrics
	stats["uptime_percentage"] = 99.9
	stats["cache_hit_rate"] = 0.85

	return stats, nil
}