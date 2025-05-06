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

# 定義處理優先順序
declare -a PRIORITY_FILES=(
  "${PROTO_DIR}/dealer/common.proto"
  "${PROTO_DIR}/dealer/ball.proto"
  "${PROTO_DIR}/dealer/game.proto"
  "${PROTO_DIR}/dealer/events.proto"
  "${PROTO_DIR}/dealer/service.proto"
)

# 處理優先文件
for proto_file in "${PRIORITY_FILES[@]}"; do
  if [ -f "$proto_file" ]; then
    echo "編譯 $(basename "$proto_file")..."
    protoc -I=${PROTO_DIR} -I=. \
      --go_out=. \
      --go_opt=module=${MODULE_NAME} \
      --go-grpc_out=. \
      --go-grpc_opt=module=${MODULE_NAME} \
      "$proto_file"
  fi
done

# 處理其他可能未在優先列表中的 proto 文件
echo "搜索其他 proto 文件..."
find ${PROTO_DIR}/dealer -name "*.proto" | while read -r proto_file; do
  # 檢查是否已在優先列表中處理過
  is_priority=false
  for pf in "${PRIORITY_FILES[@]}"; do
    if [ "$pf" = "$proto_file" ]; then
      is_priority=true
      break
    fi
  done
  
  # 如果不是優先文件，則處理它
  if [ "$is_priority" = false ]; then
    echo "編譯額外文件: $(basename "$proto_file")..."
    protoc -I=${PROTO_DIR} -I=. \
      --go_out=. \
      --go_opt=module=${MODULE_NAME} \
      --go-grpc_out=. \
      --go-grpc_opt=module=${MODULE_NAME} \
      "$proto_file"
  fi
done

echo "Protocol Buffers 生成完成！"
echo "檢查生成的文件數量: $(ls -1 ${PROTO_DIR}/generated/dealer/*.pb.go | wc -l)" 