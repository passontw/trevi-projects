# 樂透服務 Docker 部署指南

本目錄包含了用於將樂透服務及其依賴服務部署為 Docker 容器的所有必要文件。

## 文件結構

- `Dockerfile`: 用於構建樂透服務的 Docker 映像
- `docker-compose.yml`: 用於配置和啟動容器
- `build-and-deploy.sh`: 便捷腳本，用於構建映像並部署容器
- `rocketmq/`: RocketMQ 相關配置文件目錄
  - `broker.conf`: RocketMQ Broker 配置

## 快速開始

使用提供的腳本進行構建和部署：

```bash
./build-and-deploy.sh
```

此腳本會引導您完成構建和部署過程。

## 手動部署

如果您偏好手動部署，可以按照以下步驟操作：

### 1. 構建 Docker 映像

從專案根目錄執行：

```bash
docker build -t lottery-service:latest -f deploy/lottery_service/Dockerfile .
```

### 2. 啟動容器

使用 docker-compose 啟動服務：

```bash
cd deploy/lottery_service
docker-compose up -d
```

### 3. 查看日誌

```bash
docker-compose logs -f
```

## 包含的服務

本部署包含以下服務：

1. **樂透服務**：主應用服務
   - WebSocket 端口：8080
   - gRPC 端口：9100

2. **Nacos 服務**：服務註冊和配置中心
   - 管理介面：http://localhost:8848/nacos
   - 用戶名：nacos
   - 密碼：nacos

3. **TiDB 服務**：分佈式數據庫系統
   - 端口：4000（MySQL 協議）
   - 管理介面：http://localhost:10080

4. **Redis 服務**：高性能鍵值存儲
   - 端口：6379
   - 密碼：redis123

5. **RocketMQ 服務**：消息中間件
   - Name Server 端口：9876
   - Broker 端口：10909, 10911
   - 管理介面：http://localhost:8180

## 環境變數

您可以通過編輯 `.env` 文件或在 `docker-compose.yml` 中設置以下環境變數：

- `NACOS_ADDR`: Nacos 服務地址（必須使用格式：http://host:port 或 https://host:port）
- `NACOS_HOST`: Nacos 服務主機（已棄用，請使用 NACOS_ADDR）
- `NACOS_PORT`: Nacos 服務端口（已棄用，請使用 NACOS_ADDR）
- `NACOS_NAMESPACE`: Nacos 命名空間
- `NACOS_GROUP`: Nacos 組
- `NACOS_USERNAME`: Nacos 用戶名
- `NACOS_PASSWORD`: Nacos 密碼
- `NACOS_DATAID`: Nacos 數據 ID
- `ENABLE_NACOS`: 是否啟用 Nacos

> **注意**: `NACOS_ADDR` 環境變數必須使用標準格式 `http://host:port` 或 `https://host:port`，其他格式將被拒絕並使用默認值 `http://127.0.0.1:8848`。

## 健康檢查

容器配置了健康檢查，會定期檢查服務的 `/health` 端點。您可以在 `docker-compose.yml` 中根據需要調整健康檢查參數。

## 數據持久化

所有服務的數據都通過 Docker 卷進行持久化存儲，包括：

- Nacos 數據和日誌
- TiDB 數據和日誌
- Redis 數據
- RocketMQ 日誌和存儲數據

## 故障排除

如果遇到問題：

1. 確保 Docker 和 Docker Compose 已安裝並正常運行
2. 檢查容器日誌以獲取錯誤信息：`docker-compose logs -f [服務名]`
3. 確保所有必要的外部服務（如 Nacos、Redis 等）都已啟動並可訪問
4. 檢查網絡連接：`docker network inspect lottery-network`

## 常見任務

### 重新啟動單個服務

```bash
docker-compose restart [服務名]
```

### 查看特定服務的日誌

```bash
docker-compose logs -f [服務名]
```

### 停止所有服務

```bash
docker-compose down
```

### 停止所有服務並刪除數據卷

⚠️ 警告：這將刪除所有持久化數據

```bash
docker-compose down -v
``` 