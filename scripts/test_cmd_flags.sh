#!/bin/bash

# 設置顏色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== 開始測試命令行參數功能 ===${NC}"

# 確保已編譯最新的二進制文件
echo -e "${GREEN}編譯服務...${NC}"
go build -o ./build/g38_lottery_service ./cmd/lottery_service/main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}編譯失敗!${NC}"
    exit 1
fi

echo -e "${GREEN}編譯成功!${NC}"

# 測試參數指南
echo -e "${BLUE}===== 測試命令行參數使用說明 =====${NC}"
./build/g38_lottery_service --help

# 測試默認配置
echo -e "${BLUE}===== 測試默認配置啟動 (按 Ctrl+C 停止) =====${NC}"
echo -e "${YELLOW}將使用 .env 文件中的默認設定啟動服務${NC}"
timeout 3s ./build/g38_lottery_service || true
echo ""

# 測試單一參數
echo -e "${BLUE}===== 測試單一參數 (按 Ctrl+C 停止) =====${NC}"
echo -e "${YELLOW}使用自定義 Nacos 主機啟動:${NC}"
timeout 3s ./build/g38_lottery_service --nacos_host="10.1.7.31" || true
echo ""

# 測試多個參數
echo -e "${BLUE}===== 測試多個參數 (按 Ctrl+C 停止) =====${NC}"
echo -e "${YELLOW}使用自定義 Nacos 主機和端口:${NC}"
timeout 3s ./build/g38_lottery_service --nacos_host="10.1.7.31" --nacos_port="8848" || true
echo ""

# 測試服務配置
echo -e "${BLUE}===== 測試服務配置 (按 Ctrl+C 停止) =====${NC}"
echo -e "${YELLOW}使用測試模式和調試日誌級別:${NC}"
timeout 3s ./build/g38_lottery_service --server_mode="dev" --log_level="debug" || true
echo ""

# 測試所有參數
echo -e "${BLUE}===== 測試完整參數集 (按 Ctrl+C 停止) =====${NC}"
echo -e "${YELLOW}使用所有可能的參數:${NC}"
timeout 5s ./build/g38_lottery_service \
    --nacos_host="10.1.7.31" \
    --nacos_port="8848" \
    --nacos_namespace="g38_develop_game_service" \
    --nacos_group="DEFAULT_GROUP" \
    --nacos_username="nacos" \
    --nacos_password="nacos" \
    --nacos_dataid="g38_lottery" \
    --enable_nacos=true \
    --service_name="g38_lottery_service" \
    --service_port="8080" \
    --server_mode="dev" \
    --log_level="debug" || true
echo ""

echo -e "${GREEN}測試完成!${NC}"
echo -e "${BLUE}===== 參數使用總結 =====${NC}"
echo -e "${GREEN}命令行參數可以靈活地覆蓋默認設定，特別適合在不同環境中快速部署。${NC}"
echo -e "${GREEN}參數優先級: 命令行參數 > 環境變量 > .env 文件 > 默認值${NC}"
echo -e "${GREEN}使用 --help 隨時查看可用的命令行參數${NC}" 