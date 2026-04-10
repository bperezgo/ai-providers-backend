.PHONY: build build-mcp build-mcp-remote build-mock build-stresstest run run-mcp run-mcp-remote run-mock inspect-mcp test test-acceptance clean install db-up db-down db-logs db-reset db-shell test-coverage obs-up obs-down obs-logs up down obs-reset stress-test

# Build the HTTP server
build:
	@echo "Building HTTP server..."
	@go build -o bin/server cmd/server/main.go
	@echo "✓ Build complete: bin/server"

# Build the MCP server (stdio transport for Claude Desktop / Claude Code)
build-mcp:
	@echo "Building MCP server..."
	@go build -o bin/mcp-server cmd/mcp/main.go
	@echo "✓ Build complete: bin/mcp-server"

# Build the remote MCP server (Streamable HTTP transport for Claude Projects / mobile)
build-mcp-remote:
	@echo "Building remote MCP server..."
	@go build -o bin/mcp-remote cmd/mcp-remote/main.go
	@echo "✓ Build complete: bin/mcp-remote"

# Build the mock server
build-mock:
	@echo "Building mock server..."
	@go build -o bin/mockserver cmd/mockserver/main.go
	@echo "✓ Build complete: bin/mockserver"

# Build the stress test binary
build-stresstest:
	@echo "Building stress test binary..."
	@go build -o bin/stresstest cmd/stresstest/main.go
	@echo "✓ Build complete: bin/stresstest"

# Run the mock server (for stress testing without real API calls)
run-mock:
	@echo "Starting mock server..."
	@set -a && . ./.env.mock && set +a && go run cmd/mockserver/main.go

# Run the HTTP server with mock providers (requires mock server running on :9999)
run-with-mock:
	@echo "Starting HTTP server with mock providers..."
	@set -a && . ./.env.mock && set +a && go run cmd/server/main.go

# Run stress test against the backend (default: 50 requests, 5 workers)
# Usage: make stress-test [STRESS_ARGS="-n 100 -c 10"]
stress-test:
	@echo "Running stress test..."
	@go run cmd/stresstest/main.go $(STRESS_ARGS)

# Run the HTTP server
run:
	@echo "Starting HTTP server..."
	@go run cmd/server/main.go

# Run the MCP server (stdio transport — connect via Claude Desktop)
run-mcp:
	@echo "Starting MCP server (stdio)..."
	@go run cmd/mcp/main.go

# Run the remote MCP server (Streamable HTTP — connect via Claude Projects / mobile)
run-mcp-remote:
	@echo "Starting remote MCP server (HTTP)..."
	@go run cmd/mcp-remote/main.go

# Inspect the MCP server with MCP Inspector (web UI at http://localhost:5173)
inspect-mcp: build-mcp
	@echo "Starting MCP Inspector..."
	@echo "  Web UI → http://localhost:5173"
	@set -a && . ./.env && set +a && npx -y @modelcontextprotocol/inspector ./bin/mcp-server

# Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@echo "Starting development server with live reload..."
	@air

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run acceptance tests (requires real API keys: FAL_API_KEY, ANTHROPIC_API_KEY)
test-acceptance:
	@echo "Running acceptance tests (requires FAL_API_KEY + ANTHROPIC_API_KEY)..."
	@go test -tags acceptance -v -timeout 5m -run TestAcceptance ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Install dependencies
install:
	@echo "Installing dependencies..."
	@go mod download
	@echo "✓ Dependencies installed"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "✓ Dependencies tidied"

# Database commands

# Start PostgreSQL container
db-up:
	@echo "Starting PostgreSQL container..."
	@docker-compose up -d postgres
	@echo "✓ PostgreSQL started"
	@echo "Waiting for PostgreSQL to be healthy..."
	@sleep 3
	@echo "✓ PostgreSQL is ready"

# Stop PostgreSQL container
db-down:
	@echo "Stopping PostgreSQL container..."
	@docker-compose down
	@echo "✓ PostgreSQL stopped"

# View PostgreSQL logs
db-logs:
	@docker-compose logs -f postgres

