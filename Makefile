.PHONY: build run clean test version dev install-air swagger

# 獲取當前版本
VERSION := $(shell grep "APP_VERSION" .env | cut -d '=' -f2)

# 獲取 Git 信息
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# 編譯參數
LDFLAGS := -X g38_lottery_service/internal/config.AppVersion=$(VERSION) \
           -X g38_lottery_service/internal/config.GitCommit=$(GIT_COMMIT) \
           -X g38_lottery_service/internal/config.BuildDate=$(BUILD_DATE)

# 二進制名稱
BINARY_NAME := g38_lottery_service
MAIN_PKG := ./main.go

# 腳本目標
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) $(MAIN_PKG)
	@echo "Build completed: bin/$(BINARY_NAME)"

run:
	go run $(MAIN_PKG)

# 使用 Air 進行熱重載開發
dev:
	@echo "Starting development server with Air..."
	@if command -v air > /dev/null 2>&1 ; then \
		air ; \
	else \
		echo "Air not found. Please install it with 'make install-air' or manually: go install github.com/air-verse/air@latest" ; \
		exit 1 ; \
	fi

# 安裝 Air 工具
install-air:
	@echo "Installing Air for hot reload..."
	go install github.com/air-verse/air@latest

# 生成 Swagger 文檔
swagger:
	@echo "Generating Swagger documentation..."
	@if command -v swag > /dev/null 2>&1 ; then \
		swag init ; \
	else \
		echo "Swag not found. Installing swag..." ; \
		go install github.com/swaggo/swag/cmd/swag@latest ; \
		swag init ; \
	fi
	@echo "Swagger documentation generated in ./docs directory"

clean:
	@echo "Cleaning..."
	rm -f bin/$(BINARY_NAME)
	rm -rf ./tmp
	go clean
	@echo "Done!"

test:
	go test ./...

version:
	@echo "Current version: $(VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Build date: $(BUILD_DATE)"

# 版本管理目標
bump-patch:
	@echo "Bumping patch version..."
	@VERSION_PARTS=$$(echo $(VERSION) | tr '.' ' '); \
	MAJOR=$$(echo $$VERSION_PARTS | cut -d' ' -f1); \
	MINOR=$$(echo $$VERSION_PARTS | cut -d' ' -f2); \
	PATCH=$$(echo $$VERSION_PARTS | cut -d' ' -f3); \
	NEW_PATCH=$$(expr $$PATCH + 1); \
	NEW_VERSION="$$MAJOR.$$MINOR.$$NEW_PATCH"; \
	sed -i '' "s/APP_VERSION=.*/APP_VERSION=$$NEW_VERSION/" .env; \
	echo "Version bumped to $$NEW_VERSION"

bump-minor:
	@echo "Bumping minor version..."
	@VERSION_PARTS=$$(echo $(VERSION) | tr '.' ' '); \
	MAJOR=$$(echo $$VERSION_PARTS | cut -d' ' -f1); \
	MINOR=$$(echo $$VERSION_PARTS | cut -d' ' -f2); \
	NEW_MINOR=$$(expr $$MINOR + 1); \
	NEW_VERSION="$$MAJOR.$$NEW_MINOR.0"; \
	sed -i '' "s/APP_VERSION=.*/APP_VERSION=$$NEW_VERSION/" .env; \
	echo "Version bumped to $$NEW_VERSION"

bump-major:
	@echo "Bumping major version..."
	@VERSION_PARTS=$$(echo $(VERSION) | tr '.' ' '); \
	MAJOR=$$(echo $$VERSION_PARTS | cut -d' ' -f1); \
	NEW_MAJOR=$$(expr $$MAJOR + 1); \
	NEW_VERSION="$$NEW_MAJOR.0.0"; \
	sed -i '' "s/APP_VERSION=.*/APP_VERSION=$$NEW_VERSION/" .env; \
	echo "Version bumped to $$NEW_VERSION"

help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application"
	@echo "  dev         - Run the application with hot reload using Air"
	@echo "  install-air - Install Air tool for hot reload"
	@echo "  swagger     - Generate Swagger documentation"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  version     - Show current version info"
	@echo "  bump-patch  - Increment patch version (x.y.Z -> x.y.Z+1)"
	@echo "  bump-minor  - Increment minor version (x.Y.z -> x.Y+1.0)"
	@echo "  bump-major  - Increment major version (X.y.z -> X+1.0.0)"
	@echo "  help        - Show this help" 