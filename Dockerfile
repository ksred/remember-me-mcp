# Remember Me MCP Server Dockerfile
# Multi-stage build for optimal image size

# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o remember-me-mcp ./cmd/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o remember-me-http ./cmd/http-server/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/remember-me-mcp .
COPY --from=builder /app/remember-me-http .

# Copy configuration files
COPY --from=builder /app/configs/ ./configs/

# Create directories for logs and data
RUN mkdir -p /app/logs /app/data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (if needed for health checks)
EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep remember-me-mcp || exit 1

# Default to MCP server, but allow override to run HTTP server
ENTRYPOINT ["/bin/sh", "-c"]
CMD ["./remember-me-mcp"]