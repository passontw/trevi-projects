# 構建階段
FROM golang:1.23-alpine AS builder

# 設定環境變數
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off
ENV CGO_ENABLED=0

# 安裝必要工具
RUN apk add --no-cache gcc musl-dev git

# 設定工作目錄 - 這裡是關鍵，確保使用正確的模組路徑
WORKDIR /go/src/g38_lottery_service

# 複製 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 複製所有源代碼
COPY . .

# 顯示當前目錄結構，用於調試
RUN ls -la

# 構建應用程序
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o lottery_service ./cmd/lottery_service/main.go

# 最終階段
FROM alpine:latest

# 安裝必要的運行時依賴
RUN apk --no-cache add ca-certificates tzdata

# 設定工作目錄
WORKDIR /app

# 從構建階段複製編譯好的應用程序
COPY --from=builder /go/src/g38_lottery_service/lottery_service .

# 暴露端口（根據您的應用需求調整）
EXPOSE 8080

# 運行命令
CMD ["./lottery_service"]