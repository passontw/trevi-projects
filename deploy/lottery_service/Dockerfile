# syntax=docker/dockerfile:1.4
FROM golang:1.24.0-bookworm AS builder

# 設定環境變數
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off \
    CGO_ENABLED=0 \
    GONOSUMDB=* \
    GOPRIVATE=* \
    # 優化 Go 模組下載設定
    GOCACHE=/go/cache \
    GOMODCACHE=/go/mod \
    # 平行下載以提高速度
    GOMAXPROCS=8

# 安裝必要工具 (合併命令減少層數)
RUN --mount=type=cache,target=/var/cache/apt \
    apt-get update && \
    apt-get install -y --no-install-recommends gcc libc6-dev git bash protobuf-compiler libprotobuf-dev curl && \
    rm -rf /var/lib/apt/lists/*

# 設定工作目錄
WORKDIR /go/src/g38_lottery_service

# 先複製 go.mod 和 go.sum (如果存在)
COPY go.mod go.sum* ./

# 預下載依賴 - 使用 BuildKit 緩存掛載加速下載
RUN --mount=type=cache,target=/go/mod \
    --mount=type=cache,target=/go/cache \
    go mod download -x && \
    # 直接下載關鍵的依賴項，確保它們被正確快取
    go get -v github.com/apache/rocketmq-client-go/v2@latest && \
    go get -v github.com/apache/rocketmq-client-go/v2/primitive && \
    go get -v github.com/apache/rocketmq-client-go/v2/producer && \
    go get -v google.golang.org/grpc/keepalive && \
    go get -v google.golang.org/grpc/reflection

# 複製源代碼
COPY . .

# 安裝工具並生成 protobuf 代碼
RUN --mount=type=cache,target=/go/mod \
    --mount=type=cache,target=/go/cache \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.30.0/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf && \
    chmod +x /usr/local/bin/buf && \
    buf --version && \
    chmod +x scripts/gen-proto-buf.sh && \
    ./scripts/gen-proto-buf.sh

# 構建應用程序 (使用 -ldflags 減小二進制檔案大小)
RUN --mount=type=cache,target=/go/mod \
    --mount=type=cache,target=/go/cache \
    go build -ldflags="-s -w" -o lottery_service ./cmd/lottery_service/main.go

# 最終階段 - 使用更小的基礎映像
FROM alpine:3.19

# 僅安裝必要的運行時依賴 (合併命令減少層數)
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -H -h /app appuser

# 設定工作目錄
WORKDIR /app

# 從構建階段複製編譯好的應用程序和必要配置
COPY --from=builder /go/src/g38_lottery_service/lottery_service .
# 如果有配置文件，可以添加以下行 (視項目結構調整)
# COPY --from=builder /go/src/g38_lottery_service/configs ./configs

# 切換到非 root 用戶
USER appuser

# 暴露端口
EXPOSE 8080
EXPOSE 9100

# 定義健康檢查 (根據實際應用調整)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/health || exit 1

# 運行命令
CMD ["./lottery_service"]