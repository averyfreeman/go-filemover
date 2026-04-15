.PHONY: build test clean install lint fmt

# Default target
all: build

# Build the binary
build:
	go build -o bin/go-filemover ./cmd/go-filemover/main.go

# Run unit tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Install the binary to GOBIN
install:
	go install ./cmd/go-filemover

# Format the code
fmt:
	go fmt ./...

# Lint the code using golangci-lint
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

# Generate documentation export
docs:
	@echo "# API Documentation" > API.md
	@echo "## Internal/Config" >> API.md
	@go doc -all ./internal/config >> API.md
	@echo "\n## Internal/Mover" >> API.md
	@go doc -all ./internal/mover >> API.md
	@echo "\n## Internal/FS" >> API.md
	@go doc -all ./internal/fs >> API.md
