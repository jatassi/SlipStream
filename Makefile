.PHONY: dev dev-backend dev-frontend build build-backend build-frontend clean install test test-unit test-integration test-coverage test-verbose

# Build flags for embedding values at build time (set via environment variables or make arguments)
# Example: make build-backend VERSION=1.2.3 TMDB_API_KEY=xxx TVDB_API_KEY=yyy
LDFLAGS := -s -w
ifdef VERSION
	LDFLAGS += -X 'github.com/slipstream/slipstream/internal/config.Version=$(VERSION)'
endif
ifdef TMDB_API_KEY
	LDFLAGS += -X 'github.com/slipstream/slipstream/internal/config.EmbeddedTMDBKey=$(TMDB_API_KEY)'
endif
ifdef TVDB_API_KEY
	LDFLAGS += -X 'github.com/slipstream/slipstream/internal/config.EmbeddedTVDBKey=$(TVDB_API_KEY)'
endif

# Development
dev: ## Run both backend and frontend in development mode
	@echo "Starting development servers..."
	@make -j2 dev-backend dev-frontend

dev-backend: ## Run Go backend in development mode
	@echo "Starting backend on :8080..."
	@cd cmd/slipstream && go run .

dev-frontend: ## Run Vite frontend in development mode
	@echo "Starting frontend on :3000..."
	@cd web && bun run dev

# Build
build: build-backend build-frontend ## Build both backend and frontend

build-backend: ## Build Go backend (use VERSION, TMDB_API_KEY, TVDB_API_KEY to embed values)
	@echo "Building backend..."
	@go build -ldflags "$(LDFLAGS)" -o bin/slipstream ./cmd/slipstream

build-frontend: ## Build frontend for production
	@echo "Building frontend..."
	@cd web && bun run build

# Install dependencies
install: ## Install all dependencies
	@echo "Installing Go dependencies..."
	@go mod download
	@echo "Installing frontend dependencies..."
	@cd web && bun install

# Clean
clean: ## Clean build artifacts
	@rm -rf bin/
	@rm -rf web/dist/
	@rm -rf coverage/

# Testing
test: ## Run all tests
	@echo "Running all tests..."
	@go test ./...

test-verbose: ## Run all tests with verbose output
	@echo "Running all tests (verbose)..."
	@go test -v ./...

test-unit: ## Run unit tests only (scanner, quality, organizer)
	@echo "Running unit tests..."
	@go test -v ./internal/library/scanner/... ./internal/library/quality/... ./internal/library/organizer/...

test-integration: ## Run integration tests (services, API)
	@echo "Running integration tests..."
	@go test -v ./internal/library/movies/... ./internal/library/tv/... ./internal/api/...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@go tool cover -func=coverage/coverage.out | tail -1
	@echo "Coverage report generated at coverage/coverage.html"

test-coverage-view: ## Run tests and open coverage report in browser
	@make test-coverage
	@echo "Opening coverage report..."
	@start coverage/coverage.html 2>/dev/null || open coverage/coverage.html 2>/dev/null || xdg-open coverage/coverage.html 2>/dev/null || echo "Please open coverage/coverage.html manually"

# Help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
