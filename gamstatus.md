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

## 荷官端命令 (Commands)

以下是荷官端發送給服務器的命令格式，按遊戲流程順序排列：

### 1. 開始遊戲 (CmdGameStart)

```json
{
  "type": "GAME_START"
}
```

### 2. 設置幸運號碼 (CmdShowLuckyNumbers)

```json
{
  "type": "SHOW_LUCKY_NUMBERS",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "SHOW_LUCKYNUMS",
      "timeline": {
        "maxTimeout": 15  // 顯示幸運號碼的時間（秒）
      }
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56]
  },
  "timestamp": "2023-10-11T08:45:55.678Z"
}
```

### 3. 抽出球號 (CmdDrawBall)

```json
{
  "type": "DRAW_BALL",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "SHOW_BALLS",
      "timeline": {
        "maxTimeout": 3  // 動畫展示時間（秒）
      }
    },
    "drawnBalls": [
      {
        "number": 42,
        "sequence": 1,
        "drawnTime": "2023-10-11T08:47:25.678Z"
      }
    ]
  },
  "timestamp": "2023-10-11T08:47:25.678Z"
}
```

### 4. 開始額外球階段 (CmdStartExtraBetting)

```json
{
  "type": "START_EXTRA_BETTING",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "CHOOSE_EXTRA_BALL",
      "extraBallCount": 2,
      "timeline": {
        "maxTimeout": 30  // 額外球投注時間（秒）
      }
    }
  },
  "timestamp": "2023-10-11T08:47:55.678Z"
}
```

### 5. 抽出額外球 (CmdDrawExtraBall)

```json
{
  "type": "DRAW_EXTRA_BALL",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "SHOW_EXTRA_BALLS",
      "timeline": {
        "maxTimeout": 3  // 動畫展示時間（秒）
      }
    },
    "extraBalls": [
      {
        "number": 42,
        "drawnTime": "2024-06-19T08:10:00Z",
        "sequence": 1,
        "side": "LEFT"
      }
    ]
  },
  "timestamp": "2024-06-19T08:10:00Z"
}
```

### 6. 開始JP遊戲 (CmdStartJPGame)

```json
{
  "type": "START_JP_GAME",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "JP_READY",
      "hasJackpot": true,
      "timeline": {
        "maxTimeout": 10  // JP準備時間（秒）
      }
    },
    "jackpot": {
      "active": true,
      "gameId": "jp-202406120001",
      "amount": 500000
    }
  },
  "timestamp": "2023-10-11T08:49:05.678Z"
}
```

### 7. 抽出JP球 (CmdDrawJPBall)

```json
{
  "type": "DRAW_JP_BALL",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "JP_SHOW_BALLS",
      "timeline": {
        "maxTimeout": 3  // 動畫展示時間（秒）
      }
    },
    "jackpot": {
      "active": true,
      "gameId": "jp-202406120001",
      "drawnBalls": [
        {
          "number": 33,
          "sequence": 1,
          "drawnTime": "2023-10-11T08:49:15.678Z"
        }
      ]
    }
  },
  "timestamp": "2023-10-11T08:49:15.678Z"
}
```

### 8. 心跳命令 (CmdHeartbeat)

```json
{
  "type": "HEARTBEAT",
  "data": {
    "clientId": "dealer-001"
  },
  "timestamp": "2023-10-11T08:50:15.678Z"
}
```

### 9. 緊急重置 (CmdResetGame)

```json
{
  "type": "RESET_GAME",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "INITIAL"
    },
    "reason": "TECHNICAL_ISSUE"
  },
  "timestamp": "2023-10-11T08:51:15.678Z"
}
```

## 系統事件通知 (Events)

以下是系統發送給荷官端的事件通知，按遊戲流程順序排列：

### 1. 遊戲狀態變更通知 (EventGameStateChanged)

