# 構建階段
FROM golang:1.23.8-bookworm AS builder

# 設定環境變數
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off
ENV CGO_ENABLED=0

# 安裝必要工具
RUN apt-get update && apt-get install -y gcc libc6-dev git bash protobuf-compiler libprotobuf-dev

# 設定工作目錄 - 這裡是關鍵，確保使用正確的模組路徑
WORKDIR /go/src/g38_lottery_service

# 複製 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下載依賴
RUN go mod download

# 安裝 protoc-gen-go 和 protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# 複製所有源代碼
COPY . .

# 確保 script 可執行
RUN chmod +x scripts/gen-proto-smart.sh

# 生成 protobuf 代碼
RUN ./scripts/gen-proto-smart.sh

# 驗證生成是否成功
RUN ls -la internal/lottery_service/proto/generated/dealer/
RUN cat internal/lottery_service/proto/generated/dealer/common.pb.go | head -20

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