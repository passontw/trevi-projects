.PHONY: build run clean test version dev install-air swagger test test-coverage test-report test-all test-package-coverage test-handlers test-services test-merge-coverage test-html-all test-full install-cobertura test-cobertura test-gitlab

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
MAIN_PKG := ./cmd/lottery_server

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
		cd $(MAIN_PKG) && swag init -g main.go -o ../../docs/swagger ; \
	else \
		echo "Swag not found. Installing swag..." ; \
		go install github.com/swaggo/swag/cmd/swag@latest ; \
		cd $(MAIN_PKG) && swag init -g main.go -o ../../docs/swagger ; \
	fi
	@echo "Swagger documentation generated in ./docs/swagger directory"

# 運行單元測試
test:
	@echo "=== 執行單元測試 ==="
	@go test ./... -v

# 運行測試並生成覆蓋率報告
test-coverage:
	@echo "=== 執行測試並生成覆蓋率報告 ==="
	@mkdir -p coverage
	@go test -v -coverprofile=coverage/coverage.out ./... 
	@go tool cover -func=coverage/coverage.out | grep total | awk '{print "總覆蓋率: " $$3}'
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "詳細覆蓋率報告已保存至 coverage/coverage.html"
	@echo "您可以使用瀏覽器打開此文件查看詳細的覆蓋率信息"

# 運行測試並生成 JUnit 報告
test-report:
	@echo "=== 生成 JUnit 測試報告 ==="
	@mkdir -p coverage
	@if ! command -v go-junit-report > /dev/null 2>&1; then \
		echo "安裝 go-junit-report..."; \
		go install github.com/jstemmer/go-junit-report/v2@latest; \
	fi
	@go test -v ./... 2>&1 | go-junit-report > coverage/report.xml
	@echo "JUnit 報告已保存至 coverage/report.xml"

# 運行所有測試，生成覆蓋率和 JUnit 報告
test-all: test-coverage test-report
	@echo "All tests complete and reports generated in coverage/ directory"

clean:
	@echo "Cleaning..."
	rm -f bin/$(BINARY_NAME)
	rm -rf ./tmp
	rm -rf ./reports
	rm -rf ./coverage
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
	@echo "可用的測試目標:"
	@echo "  test                    - 執行單元測試"
	@echo "  test-coverage           - 運行測試並生成覆蓋率報告"
	@echo "  test-packages           - 為所有包創建詳細的測試覆蓋率報告"
	@echo "  test-report             - 運行測試並生成 JUnit 報告"
	@echo "  test-badge              - 生成測試覆蓋率徽章"
	@echo "  test-visualize          - 用 SVG 可視化測試覆蓋率"
	@echo "  test-html-all           - 生成所有 HTML 測試報告"
	@echo "  test-full               - 執行完整測試流程，生成多種報告"
	@echo "  test-handlers           - 測試所有處理程序的覆蓋率"
	@echo "  test-services           - 測試所有服務的覆蓋率"
	@echo "  test-merge-coverage     - 生成單一檔案的整合覆蓋率報告"
	@echo "  test-package-coverage   - 為特定目錄生成測試覆蓋率報告 (使用: make test-package-coverage PKG=./path/to/pkg)"
	@echo "  test-cobertura          - 生成 Cobertura XML 格式覆蓋率報告"
	@echo "  test-gitlab             - 生成適用於 GitLab CI 的覆蓋率報告"
	@echo "  test-daily              - 生成每日測試報告"
	@echo "  install-test-tools      - 安裝測試所需的工具"
	@echo ""
	@echo "其他目標:"
	@echo "  build                   - 構建應用程序"
	@echo "  run                     - 運行應用程序"
	@echo "  clean                   - 清理構建產物"
	@echo "  dev                     - 使用 Air 進行開發"
	@echo "  install-air             - 安裝 Air 工具"
	@echo "  swagger                 - 生成 Swagger 文檔"
	@echo "  version                 - 顯示當前版本信息"
	@echo "  bump-patch              - 增加補丁版本 (x.y.Z -> x.y.Z+1)"
	@echo "  bump-minor              - 增加次要版本 (x.Y.z -> x.Y+1.0)"
	@echo "  bump-major              - 增加主要版本 (X.y.z -> X+1.0.0)"
	@echo "  help                    - 顯示此幫助"

# 為所有的包創建詳細的測試覆蓋率報告
test-packages:
	@echo "=== 為所有包創建詳細的測試覆蓋率報告 ==="
	@mkdir -p coverage/packages
	@for pkg in $$(go list ./...); do \
		pkg_path=$$(echo $$pkg | sed 's|g38_lottery_service/||'); \
		pkg_name=$$(echo $$pkg | tr / _ | tr . _); \
		echo "測試包: $$pkg"; \
		go test -v -coverprofile=coverage/packages/$$pkg_name.out ./$$pkg_path; \
		go tool cover -html=coverage/packages/$$pkg_name.out -o coverage/packages/$$pkg_name.html; \
		go tool cover -func=coverage/packages/$$pkg_name.out | grep total | awk '{print "  覆蓋率: " $$3}'; \
	done
	@echo "所有包的覆蓋率報告已保存至 coverage/packages/ 目錄"

