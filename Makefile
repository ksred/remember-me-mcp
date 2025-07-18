.PHONY: build run test test-verbose test-coverage lint clean docker-up docker-down setup docker-setup help

# Default target
all: build

# Build the binary
build:
	go build -o remember-me-mcp cmd/main.go
	go build -o remember-me-http cmd/http-server/main.go

# Run the MCP application
run-mcp:
	@if [ -f .env.dev ]; then \
		echo "Loading .env.dev for development..."; \
		export $$(cat .env.dev | grep -v '^#' | xargs) && go run cmd/main.go; \
	else \
		go run cmd/main.go; \
	fi

# Run the HTTP server
run-http:
	@if [ -f .env.http ]; then \
		echo "Loading .env.http for HTTP server..."; \
		export $$(cat .env.http | grep -v '^#' | xargs) && go run cmd/http-server/main.go; \
	elif [ -f .env.dev ]; then \
		echo "Loading .env.dev for development..."; \
		export $$(cat .env.dev | grep -v '^#' | xargs) && go run cmd/http-server/main.go; \
	else \
		go run cmd/http-server/main.go; \
	fi

# Generate Swagger documentation
swagger:
	swag init -g cmd/http-server/main.go -o docs

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run tests with coverage report
test-coverage-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -f remember-me-mcp memory-server coverage.out coverage.html

# Install dependencies
deps:
	go mod download

# Install binary to system path
install: build
	@echo "Installing remember-me-mcp binary..."
	@sudo cp remember-me-mcp /usr/local/bin/
	@sudo chmod +x /usr/local/bin/remember-me-mcp
	@echo "✅ Binary installed to /usr/local/bin/remember-me-mcp"
	@echo "   You can now configure Claude Desktop to use: /usr/local/bin/remember-me-mcp"

# Uninstall binary from system path
uninstall:
	@echo "Uninstalling remember-me-mcp binary..."
	@sudo rm -f /usr/local/bin/remember-me-mcp
	@echo "✅ Binary removed from /usr/local/bin/"

# Local setup (native installation)
setup:
	@echo "Running local setup script..."
	@chmod +x scripts/setup.sh
	@./scripts/setup.sh

# Development setup (Docker DB + local binary)
dev-setup:
	@echo "Running development setup script..."
	@chmod +x scripts/dev-setup.sh
	@./scripts/dev-setup.sh

# Docker setup
docker-setup:
	@echo "Running Docker setup script..."
	@chmod +x scripts/docker-setup.sh
	@./scripts/docker-setup.sh

# Start Docker containers (full stack)
docker-up:
	docker-compose up -d --build

# Stop Docker containers (full stack)
docker-down:
	docker-compose down

# View Docker logs (full stack)
docker-logs:
	docker-compose logs -f

# Docker cleanup (remove volumes)
docker-clean:
	docker-compose down -v
	docker rmi remember-me-mcp_remember-me-mcp 2>/dev/null || true

# Check Docker status (full stack)
docker-status:
	docker-compose ps

# Database-only Docker commands for local development
docker-db:
	@echo "Starting PostgreSQL database container..."
	docker-compose -f docker-compose.db.yml up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec remember-me-postgres-dev pg_isready -U postgres -d remember_me >/dev/null 2>&1; do \
		echo -n "."; \
		sleep 2; \
	done
	@echo ""
	@echo "PostgreSQL is ready!"
	@echo "Connection details:"
	@echo "  Host: localhost"
	@echo "  Port: 5432"
	@echo "  Database: remember_me"
	@echo "  User: postgres"
	@echo "  Password: devpassword (or check .env.dev)"
	@echo ""
	@echo "Test connection: make db-test"
	@echo "Run server: make run"

# Stop database container
docker-db-down:
	@echo "Stopping PostgreSQL database container..."
	docker-compose -f docker-compose.db.yml down

# View database logs
docker-db-logs:
	docker-compose -f docker-compose.db.yml logs -f postgres

# Check database status
docker-db-status:
	docker-compose -f docker-compose.db.yml ps

# Clean database container and volumes
docker-db-clean:
	@echo "Stopping and removing PostgreSQL container and volumes..."
	docker-compose -f docker-compose.db.yml down -v
	@echo "Database container and volumes removed"

# Restart database container
docker-db-restart: docker-db-down docker-db

# Connect to database container
docker-db-connect:
	docker exec -it remember-me-postgres-dev psql -U postgres -d remember_me

# Test database connection
db-test:
	@echo "Testing database connection..."
	@if command -v psql > /dev/null; then \
		psql -h localhost -U postgres -d remember_me -c "SELECT version();" -c "SELECT * FROM pg_extension WHERE extname = 'vector';" 2>/dev/null || echo "Connection failed. Is the database running? Try: make docker-db"; \
	else \
		echo "psql not found. Install PostgreSQL client or use: make docker-db-connect"; \
	fi

