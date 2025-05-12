#!/bin/bash

# 確保腳本在出錯時停止執行
set -e

# 檢查 buf 是否已安裝
if ! command -v buf &> /dev/null; then
    echo "錯誤：未找到 buf 工具。請安裝 buf 工具後再繼續。"
    echo "安裝方法：go install github.com/bufbuild/buf/cmd/buf@latest"
    exit 1
fi

# 確保輸出目錄存在
mkdir -p internal/generated

# 清理舊的生成文件
echo "清理舊的生成文件..."
rm -rf internal/generated/*

# 執行 buf 生成
echo "使用 buf 生成 proto 文件..."
(cd proto && buf generate)

echo "Protocol Buffers 生成完成！"
echo "生成的文件位於: internal/generated/"
echo "檢查生成的文件數量: $(find internal/generated -name "*.pb.go" | wc -l)" 