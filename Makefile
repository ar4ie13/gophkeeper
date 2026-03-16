# Define variables
VERSION ?= v.0.26
BUILD_DATE := $(shell date +'%Y/%m/%d %H:%M:%S')
BUILD_COMMIT := $(shell git log -1 --oneline 2>/dev/null || echo "N/A")
BINARY_NAME_SERVER := gophkeeper
BINARY_NAME_CLIENT := client
SRC_DIR_SERVER := ./cmd/server
SRC_DIR_CLIENT := ./cmd/client
BUILD_DIR := bin

# Phony targets to prevent conflicts with files of the same name
.PHONY: build run pg-start pg-stop build-c run-c build-client-linux build-client-macos build-client-windows build-client-all

# Build the Go server application
build:
	@echo "Building $(BINARY_NAME_SERVER)..."
	go build -ldflags "-X 'main.buildVersion=$(VERSION)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) $(SRC_DIR_SERVER)/main.go

# Run the Go server application
run: build
	@echo "Running $(BINARY_NAME_SERVER)..."
	$(BUILD_DIR)/$(BINARY_NAME_SERVER)

# Build the Go client application
build-c:
	@echo "Building $(BINARY_NAME_CLIENT)..."
	go build -ldflags "-X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildVersion=$(VERSION)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildDate=$(BUILD_DATE)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT) $(SRC_DIR_CLIENT)/main.go

# Run the Go client application
run-c: build-c
	@echo "Running $(BINARY_NAME_CLIENT)..."
	$(BUILD_DIR)/$(BINARY_NAME_CLIENT) -server https://localhost:8080 -ca-cert cert.pem

# Build client for Linux (amd64 and arm64)
build-client-linux:
	@echo "Building $(BINARY_NAME_CLIENT) for Linux amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildVersion=$(VERSION)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildDate=$(BUILD_DATE)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-linux-amd64 $(SRC_DIR_CLIENT)/main.go
	@echo "Building $(BINARY_NAME_CLIENT) for Linux arm64..."
	GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildVersion=$(VERSION)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildDate=$(BUILD_DATE)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-linux-arm64 $(SRC_DIR_CLIENT)/main.go

# Build client for macOS (amd64 and arm64)
build-client-macos:
	@echo "Building $(BINARY_NAME_CLIENT) for macOS amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildVersion=$(VERSION)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildDate=$(BUILD_DATE)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-darwin-amd64 $(SRC_DIR_CLIENT)/main.go
	@echo "Building $(BINARY_NAME_CLIENT) for macOS arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildVersion=$(VERSION)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildDate=$(BUILD_DATE)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-darwin-arm64 $(SRC_DIR_CLIENT)/main.go

# Build client for Windows (amd64)
build-client-windows:
	@echo "Building $(BINARY_NAME_CLIENT) for Windows amd64..."
	GOOS=windows GOARCH=amd64 go build -ldflags "-X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildVersion=$(VERSION)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildDate=$(BUILD_DATE)' -X 'github.com/ar4ie13/gophkeeper/internal/client/config.buildCommit=$(BUILD_COMMIT)'" -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT)-windows-amd64.exe $(SRC_DIR_CLIENT)/main.go

# Build client for all platforms
build-client-all: build-client-linux build-client-macos build-client-windows
	@echo "All client builds completed!"

# Starts postgres docker container
pg-start:
	docker start pg

# Stops postgres docker container
pg-stop:
	docker stop pg
