# Onboarding: Search Events and Response Time Tracking

## Task Overview

The user reported two main issues:
1. **Search events are not saved** - They don't show in activity or on "searches today/searches this week"
2. **Response time is not saved** - Questions about whether we save this and how to do it efficiently

## Project Overview

**Remember Me MCP** is a Model Context Protocol (MCP) server that provides persistent memory capabilities for Claude Desktop. It includes:
- Memory storage and retrieval with semantic search
- Activity tracking for user actions
- HTTP API server with authentication
- PostgreSQL database with pgvector for embeddings

## Current Implementation Analysis

### 1. Search Event Tracking

**Status: Partially Implemented**

Search events ARE being saved to the database, but there might be issues with how they're displayed or retrieved.

#### How Search Events are Saved:
- **Location**: `/internal/api/memory_handlers.go:151`
- **Method**: When a search is performed, the handler logs an activity asynchronously:
```go
go s.activityService.LogActivity(c.Request.Context(), user.ID, models.ActivityMemorySearch, details, c.ClientIP(), c.GetHeader("User-Agent"))
```
- **Data Saved**: query, category, type, limit, use_semantic_search, results_count
- **Table**: `activity_logs`

#### Search Statistics Implementation:
- **Location**: `/internal/services/activity_service.go:63-98` (`GetSearchStats` method)
- **Endpoints**: 
  - `/api/v1/memories/stats` - Returns memory stats including search counts
  - `/api/v1/users/activity-stats` - Returns user-specific activity stats
- **Functionality**: Counts searches for today, this week, and this month

### 2. Response Time Tracking

**Status: Not Implemented (Infrastructure exists but not connected)**

#### Current State:
1. **Database Schema Issue**:
   - Database table uses column `duration_ms`
   - Go model uses field `ResponseTime` (maps to `response_time`)
   - This mismatch causes SQL errors when querying performance metrics

2. **Missing Middleware**:
   - `LogPerformance` method exists in `ActivityService` but is never called
   - Current `LoggerMiddleware` only logs to console, doesn't persist to database
   - No middleware tracks and saves response times

3. **Existing Infrastructure**:
   - **Model**: `PerformanceMetric` in `/internal/models/activity.go`
   - **Service Method**: `LogPerformance` in `/internal/services/activity_service.go`
   - **Table**: `performance_metrics` with columns: endpoint, method, duration_ms, status_code, user_id, error, created_at
   - **Endpoint**: `/api/v1/system/performance` (currently broken due to schema mismatch)

## Issues to Fix

### 1. Search Events Not Showing
Possible causes:
- The asynchronous logging might be failing silently
- The endpoints might not be properly called by the UI
- There could be authentication issues preventing the stats from loading

### 2. Response Time Tracking
To implement response time tracking:

1. **Fix Schema Mismatch**:
   - Option A: Update Go model to use `DurationMs` field with `gorm:"column:duration_ms"`
   - Option B: Update database schema to use `response_time` column

2. **Implement Performance Middleware**:
   - Create middleware that tracks request duration
   - Call `ActivityService.LogPerformance` after each request
   - Handle errors gracefully to avoid impacting request processing

3. **Fix SQL Queries**:
   - Update queries in `GetPerformanceStats` to handle NULL values
   - Use correct column names matching the schema

## Database Schema

### activity_logs table:
```sql
CREATE TABLE IF NOT EXISTS activity_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    type VARCHAR(100) NOT NULL,
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### performance_metrics table:
```sql
CREATE TABLE IF NOT EXISTS performance_metrics (
    id SERIAL PRIMARY KEY,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    duration_ms INTEGER NOT NULL,  -- Note: column is duration_ms, not response_time
    status_code INTEGER NOT NULL,
    user_id INTEGER REFERENCES users(id),
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Key Files and Locations

### Activity/Search Tracking:
- `/internal/models/activity.go` - Activity models
- `/internal/services/activity_service.go` - Activity service with logging and stats
- `/internal/api/memory_handlers.go` - Search handler that logs activities
- `/internal/api/server.go` - Route definitions

### Performance Tracking:
- `/internal/api/middleware.go` - Current middleware implementations
- `/internal/api/server.go:147` - Logger middleware (needs enhancement)
- `/scripts/setup-postgres-schema.sh` - Database schema definitions

### Endpoints:
- `GET /api/v1/memories/stats` - Memory statistics including search counts
- `GET /api/v1/users/activity-stats` - User activity statistics  
- `GET /api/v1/system/performance` - System performance metrics (broken)

## Next Steps

To fix the reported issues:

1. **Debug Search Event Saving**:
   - Add logging to verify LogActivity is being called
   - Check for errors in async goroutine
   - Test the stats endpoints directly

2. **Implement Response Time Tracking**:
   - Fix the schema mismatch in PerformanceMetric model
   - Create performance tracking middleware
   - Update GetPerformanceStats to handle NULL values
   - Test the performance endpoint

3. **Efficiency Considerations**:
   - Use async logging to avoid blocking requests
   - Consider batch inserts for high-traffic scenarios
   - Add appropriate database indexes
   - Implement connection pooling (already done via GORM)

## Testing Commands

```bash
# Test search activity logging
curl -X POST http://localhost:8082/api/v1/memories/search \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"query": "test search"}'

# Check search stats
curl -X GET http://localhost:8082/api/v1/memories/stats \
  -H "X-API-Key: your-api-key"

# Check user activity stats  
curl -X GET http://localhost:8082/api/v1/users/activity-stats \
  -H "X-API-Key: your-api-key"

# Check system performance (currently broken)
curl -X GET http://localhost:8082/api/v1/system/performance \
  -H "X-API-Key: your-api-key"
```

## Summary

The infrastructure for both search event tracking and response time tracking exists in the codebase, but:
- Search events are being saved but might not be displaying correctly
- Response time tracking is not connected (no middleware calls LogPerformance)
- There's a critical schema mismatch between the database and Go models for performance metrics

The fixes are straightforward but require careful implementation to ensure efficiency and reliability.