```json
{
  "type": "GAME_STATE_CHANGED",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "READY",
      "previousState": "INITIAL",
      "timeline": {
        "currentTime": "2023-10-11T08:45:45.678Z",
        "stateStartTime": "2023-10-11T08:45:45.678Z",
        "remainingTime": null,
        "maxTimeout": null
      }
    }
  },
  "timestamp": "2023-10-11T08:45:45.678Z"
}
```

### 2. 遊戲準備就緒通知 (EventGameReady)

```json
{
  "type": "GAME_READY",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "READY",
      "startTime": "2023-10-11T08:45:45.678Z",
      "hasJackpot": true,
      "timeline": {
        "currentTime": "2023-10-11T08:45:45.678Z",
        "stateStartTime": "2023-10-11T08:45:45.678Z"
      }
    }
  },
  "timestamp": "2023-10-11T08:45:45.678Z"
}
```

### 3. 幸運號碼設置通知 (EventLuckyNumbersSet)

```json
{
  "type": "LUCKY_NUMBERS_SET",
  "data": {
    "game": {
      "id": "G20240619001",
      "state": "BETTING",
      "startTime": "2024-06-19T08:00:00Z",
      "endTime": null,
      "hasJackpot": false,
      "extraBallCount": null
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56],
    "drawnBalls": [],
    "extraBalls": [],
    "jackpot": {
      "active": false,
      "gameId": null,
      "amount": null,
      "startTime": null,
      "endTime": null,
      "drawnBalls": [],
      "winner": null
    },
    "topPlayers": [],
    "totalWinAmount": null
  },  
  "timestamp": "2023-10-11T08:45:55.678Z"
}
```

### 4. 投注階段開始通知 (EventBettingStarted)

```json
{
  "type": "BETTING_STARTED",
  "data": {
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
        "remainingTime": 60,
        "maxTimeout": 60
      }
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56],
    "drawnBalls": [],
    "extraBalls": [],
    "jackpot": {
      "active": false,
      "gameId": null,
      "amount": null,
      "startTime": null,
      "endTime": null,
      "drawnBalls": [],
      "winner": null
    },
    "topPlayers": [],
    "totalWinAmount": null,
    "bettingInfo": {
      "playerCount": 0,
      "totalBetAmount": 0,
      "startTime": "2024-06-19T08:05:00Z",
      "endTime": "2024-06-19T08:06:00Z"
    }
  },
  "timestamp": "2024-06-19T08:05:00Z"
}
```

### 5. 球號抽出通知 (EventBallDrawn)

```json
{
  "type": "BALL_DRAWN",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "SHOW_BALLS",
      "timeline": {
        "currentTime": "2023-10-11T08:47:25.678Z",
        "stateStartTime": "2023-10-11T08:47:25.678Z",
        "remainingTime": null
      }
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56],
    "drawnBalls": [
      {
        "number": 42,
        "drawnTime": "2023-10-11T08:47:25.678Z",
        "sequence": 1
      }
    ],
    "ballInfo": {
      "remainingBalls": 29,
      "currentBall": {
        "number": 42,
        "sequence": 1,
        "drawnTime": "2023-10-11T08:47:25.678Z"
      }
    }
  },
  "timestamp": "2023-10-11T08:47:25.678Z"
}
```

### 6. 額外球投注階段開始通知 (EventExtraBettingStarted)

```json
{
  "type": "EXTRA_BETTING_STARTED",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "CHOOSE_EXTRA_BALL",
      "extraBallCount": 2,
      "timeline": {
        "currentTime": "2023-10-11T08:47:55.678Z",
        "stateStartTime": "2023-10-11T08:47:55.678Z",
        "remainingTime": 30,
        "maxTimeout": 30
      }
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56],
    "drawnBalls": [
      // 所有已抽出的主要球號
    ],
    "extraBalls": [],
    "jackpot": {
      "active": false,
      "gameId": null,
      "amount": null,
      "startTime": null,
      "endTime": null,
      "drawnBalls": [],
      "winner": null
    },
    "extraBetting": {
      "startTime": "2023-10-11T08:47:55.678Z",
      "endTime": "2023-10-11T08:48:25.678Z",
      "duration": 30,
      "playerCount": 0,
      "totalBetAmount": 0
    }
  },
  "timestamp": "2023-10-11T08:47:55.678Z"
}
```

