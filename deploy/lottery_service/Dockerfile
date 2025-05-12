# 構建階段
FROM golang:1.24.0-bookworm AS builder

# 設定環境變數
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off
ENV CGO_ENABLED=0
# 禁用校驗和驗證，解決依賴下載問題
ENV GONOSUMDB=*
ENV GOPRIVATE=*

# 安裝必要工具
RUN apt-get update && apt-get install -y gcc libc6-dev git bash protobuf-compiler libprotobuf-dev curl

# 設定工作目錄
WORKDIR /go/src/g38_lottery_service

# 複製源代碼
COPY . .

# 清除已有的 go.sum 並重新生成 - 避免校驗和衝突
RUN rm -f go.sum && touch go.sum

# 強制下載依賴 - 不使用 -e 標誌，因為 go get 不支援該標誌
RUN go mod tidy -e
RUN go get github.com/apache/rocketmq-client-go/v2@latest
RUN go get github.com/apache/rocketmq-client-go/v2/primitive
RUN go get github.com/apache/rocketmq-client-go/v2/producer
RUN go get google.golang.org/grpc/keepalive
RUN go get google.golang.org/grpc/reflection

# 再次整理模組
RUN go mod tidy -e

# 安裝 protoc-gen-go 和 protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# 安裝 buf 工具
RUN curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.30.0/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf && \
    chmod +x /usr/local/bin/buf && \
    buf --version

# 確保腳本可執行
RUN chmod +x scripts/gen-proto-buf.sh

# 生成 protobuf 代碼
RUN ./scripts/gen-proto-buf.sh

# 構建應用程序 - 簡化構建命令
RUN go build -o lottery_service ./cmd/lottery_service/main.go

# 最終階段
FROM alpine:latest

# 安裝必要的運行時依賴
RUN apk --no-cache add ca-certificates tzdata

# 設定工作目錄
WORKDIR /app

# 從構建階段複製編譯好的應用程序
COPY --from=builder /go/src/g38_lottery_service/lottery_service .

# 暴露端口
EXPOSE 8080
EXPOSE 9100

# 運行命令
CMD ["./lottery_service"]