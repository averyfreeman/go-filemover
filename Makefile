.PHONY: build test clean install lint fmt docs init

# Default target
all: build

# Initialize Go module at the project root
init:
	@if [ ! -f go.mod ]; then \
		go mod init go-filemover; \
	fi
	go mod tidy

# Build the binary by compiling the entire command directory
build: init
	mkdir -p bin
	go build -o bin/go-filemover ./cmd/go-filemover/

# Run unit tests
test: init
	go test -v ./cmd/go-filemover/

# Clean build artifacts and generated module files
clean:
	rm -rf bin/
	rm -f go.mod go.sum

# Install the binary to GOBIN
install: init
	go install ./cmd/go-filemover/

# Format the code
fmt:
	go fmt ./cmd/go-filemover/

# Lint the code using golangci-lint
lint: init
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
