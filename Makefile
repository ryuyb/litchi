# Litchi Project Makefile

.PHONY: help generate-mocks test test-short build clean

# Default target
help:
	@echo "Litchi Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make generate-mocks  Generate mock implementations using mockery"
	@echo "  make test            Run all tests"
	@echo "  make test-short      Run short tests (skip integration tests)"
	@echo "  make build           Build all binaries"
	@echo "  make clean           Clean generated files"

# Generate mock implementations
generate-mocks:
	@echo "Generating mocks with mockery..."
	mockery
	@echo "Mocks generated successfully"

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
	find ./internal -name "mocks_test.go" -type f -delete
	@echo "Mock files cleaned"