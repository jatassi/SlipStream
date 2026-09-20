.PHONY: dev dev-mode dev-backend dev-backend-devmode dev-frontend build build-backend build-frontend clean install test test-unit test-integration test-coverage test-verbose lint lint-fix lint-verbose lint-new deadcode wire generate-slots new-module

# Build flags for embedding values at build time (set via environment variables or make arguments)
# Example: make build-backend VERSION=1.2.3 TMDB_API_KEY=xxx TVDB_API_KEY=yyy OMDB_API_KEY=zzz
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
ifdef OMDB_API_KEY
	LDFLAGS += -X 'github.com/slipstream/slipstream/internal/config.EmbeddedOMDBKey=$(OMDB_API_KEY)'
endif

# Development
dev: ## Run both backend and frontend in development mode
	@echo "Starting development servers..."
	@make -j2 dev-backend dev-frontend

dev-mode: ## Run both servers with developer mode enabled at startup
	@echo "Starting development servers (developer mode)..."
	@make -j2 dev-backend-devmode dev-frontend

# Dev servers run through portless (https://portless.sh): each gets a stable
# named URL instead of a port, and every git worktree gets its own prefixed
# name, so parallel checkouts never fight over :3000 or :8080. The backend is
# built and exec'd directly and Vite runs through its own binary because
# `go run` and `bun run` swallow the signals portless uses to stop them;
# SLIPSTREAM_DEV_BUILD=1 keeps the dev-build behaviour `go run` used to imply.
#   backend  -> $(portless get slipstream-api)
#   frontend -> $(portless get slipstream)
dev-backend: ## Run Go backend in development mode
	@echo "Starting backend at $$(portless get slipstream-api)..."
	@mkdir -p web/dist && touch web/dist/.keep && go build -o bin/slipstream ./cmd/slipstream
	@SLIPSTREAM_DEV_BUILD=1 portless run --name slipstream-api sh -c 'SLIPSTREAM_SERVER_PORT="$$PORT" exec ./bin/slipstream'

dev-backend-devmode: ## Run Go backend with developer mode enabled at startup
	@echo "Starting backend at $$(portless get slipstream-api) (developer mode)..."
	@mkdir -p web/dist && touch web/dist/.keep && go build -o bin/slipstream ./cmd/slipstream
	@SLIPSTREAM_DEV_BUILD=1 portless run --name slipstream-api sh -c 'SLIPSTREAM_SERVER_PORT="$$PORT" exec ./bin/slipstream --dev-mode'

dev-frontend: ## Run Vite frontend in development mode
	@echo "Starting frontend at $$(portless get slipstream)..."
	@cd web && SLIPSTREAM_API_ORIGIN="$$(portless get slipstream-api)" portless run --name slipstream node_modules/.bin/vite

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
	@go test -race ./...

test-verbose: ## Run all tests with verbose output
	@echo "Running all tests (verbose)..."
	@go test -v -race ./...

test-unit: ## Run unit tests only (scanner, quality, organizer)
	@echo "Running unit tests..."
	@go test -v -race ./internal/library/scanner/... ./internal/library/quality/... ./internal/library/organizer/...

test-integration: ## Run integration tests (services, API)
	@echo "Running integration tests..."
	@go test -v -race ./internal/library/movies/... ./internal/library/tv/... ./internal/api/...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	@go test -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@go tool cover -func=coverage/coverage.out | tail -1
	@echo "Coverage report generated at coverage/coverage.html"

test-coverage-view: ## Run tests and open coverage report in browser
	@make test-coverage
	@echo "Opening coverage report..."
	@start coverage/coverage.html 2>/dev/null || open coverage/coverage.html 2>/dev/null || xdg-open coverage/coverage.html 2>/dev/null || echo "Please open coverage/coverage.html manually"

# Linting (Go)
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint")

DEADCODE := $(shell command -v deadcode 2>/dev/null || echo "$(shell go env GOPATH)/bin/deadcode")

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run ./...

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@$(GOLANGCI_LINT) run --fix ./...

lint-verbose: ## Run golangci-lint with verbose output
	@echo "Running golangci-lint (verbose)..."
	@$(GOLANGCI_LINT) run -v ./...

lint-new: ## Lint only new/changed code vs main
	@echo "Running golangci-lint on new code..."
	@$(GOLANGCI_LINT) run --new-from-rev=origin/main ./...

deadcode: ## Find unreachable functions via whole-program analysis
	@echo "Running deadcode analysis..."
	@$(DEADCODE) -test ./...

# Code Generation
generate-slots: ## Regenerate slot SQL queries from template
	@cd internal/database/queries && go generate .

# Wire (Dependency Injection)
wire: ## Regenerate Wire dependency injection code
	@echo "Regenerating Wire..."
	@cd internal/api && go run github.com/google/wire/cmd/wire@latest

# Module Scaffolding
new-module: ## Scaffold a new module (usage: make new-module MODULE_ID=music)
	@if [ -z "$(MODULE_ID)" ]; then echo "Usage: make new-module MODULE_ID=<id>"; exit 1; fi
	@go run ./scripts/new-module $(MODULE_ID)

# Help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