# Test MCP server manually
test-mcp:
	@echo "Testing MCP server..."
	@cd scripts && go run test-mcp.go

# Configure Claude Desktop for development
configure-claude:
	@echo "Configuring Claude Desktop for development..."
	@chmod +x scripts/claude-desktop-config.sh
	@./scripts/claude-desktop-config.sh development

# Configure Claude Desktop for production (after install)
configure-claude-production:
	@echo "Configuring Claude Desktop for production..."
	@chmod +x scripts/claude-desktop-config.sh
	@./scripts/claude-desktop-config.sh production

# Development mode with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without hot reload..."; \
		go run cmd/main.go; \
	fi

# Security scan
security:
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Check for updates
update:
	go get -u ./...
	go mod tidy

# Release build (optimized)
release:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o remember-me-mcp cmd/main.go

# Cross-compilation targets
build-linux:
	GOOS=linux GOARCH=amd64 go build -o remember-me-mcp-linux cmd/main.go

build-windows:
	GOOS=windows GOARCH=amd64 go build -o remember-me-mcp-windows.exe cmd/main.go

build-macos:
	GOOS=darwin GOARCH=amd64 go build -o remember-me-mcp-macos cmd/main.go

build-all: build-linux build-windows build-macos

# Database operations
db-create:
	@echo "Creating database..."
	@createdb remember_me 2>/dev/null || echo "Database may already exist"
	@psql -d remember_me -c "CREATE EXTENSION IF NOT EXISTS vector;"

db-drop:
	@echo "Dropping database..."
	@dropdb remember_me 2>/dev/null || echo "Database may not exist"

db-reset: db-drop db-create

# Help target
help:
	@echo "Remember Me MCP Server - Available Make targets:"
	@echo ""
	@echo "Development:"
	@echo "  build         Build the binary"
	@echo "  run           Run the application"
	@echo "  dev           Run in development mode with hot reload"
	@echo "  test          Run tests"
	@echo "  test-verbose  Run tests with verbose output"
	@echo "  test-coverage Run tests with coverage"
	@echo "  test-coverage-html Generate HTML coverage report"
	@echo "  lint          Run linter"
	@echo "  fmt           Format code"
	@echo "  tidy          Tidy dependencies"
	@echo "  deps          Install dependencies"
	@echo "  update        Update dependencies"
	@echo "  security      Run security scan"
	@echo ""
	@echo "Installation:"
	@echo "  install       Install binary to /usr/local/bin/ (requires sudo)"
	@echo "  uninstall     Remove binary from /usr/local/bin/ (requires sudo)"
	@echo ""
	@echo "Setup:"
	@echo "  setup         Run local setup script (full native install)"
	@echo "  dev-setup     Run development setup (Docker DB + local binary)"
	@echo "  docker-setup  Run Docker setup script (full containerized)"
	@echo ""
	@echo "Docker (Full Stack):"
	@echo "  docker-up     Start Docker containers (full stack)"
	@echo "  docker-down   Stop Docker containers (full stack)"
	@echo "  docker-logs   View Docker logs (full stack)"
	@echo "  docker-status Check Docker status (full stack)"
	@echo "  docker-clean  Clean Docker volumes and images"
	@echo ""
	@echo "Docker (Database Only - for local development):"
	@echo "  docker-db         Start PostgreSQL container"
	@echo "  docker-db-down    Stop PostgreSQL container"
	@echo "  docker-db-logs    View PostgreSQL logs"
	@echo "  docker-db-status  Check PostgreSQL status"
	@echo "  docker-db-clean   Clean PostgreSQL container and volumes"
	@echo "  docker-db-restart Restart PostgreSQL container"
	@echo "  docker-db-connect Connect to PostgreSQL container"
	@echo "  db-test           Test database connection"
	@echo ""
	@echo "Database:"
	@echo "  db-create     Create database"
	@echo "  db-drop       Drop database"
	@echo "  db-reset      Reset database"
	@echo ""
	@echo "Testing:"
	@echo "  test-mcp      Test MCP server manually"
	@echo "  configure-claude Configure Claude Desktop for development"
	@echo "  configure-claude-production Configure Claude Desktop for production"
	@echo ""
	@echo "Build:"
	@echo "  release       Build optimized release binary"
	@echo "  build-linux   Build for Linux"
	@echo "  build-windows Build for Windows"
	@echo "  build-macos   Build for macOS"
	@echo "  build-all     Build for all platforms"
	@echo ""
	@echo "Utility:"
	@echo "  clean         Clean build artifacts"
	@echo "  help          Show this help message"