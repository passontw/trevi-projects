#!/bin/bash

# 定義顏色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默認值
DEFAULT_HOST="127.0.0.1"
DEFAULT_PORT="9100"
DEFAULT_BROWSER="true"  # 默認自動打開瀏覽器

# 使用環境變數或默認值
GRPC_HOST=${GRPC_HOST:-$DEFAULT_HOST}
GRPC_PORT=${GRPC_PORT:-$DEFAULT_PORT}
OPEN_BROWSER=${OPEN_BROWSER:-$DEFAULT_BROWSER}
GRPC_WEB_PORT=${GRPC_WEB_PORT:-"0"}  # 0 表示自動選擇可用端口

# 顯示橫幅
echo -e "${BLUE}=================================================${NC}"
echo -e "${BLUE}            gRPC UI 連接工具                     ${NC}"
echo -e "${BLUE}=================================================${NC}"

# 顯示連接資訊
echo -e "${GREEN}正在連接 gRPC 服務...${NC}"
echo -e "${YELLOW}Host: ${GRPC_HOST}${NC}"
echo -e "${YELLOW}Port: ${GRPC_PORT}${NC}"
if [ "$GRPC_WEB_PORT" != "0" ]; then
    echo -e "${YELLOW}Web UI Port: ${GRPC_WEB_PORT}${NC}"
fi
echo -e "${YELLOW}自動打開瀏覽器: $([ "$OPEN_BROWSER" == "true" ] && echo "是" || echo "否")${NC}"

# 檢查 grpcui 是否已安裝
if ! command -v grpcui &> /dev/null; then
    echo -e "${YELLOW}未找到 grpcui 命令，正在安裝...${NC}"
    go install github.com/fullstorydev/grpcui/cmd/grpcui@latest
    
    # 再次檢查是否安裝成功
    if ! command -v grpcui &> /dev/null; then
        echo -e "${RED}grpcui 安裝失敗，請手動安裝：${NC}"
        echo -e "go install github.com/fullstorydev/grpcui/cmd/grpcui@latest"
        exit 1
    else
        echo -e "${GREEN}grpcui 安裝成功！${NC}"
    fi
fi

# 檢查 gRPC 服務是否可連接
echo -e "${YELLOW}正在檢查 gRPC 服務連通性...${NC}"
if ! nc -z -w 2 ${GRPC_HOST} ${GRPC_PORT} &> /dev/null; then
    echo -e "${RED}無法連接到 gRPC 服務 ${GRPC_HOST}:${GRPC_PORT}${NC}"
    echo -e "${YELLOW}請確認服務已啟動並且端口正確${NC}"
    
    # 提示用戶是否仍要繼續
    echo -e "${YELLOW}是否仍要嘗試啟動 gRPC UI？(y/n) ${NC}"
    read -r response
    if [[ "$response" != "y" && "$response" != "Y" ]]; then
        echo -e "${RED}已取消啟動 gRPC UI${NC}"
        exit 1
    fi
    echo -e "${YELLOW}繼續嘗試啟動 gRPC UI...${NC}"
else
    echo -e "${GREEN}gRPC 服務連接正常${NC}"
fi

# 構建命令參數
COMMAND="grpcui -plaintext"

# 添加 Web 端口參數
if [ "$GRPC_WEB_PORT" != "0" ]; then
    COMMAND="$COMMAND -port=${GRPC_WEB_PORT}"
fi

# 添加瀏覽器開啟選項
if [ "$OPEN_BROWSER" != "true" ]; then
    COMMAND="$COMMAND -open=false"
fi

# 添加主機和端口
COMMAND="$COMMAND ${GRPC_HOST}:${GRPC_PORT}"

# 執行 grpcui
echo -e "${GREEN}啟動 gRPC UI...${NC}"
echo -e "${YELLOW}執行命令: ${COMMAND}${NC}"
echo -e "${BLUE}=================================================${NC}"

# 執行命令
eval $COMMAND

# 捕獲退出狀態
EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]; then
    echo -e "${RED}gRPC UI 啟動失敗，退出碼: ${EXIT_CODE}${NC}"
    exit $EXIT_CODE
fi 