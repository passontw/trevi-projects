# 樂透遊戲文件

## 使用 WebUI

[grpcui](https://github.com/fullstorydev/grpcui)

## 樂透開獎服務遊戲狀態JSON架構

以下是樂透開獎服務的遊戲狀態JSON架構範例，提供荷官端和遊戲端使用：

```json
{
  "game": {
    "id": "game_64adfb18-d0c7-47b8-8f14-26d6ec3466b9",
    "state": "GAME_OVER",
    "startTime": "2025-05-05:00:00Z",
    "endTime": "2025-05-05:15:00Z",
    "hasJackpot": false,
    "extraBallCount": 3
  },
  "luckyNumbers": [1, 12, 23, 34, 45, 56, 67],
  "drawnBalls": [
    { "number": 1, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 2, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 3, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 4, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 5, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 6, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 7, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 8, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 9, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 10, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 11, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 12, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 13, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 14, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 15, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 16, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 17, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 18, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 19, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 20, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 21, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 22, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 23, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 24, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 25, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 26, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 27, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 28, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 29, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 30, "type": "BALL_TYPE_REGULAR", "isLast": true, "timestamp": "2024-06-19T08:00:00Z" }
  ],
  "extraBalls": [
    { "number": 1, "type": "BALL_TYPE_EXTRA", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 2, "type": "BALL_TYPE_EXTRA", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
    { "number": 3, "type": "BALL_TYPE_EXTRA", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" }
  ],
  "jackpot": {
    "active": true,
    "gameId": "game_64adfb18-d0c7-47b8-8f14-26d6ec3466b5",
    "amount": 500000,
    "startTime": "2025-05-05:00:10Z",
    "endTime": "2025-05-05:00:12Z",
    "drawnBalls": [
      { "number": 1, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
      { "number": 2, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
      { "number": 3, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
      { "number": 4, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
      { "number": 5, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
      { "number": 6, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
      { "number": 7, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" }
    ],
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

## gRPC 服務方法

以下是樂透遊戲服務的 gRPC 方法範例：

### DrawBall

抽出一個球。

```json
{
    "balls": [
        { "number": 1, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 2, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 3, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 4, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 5, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 6, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 7, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 8, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 9, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 10, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 11, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 12, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 13, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 14, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 15, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 16, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 17, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 18, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 19, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 20, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 21, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 22, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 23, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 24, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 25, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 26, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 27, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 28, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 29, "type": "BALL_TYPE_REGULAR", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 30, "type": "BALL_TYPE_REGULAR", "isLast": true, "timestamp": "2024-06-19T08:00:00Z" }
    ]
}
```

### CancelGame

取消當前遊戲。

```json
{
  "reason": "test"
}
```

### DrawExtraBall

抽出額外球。

```json
{
    "balls": [
        { "number": 1, "type": "BALL_TYPE_EXTRA", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 2, "type": "BALL_TYPE_EXTRA", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 3, "type": "BALL_TYPE_EXTRA", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" }
    ]
}
```

### DrawJackpotBall

抽出JP獎池球。

```json
{
    "balls": [
        { "number": 1, "type": "BALL_TYPE_JACKPOT", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 2, "type": "BALL_TYPE_JACKPOT", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 3, "type": "BALL_TYPE_JACKPOT", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" }
    ]
}
```

### DrawLuckyBall

抽出幸運球。

```json
{
    "balls": [
        { "number": 1, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 2, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 3, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 4, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 5, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 6, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" },
        { "number": 7, "type": "BALL_TYPE_LUCKY", "isLast": false, "timestamp": "2024-06-19T08:00:00Z" }
    ]
}
```

### GetGameStatus

取得當前遊戲狀態。

```json
{}
```

### StartNewRound

開始新一輪遊戲。

```json
{}
```

