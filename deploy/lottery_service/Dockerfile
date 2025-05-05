FROM golang:1.23-alpine AS builder

# 安裝必要的系統依賴
RUN apk add --no-cache gcc musl-dev git

# 設置工作目錄
WORKDIR /app

# 複製 go.mod 和 go.sum 檔案
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製所有源代碼
COPY . .

# 編譯應用程序
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o lottery_service ./cmd/lottery_service/main.go

# 使用輕量級的 Alpine 版本作為最終映像
FROM alpine:latest

# 安裝 CA 證書和時區數據，以支持 HTTPS 和正確的時間
RUN apk --no-cache add ca-certificates tzdata

# 設定工作目錄
WORKDIR /app

# 從構建器階段複製編譯好的二進制文件
COPY --from=builder /app/lottery_service .

# 複製環境配置文件範例 (如果需要的話)
COPY .env.example .env

# 暴露可能需要的端口 (根據應用需求調整)
# gRPC 服務端口
EXPOSE 9100
# WebSocket 端口
EXPOSE 8080

# 設置容器啟動命令
ENTRYPOINT ["/app/lottery_service"] 