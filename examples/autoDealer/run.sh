#!/bin/bash

# 定義終端顏色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # 無顏色

# 檢查Go環境
if ! command -v go &> /dev/null; then
    echo -e "${RED}錯誤: 未找到Go。請確保Go已安裝且在PATH中。${NC}"
    exit 1
fi

# 取得Go版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
MAJOR=$(echo $GO_VERSION | cut -d. -f1)
MINOR=$(echo $GO_VERSION | cut -d. -f2)

# 檢查Go版本 >= 1.16
if [[ "$MAJOR" -lt 1 || ("$MAJOR" -eq 1 && "$MINOR" -lt 16) ]]; then
    echo -e "${RED}錯誤: Go版本至少需要1.16。當前版本: $GO_VERSION${NC}"
    exit 1
fi

# 獲取腳本所在的目錄
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# 使用環境變量設置參數，如果未提供，則使用默認值
# 使用方式: SERVER_ADDR=localhost:9090 ROOM_ID=SG02 CONFIG_FILE=custom_config.json ./run.sh
SERVER_ADDR=${SERVER_ADDR:-"localhost:9100"}
ROOM_ID=${ROOM_ID:-"SG01"}
CONFIG_FILE=${CONFIG_FILE:-"config.json"}

echo -e "${GREEN}===== 樂透自動荷官啟動腳本 =====${NC}"
echo -e "${YELLOW}服務器地址: $SERVER_ADDR${NC}"
echo -e "${YELLOW}房間ID: $ROOM_ID${NC}"
echo -e "${YELLOW}配置文件: $CONFIG_FILE${NC}"

# 檢查配置文件是否存在
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${YELLOW}警告: 配置文件 '$CONFIG_FILE' 不存在，將使用預設配置${NC}"
fi

# 編譯程序
echo -e "${GREEN}編譯自動荷官程序...${NC}"
go build -o autoDealerbinary main.go

# 檢查編譯是否成功
if [ $? -ne 0 ]; then
    echo -e "${RED}編譯失敗，請檢查錯誤信息。${NC}"
    exit 1
fi

# 添加執行權限
chmod +x autoDealerbinary

echo -e "${GREEN}編譯成功，正在啟動自動荷官...${NC}"

# 運行程序
SERVER_ADDR=$SERVER_ADDR ROOM_ID=$ROOM_ID CONFIG_FILE=$CONFIG_FILE ./autoDealerbinary

# 運行結束後
echo -e "${GREEN}自動荷官已退出${NC}"

echo -e "${GREEN}完成！${NC}" 