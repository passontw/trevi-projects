#!/bin/bash

# 確保腳本在出錯時停止執行
set -e

# 設定環境變數
PROTO_DIR="internal/lottery_service/proto"
MODULE_NAME="g38_lottery_service"

# 創建必要的目錄
mkdir -p ${PROTO_DIR}/generated/dealer

# 清理生成的文件
echo "清理舊的生成文件..."
rm -rf ${PROTO_DIR}/generated/dealer/*.pb.go

echo "使用單一命令一次性處理所有 proto 文件..."
# 一次處理所有 proto 文件而非分開處理，確保依賴關係被正確處理
protoc \
  --proto_path=. \
  --proto_path=${PROTO_DIR} \
  --go_out=. \
  --go_opt=module=${MODULE_NAME} \
  --go-grpc_out=. \
  --go-grpc_opt=module=${MODULE_NAME} \
  ${PROTO_DIR}/dealer/*.proto

echo "Protocol Buffers 生成完成！"
echo "檢查生成的文件數量: $(ls -1 ${PROTO_DIR}/generated/dealer/*.pb.go | wc -l)" 