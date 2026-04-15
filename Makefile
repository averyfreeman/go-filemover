.PHONY: build test clean install lint fmt docs

# Default target
all: build

# Build the binary
build:
	mkdir -p bin
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
	@printf "\n## Internal/Config\n" >> API.md
	@go doc -all ./internal/config >> API.md
	@printf "\n## Internal/Mover\n" >> API.md
	@go doc -all ./internal/mover >> API.md
	@printf "\n## Internal/FS\n" >> API.md
	@go doc -all ./internal/fs >> API.md