# Run database seeds (idempotent)
db-seed:
	@echo "Running database seeds..."
	@go run ./cmd/seed
	@echo "✓ Seeds complete"

# Reset database (drops volume and recreates)
db-reset:
	@echo "Resetting database..."
	@docker-compose down -v
	@docker-compose up -d postgres
	@sleep 3
	@echo "✓ Database reset complete"
	@echo "Run 'make run' to apply migrations, then 'make db-seed' to seed data"

# Open PostgreSQL shell
db-shell:
	@echo "Opening PostgreSQL shell..."
	@docker exec ai-backend-postgres psql -U ai_backend -d ai_backend

# Docker commands

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t laguna-backend .
	@echo "✓ Docker image built"

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env laguna-backend

# Observability commands

# Start full observability stack (Prometheus, Loki, Tempo, Alloy, Grafana, MinIO)
obs-up:
	@echo "Starting observability stack..."
	@docker-compose up -d minio minio-init prometheus loki tempo alloy grafana alertmanager
	@echo "Observability stack started"
	@echo "  Grafana:      http://localhost:3000 (admin/changeme)"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Alloy UI:     http://localhost:12345"
	@echo "  Alertmanager: http://localhost:9093"
	@echo "  MinIO Console: http://localhost:9001 (loki/supersecret)"

# Stop observability stack
obs-down:
	@echo "Stopping observability stack..."
	@docker-compose stop minio prometheus loki tempo alloy grafana alertmanager
	@echo "Observability stack stopped"

# View observability logs
obs-logs:
	@docker-compose logs -f minio prometheus loki tempo alloy grafana alertmanager

# Start everything (database + observability)
up:
	@echo "Starting all services..."
	@docker-compose up -d
	@echo "All services started"

# Stop everything
down:
	@echo "Stopping all services..."
	@docker-compose down
	@echo "All services stopped"

# Reset observability data (drops volumes)
obs-reset:
	@echo "Resetting observability data..."
	@docker-compose down -v --remove-orphans
	@docker-compose up -d
	@echo "Observability data reset"

# Help
help:
	@echo "Available commands:"
	@echo "  make build         - Build the HTTP server"
	@echo "  make build-mcp     - Build the MCP server (bin/mcp-server)"
	@echo "  make build-mcp-remote - Build remote MCP server (bin/mcp-remote)"
	@echo "  make run           - Run the HTTP server"
	@echo "  make run-mcp       - Run the MCP server (stdio transport)"
	@echo "  make run-mcp-remote - Run remote MCP server (HTTP transport)"
	@echo "  make inspect-mcp   - Inspect MCP server via web UI (http://localhost:5173)"
	@echo "  make dev           - Run with live reload (requires air)"
	@echo "  make test            - Run tests"
	@echo "  make test-acceptance - Run acceptance tests (real API calls)"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install       - Install dependencies"
	@echo "  make tidy          - Tidy dependencies"
	@echo ""
	@echo "Database commands:"
	@echo "  make db-up         - Start PostgreSQL container"
	@echo "  make db-down       - Stop PostgreSQL container"
	@echo "  make db-logs       - View PostgreSQL logs"
	@echo "  make db-reset      - Reset database (drops all data)"
	@echo "  make db-shell      - Open PostgreSQL shell"
	@echo ""
	@echo "Observability commands:"
	@echo "  make obs-up        - Start observability stack (Prometheus, Loki, Tempo, Alloy, Grafana, MinIO)"
	@echo "  make obs-down      - Stop observability stack"
	@echo "  make obs-logs      - View observability logs"
	@echo "  make obs-reset     - Reset observability data (drops volumes)"
	@echo "  make up            - Start all services (database + observability)"
	@echo "  make down          - Stop all services"
	@echo ""
	@echo "Mock server commands:"
	@echo "  make build-mock    - Build the mock server"
	@echo "  make run-mock      - Run the mock server (port 9999)"
	@echo "  make run-with-mock - Run HTTP server pointing to mock providers"
	@echo "  make stress-test   - Run stress test (default 50 reqs, 5 workers)"
	@echo "                       STRESS_ARGS=\"-n 100 -c 10\" for custom config"
	@echo ""
	@echo "Docker commands:"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
