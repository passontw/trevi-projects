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