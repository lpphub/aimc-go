.PHONY: build run test clean dev

APP_NAME=aimc-go
BUILD_DIR=./build
GO=go

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) main.go

run:
	$(GO) run main.go

test:
	$(GO) test -v ./...

clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned"

dev:
	@go mod tidy
	@$(GO) run main.go