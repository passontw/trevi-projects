# Dealer Client 測試工具

這是一個用於測試 gRPC DealerService 介面的命令行工具，可以用來與彩票服務進行互動，模擬荷官的操作。

## 編譯方法

在專案根目錄下執行以下命令來編譯 dealer_client 工具：

```bash
go build -o dealer_client cmd/dealer_client/main.go
```

編譯完成後，將在項目根目錄生成一個名為 `dealer_client` 的可執行文件。

## 使用方法

```bash
./dealer_client <命令> [參數...]
```

## 可用命令

- `start_new_round` - 開始新局遊戲
- `advance_stage [force]` - 推進遊戲階段，可選參數 `force` 強制推進
- `get_status` - 獲取當前遊戲狀態
- `draw_ball <球號> [last]` - 抽取常規球，可選參數 `last` 表示最後一個球
- `draw_extra_ball <球號> [last]` - 抽取額外球，可選參數 `last` 表示最後一個球
- `draw_jackpot_ball <球號> [last]` - 抽取JP球，可選參數 `last` 表示最後一個球
- `draw_lucky_ball <球號> [last]` - 抽取幸運球，可選參數 `last` 表示最後一個球
- `set_jackpot <true|false>` - 設置是否有JP
- `notify_jackpot_winner <贏家ID>` - 通知JP贏家
- `cancel_game <原因>` - 取消遊戲

## 使用示例

```bash
# 開始新局遊戲
./dealer_client start_new_round

# 抽取常規球
./dealer_client draw_ball 42

# 抽取最後一個常規球
./dealer_client draw_ball 99 last

# 設置有JP
./dealer_client set_jackpot true

# 通知JP贏家
./dealer_client notify_jackpot_winner player123

# 取消遊戲
./dealer_client cancel_game "設備故障"

# 強制推進遊戲階段
./dealer_client advance_stage force

# 查看當前遊戲狀態
./dealer_client get_status
```

## 注意事項

1. 確保彩票服務已啟動並監聽在預設的 gRPC 端口（9100）上
2. 若需要連接到不同的服務端點，請修改 `main.go` 中的連接地址
3. 命令必須按照遊戲流程的順序執行，否則可能會收到錯誤
4. 使用 `get_status` 命令可以隨時查看遊戲的當前狀態

## 疑難排解

如果遇到連接問題，請確認：

1. 彩票服務是否已啟動
2. gRPC 端口是否正確（默認 9100）
3. 是否有防火牆阻止連接
4. 服務日誌中是否有錯誤訊息 