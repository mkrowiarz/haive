.PHONY: build test clean install install-latest fmt lint all

BINARY := pm
VERSION := 1.0.0
BUILD_DIR := bin
MAIN_PATH := ./cmd/pm
PACKAGE := github.com/mkrowiarz/mcp-symfony-stack/cmd/pm

all: build

build:
	go build -o $(BINARY) $(MAIN_PATH)

build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(MAIN_PATH)

test:
	go test ./... -v

test-coverage:
	go test ./... -cover

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

install: build
	go install $(MAIN_PATH)

install-latest:
	go install $(PACKAGE)@latest

install-private:
	GOPRIVATE=github.com/mkrowiarz/* go install $(PACKAGE)@latest

fmt:
	go fmt ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

deps:
	go mod download

.PHONY: mcp
mcp: build
	./$(BINARY) --mcp

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary (default)"
	@echo "  build-all     - Build for all platforms"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install binary to ~/go/bin (from local source)"
	@echo "  install-latest- Install latest from remote to ~/go/bin"
	@echo "  install-private- Install from private repo (requires git SSH config)"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  tidy          - Tidy go.mod"
	@echo "  deps          - Download dependencies"
	@echo "  mcp           - Start MCP server"
