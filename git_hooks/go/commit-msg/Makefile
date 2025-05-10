# Define the binary name
BINARY_NAME=commit-msg

# Define the Go source directory (if not in root)
SRC_DIR=.

# Set the main package (if needed)
MAIN_PKG=commit-msg.go

# Go commands
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(MAIN_PKG)

run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

test:
	@echo "Running tests..."
	@go test ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)

lint:
	@echo "Linting code..."
	@golangci-lint run

mod-tidy:
	@echo "Tidying up Go modules..."
	@go mod tidy

install-deps:
	@echo "Installing dependencies..."
	@go mod download

.PHONY: build run test fmt vet clean lint mod-tidy install-deps
