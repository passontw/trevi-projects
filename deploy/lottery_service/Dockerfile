# syntax=docker/dockerfile:1.4
FROM golang:1.24.1-bookworm AS builder

# 增加代理參數
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY

# 設定環境變數
ENV GO111MODULE=on \
    # 修改 GOPROXY 設置，如果代理失敗直接使用直連模式
    GOPROXY=direct \
    GOSUMDB=off \
    CGO_ENABLED=0 \
    GONOSUMDB=* \
    GOPRIVATE=git.trevi.cc \
    # 強制 Go 使用 HTTP 協議訪問私有倉庫
    GOINSECURE=git.trevi.cc \
    # 優化 Go 模組下載設定
    GOCACHE=/go/cache \
    GOMODCACHE=/go/mod \
    # 平行下載以提高速度
    GOMAXPROCS=8 \
    # 設定代理(如果提供)
    HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY}

# 增加構建參數來接收 Git 相關信息
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
# 增加Git認證參數
ARG GIT_USER
ARG GIT_PASSWORD

# 安裝必要工具 (合併命令減少層數)
RUN --mount=type=cache,target=/var/cache/apt \
    apt-get update && \
    apt-get install -y --no-install-recommends gcc libc6-dev git bash protobuf-compiler libprotobuf-dev curl vim && \
    rm -rf /var/lib/apt/lists/*

# 設定工作目錄
WORKDIR /go/src/g38_lottery_service

# 配置Git以訪問私有倉庫(如果提供了認證參數)
# 僅使用 HTTP 協議
RUN if [ -n "$GIT_USER" ] && [ -n "$GIT_PASSWORD" ]; then \
    echo "配置 Git 認證 (僅使用 HTTP)..." && \
    # 僅使用 HTTP
    git config --global url."http://${GIT_USER}:${GIT_PASSWORD}@git.trevi.cc".insteadOf "http://git.trevi.cc" && \
    git config --global url."http://${GIT_USER}:${GIT_PASSWORD}@git.trevi.cc".insteadOf "https://git.trevi.cc" && \
    git config --global url."http://${GIT_USER}:${GIT_PASSWORD}@git.trevi.cc".insteadOf "git.trevi.cc" && \
    git config --global url."http://${GIT_USER}:${GIT_PASSWORD}@git.trevi.cc/".insteadOf "https://git.trevi.cc/" && \
    git config --global url."http://${GIT_USER}:${GIT_PASSWORD}@git.trevi.cc/".insteadOf "git.trevi.cc/" && \
    # 禁用 SSL 驗證
    git config --global http.sslVerify false && \
    # 設定更高的超時時間以應對可能的網絡延遲
    git config --global http.lowSpeedLimit 1000 && \
    git config --global http.lowSpeedTime 30 && \
    # 顯示 Git 設定以供確認
    echo "Git configuration:" && \
    git config --global --list | grep -v password; \
else \
    echo "警告: 未提供 Git 認證資訊，可能無法訪問私有倉庫"; \
fi

# 為 Go 工具增加特殊設定，確保使用 HTTP 協議
RUN echo "為 Go 工具增加額外設定，確保使用 HTTP 而非 HTTPS..." && \
    go env -w GOINSECURE=git.trevi.cc && \
    echo 'replace git.trevi.cc => http://git.trevi.cc' > /tmp/go.mod.extra && \
    cat /tmp/go.mod.extra

# 先複製 go.mod 和 go.sum (如果存在)
COPY go.mod go.sum* ./

# 顯示 go.mod 內容用於調試
RUN echo "go.mod 內容:" && \
    cat go.mod && \
    echo "檢查是否有 replace 指令:" && \
    grep -n "replace" go.mod || echo "沒有 replace 指令"
# 複製源代碼
COPY . .

# 安裝工具並生成 protobuf 代碼
RUN echo "安裝 protobuf 工具..." && \
    GOPROXY=direct go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
    GOPROXY=direct go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.30.0/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf && \
    chmod +x /usr/local/bin/buf && \
    buf --version && \
    chmod +x scripts/gen-proto-buf.sh && \
    echo "生成 protobuf 代碼..." && \
    ./scripts/gen-proto-buf.sh || echo "警告: Proto 生成可能有問題，但繼續構建..."

# 為了確保私有模組能夠被正確訪問，再次同步依賴
RUN echo "代碼複製後同步依賴..." && \
    GOPROXY=direct go mod tidy || echo "mod tidy 警告，但繼續構建..." && \
    # 顯示模組圖用於調試
    echo "模組依賴關係:" && \
    go mod graph | grep trevi || echo "在依賴關係中找不到 trevi 模組" && \
    # 列出所有依賴項，確認是否有問題
    echo "模組列表:" && \
    go list -m all | grep trevi || echo "在依賴列表中找不到 trevi 模組"

# 構建應用程序 (使用 -ldflags 減小二進制檔案大小)
RUN echo "使用 -mod=mod 構建應用程序，以確保使用已下載的模組..." && \
    GOPROXY=direct go build -v -mod=mod -ldflags="-s -w -X 'main.GitHash=${GIT_COMMIT}' -X 'main.BuildTime=${BUILD_TIME}'" -o lottery_service ./cmd/lottery_service/main.go

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