# 樂透遊戲自動荷官

這個程式會自動為樂透遊戲服務進行抽球操作，遵循完整遊戲流程，包括常規球、額外球、JP球和幸運球的抽取。

## 功能特點

- 自動連接到遊戲服務器並訂閱遊戲事件
- 自動處理遊戲階段轉換
- 自動抽取常規球、額外球、JP球和幸運球
- 支援自定義球數量和範圍
- 支援透過配置文件調整遊戲參數
- 支援透過環境變數設定服務器地址和房間ID
- 提供優雅的關閉機制（通過Ctrl+C中斷）

## 系統要求

- Go 1.16 或更高版本
- 連接到運行中的樂透遊戲服務

## 使用方法

### Linux/macOS

```bash
# 使用默認配置運行
./run.sh

# 使用自定義參數運行
SERVER_ADDR=localhost:9090 ROOM_ID=SG02 CONFIG_FILE=custom_config.json ./run.sh
```

### Windows

```cmd
:: 使用默認配置運行
run.bat

:: 使用自定義參數運行（在運行前設置環境變數）
set SERVER_ADDR=localhost:9090
set ROOM_ID=SG02
set CONFIG_FILE=custom_config.json
run.bat
```

## 配置文件

自動荷官支援透過JSON格式的配置文件自定義參數：

```json
{
  "game": {
    "regular_balls": {
      "count": 30,
      "max_value": 80
    },
    "extra_balls": {
      "count": 3,
      "max_value": 80
    },
    "jackpot_balls": {
      "count": 1,
      "max_value": 80
    },
    "lucky_balls": {
      "count": 7,
      "max_value": 80
    }
  },
  "timing": {
    "regular_ball_interval_ms": 500,
    "extra_ball_interval_ms": 1000,
    "jackpot_ball_interval_ms": 1000,
    "lucky_ball_interval_ms": 700,
    "card_purchase_duration_sec": 5,
    "game_over_wait_sec": 5
  }
}
```

### 配置項目說明

#### 遊戲配置

- `regular_balls.count`: 常規球數量
- `regular_balls.max_value`: 常規球最大值（範圍1-max_value）
- `extra_balls.count`: 額外球數量
- `extra_balls.max_value`: 額外球最大值
- `jackpot_balls.count`: JP球數量
- `jackpot_balls.max_value`: JP球最大值
- `lucky_balls.count`: 幸運球數量
- `lucky_balls.max_value`: 幸運球最大值

#### 時間配置

- `regular_ball_interval_ms`: 常規球抽取間隔（毫秒）
- `extra_ball_interval_ms`: 額外球抽取間隔（毫秒）
- `jackpot_ball_interval_ms`: JP球抽取間隔（毫秒）
- `lucky_ball_interval_ms`: 幸運球抽取間隔（毫秒）
- `card_purchase_duration_sec`: 購買卡片階段持續時間（秒）
- `game_over_wait_sec`: 遊戲結束後等待開始新遊戲的時間（秒）

## 環境變數

- `SERVER_ADDR`: 服務器地址，默認為 `localhost:8080`
- `ROOM_ID`: 房間ID，默認為 `SG01`
- `CONFIG_FILE`: 配置文件路徑，默認為 `config.json`

## 遊戲流程

1. 連接到服務器並訂閱遊戲事件
2. 開始新遊戲（發送StartNewRound請求）
3. 處理卡片購買階段（等待一段時間）
4. 抽取常規球
5. 選擇額外球側邊
6. 抽取額外球
7. 抽取幸運球
8. 抽取JP球
9. 遊戲結束後，等待一段時間，然後開始新遊戲

## 信號處理

程式會監聽`SIGINT`和`SIGTERM`信號（在控制台按下Ctrl+C），在接收到這些信號時優雅地關閉。 