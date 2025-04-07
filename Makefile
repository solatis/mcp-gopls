.PHONY: build clean test

# Variables
BINARY_NAME=mcp-gopls
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

# Build the project
build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/...

# Clean build artifacts
clean:
	$(GO) clean
	rm -f $(BINARY_NAME)

# Run tests
test:
	$(GO) test -v ./...

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Default target
all: deps build 