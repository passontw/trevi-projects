# Gin Web 應用框架骨架

這是一個基於 Golang 的 Web 應用框架骨架，採用六邊形架構（Hexagonal Architecture），使用 Gin、GORM、Redis 和 PostgreSQL 技術。

## 架構概述

本項目採用六邊形架構（也稱為端口和適配器架構），主要包含以下層次：

- **Domain**: 核心業務邏輯和實體
- **Service**: 應用服務層，協調領域對象完成業務邏輯
- **Repository**: 數據存取層，提供數據持久化功能
- **Handler**: HTTP 請求處理層，負責接收和回應 HTTP 請求
- **Middleware**: HTTP 中間件，處理橫切關注點如身份驗證、日誌等
- **Config**: 應用配置管理
- **Core**: 核心框架組件

## 技術棧

- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL
- **Cache**: Redis
- **Dependency Injection**: Uber FX
- **文檔**: Swagger
- **Container**: Docker
- **Hot-reload**: Air

## 快速開始

### 先決條件

- Go 1.18+
- Docker & Docker Compose (可選，用於運行 PostgreSQL 和 Redis)
- PostgreSQL
- Redis

### 創建新項目

1. 克隆本倉庫:
   ```bash
   git clone https://github.com/yourusername/gin-skeleton.git
   cd gin-skeleton
   ```

2. 運行設置腳本:
   ```bash
   ./setup.sh
   ```

3. 按提示輸入您的項目名稱 (例如: github.com/username/project)

4. 安裝依賴:
   ```bash
   go mod tidy
   ```

### 本地開發

1. 啟動數據庫和 Redis (如果使用 Docker):
   ```bash
   docker-compose -f deployments/docker/docker-compose.yaml up -d
   ```

2. 運行應用:
   ```bash
   go run ./main.go
   ```

3. 使用 Air 進行熱重載開發:
   ```bash
   # 首次使用，安裝 Air 工具
   make install-air
   
   # 使用 Air 啟動，支持熱重載
   make dev
   ```

4. 生成 Swagger API 文檔:
   ```bash
   # 生成 Swagger 文檔
   make swagger
   
   # 啟動應用後，可以通過瀏覽器訪問 Swagger UI
   # http://localhost:<port>/api-docs/index.html
   ```

## 項目結構

```
project/
├── cmd/                # 命令行工具
├── deployments/        # 部署相關文件 (Docker, Kubernetes等)
├── docs/               # API 文檔
├── internal/           # 內部應用代碼
│   ├── config/         # 配置處理
│   ├── domain/         # 領域模型和業務邏輯
│   ├── handler/        # HTTP 處理器
│   ├── interfaces/     # 接口定義
│   ├── middleware/     # HTTP 中間件
│   ├── model/          # 數據模型
│   ├── repository/     # 數據存取層
│   └── service/        # 應用服務
├── migrations/         # 數據庫遷移文件
├── pkg/                # 可重用的公共庫
├── .air.toml           # Air 配置文件
├── go.mod              # Go 模塊定義
├── go.sum              # Go 依賴校驗
├── main.go             # 主程序入口
└── README.md           # 項目說明
```

## 功能特點

- **六邊形架構**: 清晰的關注點分離，便於測試和維護
- **依賴注入**: 使用 Uber FX 進行依賴管理
- **API 文檔**: 集成 Swagger 自動生成 API 文檔
- **數據庫遷移**: 內置數據庫遷移支持
- **熱重載**: 支持使用 Air 進行開發時的熱重載
- **中間件**: 預配置常用中間件
- **錯誤處理**: 統一的錯誤處理機制
- **配置管理**: 靈活的配置管理

## 自定義

1. 修改 `internal/config` 中的配置文件以適應您的需求
2. 在 `internal/domain` 中添加您的業務邏輯
3. 在 `internal/repository` 中實現數據存取邏輯
4. 在 `internal/handler` 中添加 API 端點

## 貢獻

1. Fork 本倉庫
2. 創建您的特性分支: `git checkout -b my-new-feature`
3. 提交您的更改: `git commit -am 'Add some feature'`
4. 推送到分支: `git push origin my-new-feature`
5. 提交拉取請求

## 許可證

[MIT](LICENSE)

## 測試

本專案採用完善的測試策略，包括單元測試、整合測試和覆蓋率報告。

### 執行測試

1. 執行所有測試：
   ```bash
   make test
   ```

2. 生成測試覆蓋率報告：
   ```bash
   make test-coverage
   ```
   覆蓋率報告將保存在 `coverage/coverage.html` 中。

3. 測試特定包：
   ```bash
   make test-package-coverage PKG=./internal/service
   ```

4. 執行服務層測試：
   ```bash
   make test-services
   ```

5. 執行處理器層測試：
   ```bash
   make test-handlers
   ```

6. 執行完整測試流程（包括覆蓋率、JUnit 報告等）：
   ```bash
   make test-full
   ```

