.PHONY: all build run test test-verbose test-coverage bench clean deps

# Variables
BINARY_NAME=pixel-tracker
GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOGET=$(GO) get
GOMOD=$(GO) mod

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v

# Run the application
run:
	$(GO) run main.go

# Run tests
test: build
	$(GOTEST) -v ./...

# Run tests with verbose output
test-verbose:
	$(GOTEST) -v -run . ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	$(GO) tool cover -func=coverage.out

# Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Run tests with race detector
test-race:
	$(GOTEST) -race -v ./...

# Run a specific test
test-one:
	@read -p "Enter test name: " test_name; \
	$(GOTEST) -v -run $$test_name ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	$(GO) fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run the application with live reload (requires air)
dev:
	air

# Docker build
docker-build:
	docker build -t $(BINARY_NAME) .

# Docker run
docker-run:
	docker run -p 8080:8080 $(BINARY_NAME)

# Help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make run          - Run the application"
	@echo "  make test         - Run tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-coverage- Run tests with coverage report"
	@echo "  make bench        - Run benchmarks"
	@echo "  make test-race    - Run tests with race detector"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make deps         - Download dependencies"
	@echo "  make fmt          - Format code"
	@echo "  make help         - Show this help message"