### 7. 額外球抽出通知 (EventExtraBallDrawn)

```json
{
  "type": "EXTRA_BALL_DRAWN",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "SHOW_EXTRA_BALLS",
      "timeline": {
        "currentTime": "2023-10-11T08:48:35.678Z",
        "stateStartTime": "2023-10-11T08:48:35.678Z",
        "remainingTime": null
      }
    },
    "extraBalls": [
      {
        "number": 18,
        "drawnTime": "2023-10-11T08:48:35.678Z",
        "sequence": 1,
        "side": "LEFT"
      }
    ],
    "extraBallInfo": {
      "remainingExtraBalls": 1,
      "currentBall": {
        "number": 18,
        "drawnTime": "2023-10-11T08:48:35.678Z",
        "sequence": 1,
        "side": "LEFT"
      },
      "totalExtraBalls": 2
    }
  },
  "timestamp": "2023-10-11T08:48:35.678Z"
}
```

### 8. 遊戲結果通知 (EventGameResult)

```json
{
  "type": "GAME_RESULT",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "MG_CONCLUDE",
      "startTime": "2023-10-11T08:45:45.678Z",
      "endTime": "2023-10-11T08:48:55.678Z",
      "hasJackpot": false,
      "extraBallCount": 2,
      "timeline": {
        "currentTime": "2023-10-11T08:48:55.678Z",
        "stateStartTime": "2023-10-11T08:48:55.678Z"
      }
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56],
    "drawnBalls": [
      {"number": 42, "sequence": 1, "drawnTime": "2023-10-11T08:47:25.678Z"},
      {"number": 15, "sequence": 2, "drawnTime": "2023-10-11T08:47:27.678Z"},
      // ... 其他已抽出的主要球號
    ],
    "extraBalls": [
      {"number": 18, "sequence": 1, "side": "LEFT", "drawnTime": "2023-10-11T08:48:35.678Z"},
      {"number": 24, "sequence": 2, "side": "RIGHT", "drawnTime": "2023-10-11T08:48:40.678Z"}
    ],
    "jackpot": {
      "active": false,
      "gameId": null,
      "amount": null,
      "startTime": null,
      "endTime": null,
      "drawnBalls": [],
      "winner": null
    },
    "topPlayers": [
      {"userId": "u-5678", "nickname": "贏家A", "winAmount": 3000, "betAmount": 1000, "cards": 2},
      {"userId": "u-2345", "nickname": "贏家B", "winAmount": 2400, "betAmount": 800, "cards": 1}
    ],
    "totalWinAmount": 19200,
    "gameStats": {
      "totalBets": 256,
      "totalPlayers": 185,
      "totalWinners": 48,
      "totalBetAmount": 25600,
      "winRate": 0.259,  // 贏家比例
      "rtp": 0.75  // Return to Player 比例 (獎金/投注)
    }
  },
  "timestamp": "2023-10-11T08:48:55.678Z"
}
```

### 9. JP遊戲開始通知 (EventJPGameStart)

```json
{
  "type": "JP_GAME_START",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "JP_READY",
      "hasJackpot": true,
      "timeline": {
        "currentTime": "2023-10-11T08:49:05.678Z",
        "stateStartTime": "2023-10-11T08:49:05.678Z"
      }
    },
    "jackpot": {
      "active": true,
      "gameId": "jp-202406120001",
      "amount": 500000,
      "startTime": "2023-10-11T08:49:05.678Z",
      "endTime": null,
      "drawnBalls": [],
      "winner": null,
      "participantCount": 35
    },
    "luckyNumbers": [3, 7, 11, 25, 38, 42, 56],
    "drawnBalls": [
      // 主遊戲已抽出的球號
    ],
    "extraBalls": [
      // 主遊戲已抽出的額外球
    ]
  },
  "timestamp": "2023-10-11T08:49:05.678Z"
}
```

