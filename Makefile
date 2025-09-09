.PHONY: build test clean install dev run-tests integration-test

# Build the application
build:
	go build -o timemachine ./cmd/timemachine

# Build for all platforms  
build-all:
	@echo "Building cross-platform binaries..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/timemachine-linux-amd64 ./cmd/timemachine
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o dist/timemachine-linux-arm64 ./cmd/timemachine
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o dist/timemachine-macos-amd64 ./cmd/timemachine
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o dist/timemachine-macos-arm64 ./cmd/timemachine
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/timemachine-windows-amd64.exe ./cmd/timemachine
	@echo "✅ Cross-platform binaries built in dist/"

# Create release (for maintainers)
release:
	@echo "Creating GitHub release..."
	@read -p "Enter release tag (e.g., v1.0.0): " tag; \
	git tag $$tag && git push origin $$tag

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