# =============================================================================
# Cashflow Service Makefile
# =============================================================================

# Variables
BINARY_NAME=cashflow
MAIN_PATH=./cmd/main.go
BIN_DIR=./bin
MIGRATIONS_PATH=./migrations

# Load environment variables if .env exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Construct database connection string
DB_STRING=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# =============================================================================
# Core Commands
# =============================================================================

.PHONY: help
help: ## Show this help message
	@echo "Cashflow Service - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

.PHONY: run
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	@go run $(MAIN_PATH)

.PHONY: dev
dev: ## Run with hot reload (requires air)
	@echo "Starting development server with hot reload..."
	@air

.PHONY: clean
clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf tmp/
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# =============================================================================
# Testing Commands
# =============================================================================

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./... -count=1

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -v ./internal/financial -count=1
	@go test -v ./internal/s3 -count=1

.PHONY: test-integration
test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	@go test -v ./... -run Integration -count=1

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out
	@echo ""
	@echo "✓ Coverage report saved to coverage.out"

.PHONY: test-cover-html
test-cover-html: test-cover ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"
	@echo "Opening in browser..."
	@open coverage.html || xdg-open coverage.html || start coverage.html

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@go test -race -v ./... -count=1

# =============================================================================
# Code Quality
# =============================================================================

.PHONY: lint
lint: ## Run linters (requires golangci-lint)
	@echo "Running linters..."
	@golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet passed"

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

# =============================================================================
# Database Migrations
# =============================================================================

.PHONY: migrate-up
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_STRING)" up
	@echo "✓ Migrations applied"

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	@echo "Rolling back last migration..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_STRING)" down 1
	@echo "✓ Migration rolled back"

.PHONY: migrate-status
migrate-status: ## Show migration status
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_STRING)" version

.PHONY: migrate-create
migrate-create: ## Create new migration (usage: make migrate-create NAME=create_users)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required"; \
		echo "Usage: make migrate-create NAME=create_users"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)..."
	@migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)
	@echo "✓ Migration files created"

.PHONY: migrate-force
migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@echo "Forcing migration to version $(VERSION)..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_STRING)" force $(VERSION)
	@echo "✓ Migration forced"

# =============================================================================
# Dependencies
# =============================================================================

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@echo "✓ Dependencies downloaded"

.PHONY: deps-tidy
deps-tidy: ## Tidy and verify dependencies
	@echo "Tidying dependencies..."
	@go mod tidy
	@go mod verify
	@echo "✓ Dependencies tidied"

.PHONY: deps-update
deps-update: ## Update all dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies updated"

# =============================================================================
# Docker Commands
# =============================================================================

.PHONY: docker-up
docker-up: ## Start Docker services (PostgreSQL, Redis)
	@echo "Starting Docker services..."
	@docker-compose up -d
	@echo "✓ Docker services started"

.PHONY: docker-down
docker-down: ## Stop Docker services
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "✓ Docker services stopped"

.PHONY: docker-logs
docker-logs: ## Show Docker service logs
	@docker-compose logs -f

.PHONY: docker-build
docker-build: ## Build Docker image for the application
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .
	@echo "✓ Docker image built"

# =============================================================================
# Utility Commands
# =============================================================================

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✓ Tools installed"

.PHONY: mod-graph
mod-graph: ## Show module dependency graph
	@go mod graph

.PHONY: list-tests
list-tests: ## List all test functions
	@echo "Available tests:"
	@grep -r "^func Test" internal/ --include="*_test.go" | sed 's/.*func /  - /' | sed 's/(t \*testing.T).*//'

# =============================================================================
# CI/CD Targets
# =============================================================================

.PHONY: ci
ci: deps-tidy fmt vet test-race test-cover ## Run full CI pipeline
	@echo "✓ CI pipeline complete"

.PHONY: ci-quick
ci-quick: fmt vet test ## Run quick CI checks
	@echo "✓ Quick CI checks complete"

# Default target
.DEFAULT_GOAL := help