### 測試結構

每個測試文件應遵循以下結構：

1. 使用 `_test.go` 後綴
2. 為測試建立獨立的測試套件
3. 使用 `testify/suite` 進行組織
4. 使用 `mock` 對外部依賴進行模擬

以下是測試文件的範例結構：

```go
package service_test

import (
	"testing"
	// 引入必要的包
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 定義測試套件
type MyServiceTestSuite struct {
	suite.Suite
	// 測試所需的依賴和被測對象
}

// 測試前設置
func (suite *MyServiceTestSuite) SetupTest() {
	// 初始化測試環境
}

// 具體測試案例
func (suite *MyServiceTestSuite) TestSomeFunction() {
	// 準備測試數據
	// 執行測試
	// 驗證結果
}

// 執行測試套件
func TestMyServiceSuite(t *testing.T) {
	suite.Run(t, new(MyServiceTestSuite))
}
```

## CI/CD 環境與測試流程

本專案支援在 CI/CD 環境中進行測試和部署。以下是在 CI/CD 中從頭到尾建立 Docker 環境並執行單元測試的步驟：

### 1. 準備 Docker 測試環境

在 CI/CD 流程中，我們使用專門的 Docker 容器來進行測試，確保測試環境的一致性和隔離性。

#### 測試環境 Docker Compose 配置

在 `internal/testing/docker-compose-test.yaml` 中配置測試所需的依賴服務，如數據庫和 Redis：

```yaml
version: '3.8'

services:
  tidb:
    image: pingcap/tidb:latest
    ports:
      - "4000:4000"
    environment:
      - MYSQL_ROOT_PASSWORD=
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 10s
      retries: 5

  redis:
    image: redis:latest
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 10s
      retries: 5
```

### 2. CI/CD 測試流程

以下是在 CI/CD 流水線中執行測試的步驟：

1. **建立測試環境**：

   ```bash
   # 啟動測試用的 Docker 容器
   docker-compose -f internal/testing/docker-compose-test.yaml up -d
   
   # 等待服務就緒
   docker-compose -f internal/testing/docker-compose-test.yaml healthcheck
   ```

2. **初始化測試數據庫**：

   ```bash
   # 執行測試數據庫初始化腳本
   mysql -h localhost -P 4000 -u root < internal/testing/init_test_db.sql
   ```

3. **執行測試並生成報告**：

   ```bash
   # 執行測試並生成覆蓋率報告
   make test-full
   
   # 或僅運行測試
   make test
   ```

4. **處理測試報告**：

   ```bash
   # 生成 CI 相容的 Cobertura 格式報告
   make test-cobertura
   
   # 生成 GitLab CI 覆蓋率報告
   make test-gitlab
   ```

5. **清理測試環境**：

   ```bash
   # 停止並移除測試容器
   docker-compose -f internal/testing/docker-compose-test.yaml down -v
   ```

### 3. GitLab CI 配置範例

在 `.gitlab-ci.yml` 中配置自動化測試流程：

```yaml
stages:
  - test
  - build
  - deploy

test:
  stage: test
  image: golang:latest
  services:
    - name: pingcap/tidb:latest
      alias: tidb
    - name: redis:latest
      alias: redis
  variables:
    MYSQL_ROOT_PASSWORD: ""
  before_script:
    - go mod download
    - apt-get update && apt-get install -y mysql-client
    - mysql -h tidb -P 4000 -u root < internal/testing/init_test_db.sql
  script:
    - make test-coverage
    - make test-cobertura
  artifacts:
    reports:
      cobertura: coverage/cobertura.xml
    paths:
      - coverage/
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+\%)$/'
```

### 4. GitHub Actions 配置範例

在 `.github/workflows/test.yml` 中配置自動化測試流程：

```yaml
name: Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      tidb:
        image: pingcap/tidb:latest
        ports:
          - 4000:4000
      redis:
        image: redis:latest
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.20'

    - name: Install dependencies
      run: go mod download

    - name: Initialize test database
      run: |
        sudo apt-get update
        sudo apt-get install -y mysql-client
        mysql -h localhost -P 4000 -u root < internal/testing/init_test_db.sql

    - name: Run tests
      run: make test-full

    - name: Upload test coverage
      uses: actions/upload-artifact@v3
      with:
        name: coverage-report
        path: coverage/
```

## 測試最佳實踐

1. **模擬外部依賴**：使用 mock 對象模擬數據庫、Redis 等外部依賴。
2. **遵循 AAA 模式**：測試程序遵循 Arrange（準備）、Act（執行）、Assert（驗證）模式。
3. **測試覆蓋率**：定期監控測試覆蓋率，目標保持在 70% 以上。
4. **獨立性**：確保測試之間互不影響，可以單獨或以任意順序運行。
5. **速度**：盡量保持單元測試運行速度，避免不必要的 I/O 操作。

參考 Makefile 中的其他測試相關命令以獲取更多測試功能。 