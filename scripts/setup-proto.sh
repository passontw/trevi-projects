#!/bin/bash

# 確保腳本在出錯時停止執行
set -e

echo "開始設置 Protocol Buffers 環境..."

# 安裝 buf 工具
echo "正在安裝 buf 工具..."
go install github.com/bufbuild/buf/cmd/buf@latest

# 安裝 protoc 插件
echo "正在安裝 protoc-gen-go 和 protoc-gen-go-grpc 插件..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 安裝 protoc 驗證插件
echo "正在安裝 protoc-gen-validate 插件..."
go install github.com/envoyproxy/protoc-gen-validate@latest

echo "環境設置完成！"
echo "現在您可以運行 scripts/gen-proto-buf.sh 來生成 Proto 文件" 