# 生成測試覆蓋率徽章
test-badge:
	@echo "=== 生成測試覆蓋率徽章 ==="
	@echo "生成內部測試覆蓋率數據..."
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out ./internal/...
	@go tool cover -func=coverage/coverage.out | grep total | awk '{print "總覆蓋率: " $$3}'
	@echo "創建覆蓋率 badge markdown..."
	@coverage=$$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $$3}' | tr -d '%') && \
	echo "# 測試覆蓋率" > COVERAGE.md && \
	echo "" >> COVERAGE.md && \
	echo "![Coverage](https://img.shields.io/badge/Coverage-$${coverage}%25-$(shell if [ "$$(echo "$$coverage < 50" | bc -l)" -eq 1 ]; then echo "red"; elif [ "$$(echo "$$coverage < 80" | bc -l)" -eq 1 ]; then echo "yellow"; else echo "brightgreen"; fi).svg)" >> COVERAGE.md
	@echo "覆蓋率徽章已保存至 COVERAGE.md"

# 用 SVG 可視化測試覆蓋率
test-visualize:
	@echo "=== 生成測試覆蓋率 SVG 可視化 ==="
	@mkdir -p coverage/visual
	@if ! command -v go-cover-treemap > /dev/null 2>&1; then \
		echo "安裝 go-cover-treemap..."; \
		go install github.com/nikolaydubina/go-cover-treemap@latest; \
	fi
	@go test -coverprofile=coverage/coverage.out ./internal/...
	@cat coverage/coverage.out | go-cover-treemap > coverage/visual/coverage-tree.svg
	@echo "覆蓋率樹狀圖已保存至 coverage/visual/coverage-tree.svg"

# 生成 HTML 報告
test-html-all:
	@echo "=== 生成所有 HTML 測試報告 ==="
	@mkdir -p coverage
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "總覆蓋率 HTML 報告已保存至 coverage/coverage.html"

# 主要的測試覆蓋率命令（包含最常用的報告）
test-full: clean
	@echo "=== 執行完整測試流程 ==="
	@mkdir -p coverage
	@mkdir -p coverage/visual
	@mkdir -p coverage/packages
	
	@echo "1. 執行基本測試..."
	@go test ./... -v
	
	@echo "2. 生成覆蓋率報告..."
	@go test -coverprofile=coverage/coverage.out ./...
	@go tool cover -func=coverage/coverage.out > coverage/coverage-func.txt
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	
	@echo "3. 生成視覺化報告..."
	@if command -v go-cover-treemap > /dev/null 2>&1; then \
		cat coverage/coverage.out | go-cover-treemap > coverage/visual/coverage-tree.svg; \
		echo "   覆蓋率樹狀圖已保存"; \
	else \
		echo "   未安裝 go-cover-treemap，跳過生成樹狀圖"; \
	fi
	
	@echo "4. 生成測試覆蓋率徽章..."
	@if command -v gopherbadger > /dev/null 2>&1; then \
		gopherbadger -md="COVERAGE.md" -png=false; \
		echo "   覆蓋率徽章已保存"; \
	else \
		echo "   未安裝 gopherbadger，跳過生成徽章"; \
	fi
	
	@echo "5. 生成 JUnit 報告..."
	@if command -v go-junit-report > /dev/null 2>&1; then \
		go test -v ./... 2>&1 | go-junit-report > coverage/report.xml; \
		echo "   JUnit 報告已保存"; \
	else \
		echo "   未安裝 go-junit-report，跳過生成 JUnit 報告"; \
	fi
	
	@echo ""
	@echo "=== 測試完成 ==="
	@echo "總覆蓋率: " $$(cat coverage/coverage-func.txt | grep total | awk '{print $$3}')
	@echo ""
	@echo "報告文件:"
	@echo "- HTML覆蓋率報告: coverage/coverage.html"
	@echo "- 功能覆蓋率報告: coverage/coverage-func.txt"
	@if [ -f coverage/visual/coverage-tree.svg ]; then \
		echo "- 覆蓋率樹狀圖: coverage/visual/coverage-tree.svg"; \
	fi
	@if [ -f coverage/report.xml ]; then \
		echo "- JUnit報告: coverage/report.xml"; \
	fi
	@if [ -f COVERAGE.md ]; then \
		echo "- 覆蓋率徽章: COVERAGE.md"; \
	fi

# 安裝測試所需的工具
install-test-tools:
	@echo "=== 安裝測試工具 ==="
	@go install github.com/jstemmer/go-junit-report/v2@latest
	@go install github.com/nikolaydubina/go-cover-treemap@latest
	@go install github.com/jpoles1/gopherbadger@latest
	@go install github.com/t-yuki/gocover-cobertura@latest
	@go install github.com/wadey/gocovmerge@latest
	@echo "所有測試工具已安裝完成"