### 10. JP球抽出通知 (EventJPBallDrawn)

```json
{
  "type": "JP_BALL_DRAWN",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "JP_SHOW_BALLS",
      "timeline": {
        "currentTime": "2023-10-11T08:49:15.678Z",
        "stateStartTime": "2023-10-11T08:49:15.678Z"
      }
    },
    "jackpot": {
      "active": true,
      "gameId": "jp-202406120001",
      "amount": 500000,
      "startTime": "2023-10-11T08:49:05.678Z",
      "drawnBalls": [
        {
          "number": 33,
          "sequence": 1,
          "drawnTime": "2023-10-11T08:49:15.678Z"
        }
      ]
    },
    "jpBallInfo": {
      "currentBall": {
        "number": 33,
        "sequence": 1,
        "drawnTime": "2023-10-11T08:49:15.678Z"
      },
      "remainingBalls": 2,
      "totalJPBalls": 3
    }
  },
  "timestamp": "2023-10-11T08:49:15.678Z"
}
```

### 11. JP遊戲結果通知 (EventJPGameResult)

```json
{
  "type": "JP_GAME_RESULT",
  "data": {
    "game": {
      "id": "g-202406120001",
      "state": "JP_CONCLUDE",
      "hasJackpot": true,
      "timeline": {
        "currentTime": "2023-10-11T08:49:35.678Z",
        "stateStartTime": "2023-10-11T08:49:35.678Z"
      }
    },
    "jackpot": {
      "active": true,
      "gameId": "jp-202406120001",
      "amount": 500000,
      "startTime": "2023-10-11T08:49:05.678Z",
      "endTime": "2023-10-11T08:49:35.678Z",
      "drawnBalls": [
        {"number": 33, "sequence": 1, "drawnTime": "2023-10-11T08:49:15.678Z"},
        {"number": 42, "sequence": 2, "drawnTime": "2023-10-11T08:49:25.678Z"},
        {"number": 7, "sequence": 3, "drawnTime": "2023-10-11T08:49:35.678Z"}
      ],
      "winner": {
        "userId": "u-10086",
        "nickname": "幸運兒",
        "cardId": "JP001",
        "amount": 500000
      },
      "participantCount": 35
    },
    "jpStats": {
      "winningPattern": "3 MATCHES",
      "winningTime": "2023-10-11T08:49:35.678Z",
      "totalDuration": 30,
      "participationRate": 0.189  // 參與率 (參與JP人數/總玩家數)
    }
  },
  "timestamp": "2023-10-11T08:49:35.678Z"
}
```

### 12. 連線狀態通知 (EventConnectionStatus)

```json
{
  "type": "CONNECTION_STATUS",
  "data": {
    "status": "STABLE",
    "clientId": "dealer-001",
    "serverTime": "2023-10-11T08:49:45.678Z"
  },
  "timestamp": "2023-10-11T08:49:45.678Z"
}
```

### 13. 錯誤通知 (EventError)

```json
{
  "type": "ERROR",
  "data": {
    "code": "INVALID_COMMAND",
    "message": "無效的指令參數",
    "details": "幸運號碼必須為7個不重複的號碼",
    "requestId": "req-12345",
    "timestamp": "2023-10-11T08:50:05.678Z"
  },
  "timestamp": "2023-10-11T08:50:05.678Z"
}
```

### 14. 心跳回應 (EventHeartbeatResponse)

```json
{
  "type": "HEARTBEAT_RESPONSE",
  "data": {
    "serverTime": "2023-10-11T08:50:15.678Z",
    "status": "ACTIVE"
  },
  "timestamp": "2023-10-11T08:50:15.678Z"
}
```
