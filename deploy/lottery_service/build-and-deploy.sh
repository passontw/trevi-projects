#!/bin/bash
set -e

# 顯示說明信息
echo "===== 樂透服務 Docker 構建與部署工具 ====="
echo "此腳本將構建並部署樂透服務的 Docker 容器"

# 獲取腳本所在的目錄
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 切換到項目根目錄
cd "$PROJECT_ROOT"

# 構建 Docker 鏡像
echo "正在構建 Docker 鏡像..."
docker build -t lottery-service:latest -f "$SCRIPT_DIR/Dockerfile" .

# 檢查是否要啟動容器
read -p "是否要部署容器？(y/n): " should_deploy

if [ "$should_deploy" = "y" ] || [ "$should_deploy" = "Y" ]; then
    # 切換到部署目錄
    cd "$SCRIPT_DIR"
    
    # 部署容器
    echo "正在部署容器..."
    docker-compose up -d
    
    echo "部署完成！容器已在後台運行"
    echo "可以使用 'docker-compose logs -f' 查看日誌"
else
    echo "Docker 鏡像構建完成，但未部署容器"
    echo "您可以稍後使用 'cd $SCRIPT_DIR && docker-compose up -d' 來部署"
fi

echo "===== 完成 =====" 