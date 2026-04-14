.PHONY: build test

build:
	go build -o bin/go-filemover ./cmd/main.go

test:
	go test -v ./...
