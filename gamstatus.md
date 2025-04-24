# 樂透開獎服務遊戲狀態JSON架構

以下是樂透開獎服務的遊戲狀態JSON架構範例，提供荷官端和遊戲端使用：

```json
{
  "game": {
    "id": "G20240619001",
    "state": "BETTING",
    "startTime": "2024-06-19T08:00:00Z",
    "endTime": null,
    "hasJackpot": false,
    "extraBallCount": 3,
    "timeline": {
      "currentTime": "2024-06-19T08:05:30Z",
      "stateStartTime": "2024-06-19T08:05:00Z",
      "remainingTime": 25,
      "maxTimeout": 60
    }
  },
  "luckyNumbers": [1, 12, 23, 34, 45, 56, 67],
  "drawnBalls": [
    {
      "number": 5,
      "drawnTime": "2024-06-19T08:06:00Z",
      "sequence": 1
    },
    {
      "number": 17,
      "drawnTime": "2024-06-19T08:06:01Z",
      "sequence": 2
    }
  ],
  "extraBalls": [
    {
      "number": 42,
      "drawnTime": "2024-06-19T08:10:00Z",
      "sequence": 1,
      "side": "LEFT"
    }
  ],
  "jackpot": {
    "active": false,
    "gameId": null,
    "amount": 500000,
    "startTime": null,
    "endTime": null,
    "drawnBalls": [],
    "winner": null
  },
  "topPlayers": [
    {
      "userId": "U123456",
      "nickname": "幸運星",
      "winAmount": 25000,
      "betAmount": 5000,
      "cards": 3
    },
    {
      "userId": "U789012",
      "nickname": "好運連連",
      "winAmount": 18500,
      "betAmount": 3500,
      "cards": 2
    },
    {
      "userId": "U345678",
      "nickname": "財神到",
      "winAmount": 12000,
      "betAmount": 2000,
      "cards": 1
    }
  ],
  "totalWinAmount": 75800
}
```

## 欄位說明

### 遊戲基本資訊 (game)
- `id`: 遊戲唯一識別碼
- `state`: 遊戲當前狀態，可能的值包括：
  - `INITIAL`: 初始狀態
  - `READY`: 待機狀態
  - `SHOW_LUCKYNUMS`: 開七個幸運球的狀態
  - `BETTING`: 投注狀態
  - `SHOW_BALLS`: 開獎狀態
  - `CHOOSE_EXTRA_BALL`: 額外球投注狀態
  - `SHOW_EXTRA_BALLS`: 額外球開獎狀態
  - `MG_CONCLUDE`: 結算狀態
  - `JP_READY`: JP待機狀態
  - `JP_SHOW_BALLS`: JP開獎狀態
  - `JP_CONCLUDE`: JP結算狀態
- `startTime`: 遊戲開始時間
- `endTime`: 遊戲結束時間，未結束時為null
- `hasJackpot`: 是否有JP遊戲
- `extraBallCount`: 額外球數量
- `timeline`: 時間相關資訊
  - `currentTime`: 目前時間
  - `stateStartTime`: 當前狀態開始時間
  - `remainingTime`: 當前狀態剩餘秒數
  - `maxTimeout`: 最大超時時間(秒)

### 幸運號碼 (luckyNumbers)
遊戲開始前設定的7個幸運號碼陣列

### 已抽出的球 (drawnBalls)
- `number`: 球號
- `drawnTime`: 抽出時間
- `sequence`: 抽出順序

### 額外球 (extraBalls)
- `number`: 球號
- `drawnTime`: 抽出時間
- `sequence`: 抽出順序
- `side`: 球的位置（LEFT或RIGHT）

### JP遊戲資訊 (jackpot)
- `active`: JP遊戲是否啟用
- `gameId`: JP遊戲ID
- `amount`: JP獎金金額
- `startTime`: JP遊戲開始時間
- `endTime`: JP遊戲結束時間
- `drawnBalls`: JP遊戲中抽出的球
- `winner`: JP獲勝者資訊

### 前三名玩家 (topPlayers)
- `userId`: 用戶ID
- `nickname`: 用戶暱稱
- `winAmount`: 贏取金額
- `betAmount`: 投注金額
- `cards`: 購買的卡片數量

### 總贏錢金額 (totalWinAmount)
所有玩家在當前遊戲中贏取的總金額 