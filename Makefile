.PHONY: build test clean install lint fmt docs

# Default target
all: build

# Build the binary
build:
	mkdir -p bin
	go build -o bin/go-filemover ./cmd/go-filemover/

# Run unit tests
test:
	go test -v ./cmd/go-filemover/

# Clean build artifacts
clean:
	rm -rf bin/

# Install the binary to GOBIN
install:
	go install ./cmd/go-filemover/

# Format the code
fmt:
	go fmt ./cmd/go-filemover/

# Lint the code using golangci-lint
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./cmd/go-filemover/...; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

# Generate documentation export
docs:
	@echo "# API Documentation" > API.md
	@printf "\n## Go-Filemover API\n" >> API.md
	@go doc -all ./cmd/go-filemover/ >> API.md
