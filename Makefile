# Litchi Project Makefile

.PHONY: help generate-mocks test test-short build clean swagger-gen

# Default target
help:
	@echo "Litchi Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make generate-mocks  Generate mock implementations using mockery"
	@echo "  make swagger-gen     Generate Swagger/OpenAPI documentation"
	@echo "  make test            Run all tests"
	@echo "  make test-short      Run short tests (skip integration tests)"
	@echo "  make build           Build all binaries"
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
	swag init --v3.1 -g cmd/server/main.go -d . -o ./docs/api \
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

# Build all binaries
build:
	go build ./cmd/server
	go build ./cmd/worker

# Clean generated mock files
clean-mocks:
	@echo "Cleaning generated mock files..."
	find ./internal -name "mocks.go" -path "*/domain/*" -type f -delete
	@echo "Mock files cleaned"