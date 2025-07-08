# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=main
BINARY_UNIX=$(BINARY_NAME)_unix

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v .

# Build for linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v .

# Run the application
run:
	$(GOCMD) run cmd/main.go

# Clean build files
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run tests with coverage and generate XML report
test-coverage-xml:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	gocover-cobertura < coverage.out > coverage.xml

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run linter
lint:
	golangci-lint run ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Docker commands
docker-build:
	docker build -t web-page-analyzer .

docker-run:
	docker run -p 8080:8080 web-page-analyzer

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux"
	@echo "  run           - Run the application"
	@echo "  clean         - Clean build files"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-coverage-xml - Run tests with coverage and generate XML report"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  vet           - Vet code"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  help          - Show this help"

.PHONY: build build-linux run dev clean test test-coverage test-coverage-xml deps swagger fmt lint vet mocks install-tools docker-build docker-run help