# 測試所有處理程序的覆蓋率
test-handlers:
	@mkdir -p coverage
	@echo "=== 測試處理程序覆蓋率 ==="
	@go test -v -coverprofile=coverage/handler.out ./internal/handler/...
	@go tool cover -func=coverage/handler.out | grep total | awk '{print "總覆蓋率: " $$3}'
	@go tool cover -html=coverage/handler.out -o coverage/handler.html
	@echo "處理程序覆蓋率報告已保存至 coverage/handler.html"

# 測試所有服務的覆蓋率
test-services:
	@mkdir -p coverage
	@echo "=== 測試服務覆蓋率 ==="
	@go test -v -coverprofile=coverage/service.out ./internal/service/...
	@go tool cover -func=coverage/service.out | grep total | awk '{print "總覆蓋率: " $$3}'
	@go tool cover -html=coverage/service.out -o coverage/service.html
	@echo "服務覆蓋率報告已保存至 coverage/service.html"

# 生成單一檔案的整合覆蓋率報告
test-merge-coverage:
	@mkdir -p coverage
	@echo "=== 生成整合覆蓋率報告 ==="
	@if ! command -v gocovmerge > /dev/null 2>&1; then \
		echo "安裝 gocovmerge..."; \
		go install github.com/wadey/gocovmerge@latest; \
	fi
	@gocovmerge coverage/*.out > coverage/merged.out
	@go tool cover -func=coverage/merged.out | grep total | awk '{print "總覆蓋率: " $$3}'
	@go tool cover -html=coverage/merged.out -o coverage/merged.html
	@echo "合併的覆蓋率報告已保存至 coverage/merged.html"

# 安裝 gocover-cobertura 工具
install-cobertura:
	@go install github.com/t-yuki/gocover-cobertura@latest
	@echo "gocover-cobertura 已安裝完成"

# 生成 Cobertura XML 格式覆蓋率報告（適用於 GitLab）
test-cobertura: test-coverage
	@mkdir -p coverage
	@echo "=== 生成 Cobertura XML 格式覆蓋率報告 ==="
	@if ! command -v gocover-cobertura > /dev/null 2>&1; then \
		echo "安裝 gocover-cobertura..."; \
		go install github.com/t-yuki/gocover-cobertura@latest; \
	fi
	@gocover-cobertura < coverage/coverage.out > coverage/cobertura.xml
	@echo "Cobertura XML 覆蓋率報告已保存至 coverage/cobertura.xml"

# 為特定目錄生成測試覆蓋率報告
test-package-coverage:
	@mkdir -p coverage/packages
	@if [ -z "$(PKG)" ]; then \
		echo "錯誤: 需要提供 PKG 參數。範例: make test-package-coverage PKG=./internal/handler"; \
		exit 1; \
	fi
	@echo "=== 為 $(PKG) 生成覆蓋率報告 ==="
	@PKG_NAME=$$(echo $(PKG) | tr / _ | tr . _); \
	go test -v -coverprofile=coverage/packages/$$PKG_NAME.out $(PKG) && \
	go tool cover -func=coverage/packages/$$PKG_NAME.out | grep total | awk '{print "總覆蓋率: " $$3}' && \
	go tool cover -html=coverage/packages/$$PKG_NAME.out -o coverage/packages/$$PKG_NAME.html
	@echo "包覆蓋率報告已保存至 coverage/packages/"

# GitLab 覆蓋率報告
test-gitlab: test-coverage test-cobertura
	@echo "=== 生成 GitLab 覆蓋率報告 ==="
	@echo "覆蓋率: $$(go tool cover -func=coverage/coverage.out | grep total | awk '{print $$3}')"
	@echo "GitLab 覆蓋率報告已生成至 coverage/ 目錄"
	@echo ""
	@echo "要與 GitLab CI 整合，請在 .gitlab-ci.yml 中添加以下內容:"
	@echo ""
	@echo "  artifacts:"
	@echo "    reports:"
	@echo "      cobertura: coverage/cobertura.xml"
	@echo "    paths:"
	@echo "      - coverage/"
	@echo ""
	@echo "  coverage: '/total:\\s+\\(statements\\)\\s+(\\d+\\.\\d+\\%)$/'"

# 生成每日測試報告
test-daily:
	@echo "=== 生成每日測試報告 ==="
	@mkdir -p coverage/daily
	@DATE=$$(date +%Y-%m-%d); \
	mkdir -p coverage/daily/$$DATE; \
	go test -coverprofile=coverage/daily/$$DATE/coverage.out ./...; \
	go tool cover -func=coverage/daily/$$DATE/coverage.out > coverage/daily/$$DATE/coverage-func.txt; \
	go tool cover -html=coverage/daily/$$DATE/coverage.out -o coverage/daily/$$DATE/coverage.html; \
	echo "每日測試報告已保存至 coverage/daily/$$DATE/"
	@echo "覆蓋率: $$(cat coverage/daily/$$(date +%Y-%m-%d)/coverage-func.txt | grep total | awk '{print $$3}')" 