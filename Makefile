.PHONY: build test clean install dev run-tests integration-test

# Build the application
build:
	go build -o timemachine ./cmd/timemachine

# Build for all platforms
build-all:
	./scripts/build.sh

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f timemachine
	rm -f coverage.out
	rm -rf dist/

# Install locally
install:
	go install ./cmd/timemachine

# Development mode (with race detection)
dev:
	go run -race ./cmd/timemachine

# Run integration tests
integration-test:
	./scripts/integration-test.sh

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy