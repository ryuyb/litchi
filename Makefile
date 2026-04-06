# Litchi Project Makefile

.PHONY: help generate-mocks test test-short build clean swagger-gen build-embed frontend-build copy-dist dev

# Default target
help:
	@echo "Litchi Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make generate-mocks  Generate mock implementations using mockery"
	@echo "  make swagger-gen     Generate Swagger/OpenAPI documentation"
	@echo "  make test            Run all tests"
	@echo "  make test-short      Run short tests (skip integration tests)"
	@echo "  make build           Build backend binary (development mode)"
	@echo "  make build-embed     Build production binary with embedded frontend"
	@echo "  make frontend-build  Build frontend (TanStack Start SPA mode)"
	@echo "  make dev             Run backend in development mode"
	@echo "  make clean           Clean generated files"

# Generate mock implementations
generate-mocks:
	@echo "Generating mocks with mockery..."
	mockery
	@echo "Mocks generated successfully"

# Generate Swagger/OpenAPI documentation (OpenAPI 3.1)
swagger-gen:
	@echo "Generating Swagger documentation (OpenAPI 3.1)..."
	@mkdir -p ./docs/api
	swag init --v3.1 -g cmd/litchi/server.go -d . -o ./docs/api \
		--parseInternal --parseDependencyLevel 3 --outputTypes go,json,yaml --propertyStrategy camelcase
	@echo "Swagger documentation generated in ./docs/api"

# Run all tests (including integration tests with Docker)
test:
	go test ./... -v

# Run short tests (skip integration tests)
test-short:
	go test ./... -short -v

# Run integration tests only
test-integration:
	go test ./... -v -run Integration

# Build backend binary (development mode - no frontend embedded)
build:
	go build -ldflags "-X main.Version=dev" ./cmd/litchi

# Build frontend (TanStack Start SPA mode)
frontend-build:
	@echo "Building frontend..."
	cd web && pnpm build
	@echo "Frontend built in web/dist"

# Copy frontend dist to static package for embedding
copy-dist:
	@echo "Copying frontend dist to static package..."
	rm -rf internal/infrastructure/static/dist
	cp -r web/dist internal/infrastructure/static/dist
	@echo "Frontend copied to internal/infrastructure/static/dist"

# Build production binary with embedded frontend
build-embed: frontend-build copy-dist
	@echo "Building production binary with embedded frontend..."
	go build -tags embed -ldflags "-X main.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) -X main.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown) -X main.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/litchi
	@echo "Production binary ready"

# Run backend in development mode
dev:
	go run ./cmd/litchi server

# Clean generated mock files
clean-mocks:
	@echo "Cleaning generated mock files..."
	find ./internal -name "mocks.go" -path "*/domain/*" -type f -delete
	@echo "Mock files cleaned"

# Clean all generated files including frontend dist
clean: clean-mocks
	@echo "Cleaning frontend dist..."
	rm -rf internal/infrastructure/static/dist
	mkdir -p internal/infrastructure/static/dist
	touch internal/infrastructure/static/dist/.gitkeep
	@echo "All cleaned"