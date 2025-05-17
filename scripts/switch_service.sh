#!/bin/bash

# 定義顏色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 顯示選單
echo -e "${YELLOW}請選擇要啟動的服務:${NC}"
echo -e "${GREEN}1)${NC} 彩票服務 (lottery_service)"
echo -e "${GREEN}2)${NC} 主持人服務 (host_service)"
echo -n "請輸入選項 [1-2]: "

# 讀取用戶輸入
read -r choice

# 根據選擇啟動對應服務
case $choice in
  1)
    echo -e "${GREEN}啟動彩票服務...${NC}"
    export AIR_SERVICE=lottery_service
    
    # 複製環境變數檔案
    if [ -f "./cmd/lottery_service/.env" ]; then
      cp ./cmd/lottery_service/.env ./.env
      echo -e "${GREEN}已複製環境變數檔案: ./cmd/lottery_service/.env -> ./.env${NC}"
    else
      echo -e "${YELLOW}警告: 找不到環境變數檔案 ./cmd/lottery_service/.env${NC}"
    fi
    ;;
  2)
    echo -e "${GREEN}啟動主持人服務...${NC}"
    export AIR_SERVICE=host_service
    
    # 複製環境變數檔案
    if [ -f "./cmd/host_service/.env" ]; then
      cp ./cmd/host_service/.env ./.env
      echo -e "${GREEN}已複製環境變數檔案: ./cmd/host_service/.env -> ./.env${NC}"
    else
      echo -e "${YELLOW}警告: 找不到環境變數檔案 ./cmd/host_service/.env${NC}"
    fi
    ;;
  *)
    echo -e "${YELLOW}無效選項，默認啟動彩票服務${NC}"
    export AIR_SERVICE=lottery_service
    
    # 複製環境變數檔案
    if [ -f "./cmd/lottery_service/.env" ]; then
      cp ./cmd/lottery_service/.env ./.env
      echo -e "${GREEN}已複製環境變數檔案: ./cmd/lottery_service/.env -> ./.env${NC}"
    else
      echo -e "${YELLOW}警告: 找不到環境變數檔案 ./cmd/lottery_service/.env${NC}"
    fi
    ;;
esac

# 顯示啟動資訊
echo -e "${GREEN}正在使用 air 啟動 ${AIR_SERVICE}...${NC}"

# 啟動 air
air 