# gRPC 實現摘要

本文檔總結了 `g38_lottery_service` 彩票服務中的 gRPC 實現。

## 概述

我們實現了一個基於 gRPC 的通信層，允許外部系統與彩票服務進行交互。這個實現包括了以下幾個核心部分：

1. Proto 定義：分為五個獨立的 .proto 文件，按功能組織
2. gRPC 服務實現：管理遊戲生命週期、球類抽取等操作
3. 配置管理：支持通過 Nacos 和環境變數配置 gRPC 端口
4. 測試工具：命令行客戶端用於測試 gRPC 功能

## Proto 定義結構

我們將 Proto 定義分為五個獨立文件，以提高可維護性和可讀性：

1. `common.proto` - 共享的枚舉和類型定義
   - `GameStage` - 定義遊戲階段的枚舉
   - `ExtraBallSide` - 定義額外球選邊的枚舉
   - `GameEventType` - 定義遊戲事件類型的枚舉

2. `ball.proto` - 球相關的定義
   - `BallType` - 定義球類型的枚舉
   - `Ball` - 定義球的基本結構
   - 各種球類型的請求和響應消息

3. `events.proto` - 事件系統的定義
   - `GameEvent` - 基本事件結構
   - 各種具體事件消息
   - 事件訂閱的請求消息

4. `game.proto` - 遊戲狀態和操作的定義
   - `GameData` - 包含完整遊戲狀態的結構
   - 開始新局、推進階段等的請求和響應消息

5. `service.proto` - 服務接口定義
   - `DealerService` - 定義所有服務方法
   - 包括單一請求/響應和流式 RPC

## 服務實現

`DealerService` 實現了以下主要功能：

1. **遊戲生命週期管理**
   - 開始新局 (`StartNewRound`)
   - 推進遊戲階段 (`AdvanceStage`)
   - 取消遊戲 (`CancelGame`)

2. **球類抽取操作**
   - 抽取常規球 (`DrawBall`)
   - 抽取額外球 (`DrawExtraBall`)
   - 抽取 JP 球 (`DrawJackpotBall`)
   - 抽取幸運球 (`DrawLuckyBall`)

3. **狀態查詢和設置**
   - 獲取遊戲狀態 (`GetGameStatus`)
   - 設置 JP 狀態 (`SetHasJackpot`)
   - 通知 JP 獲獎者 (`NotifyJackpotWinner`)

4. **事件訂閱**
   - 訂閱遊戲事件 (`SubscribeGameEvents`) - 流式 RPC

此外，服務實現還包括：

- 事件處理函數，響應遊戲狀態的變化
- WebSocket 事件廣播，將變化通知給所有連接的客戶端
- 轉換函數，在內部數據模型和 Proto 之間轉換

## 配置管理

我們擴展了現有的配置系統，增加了對 gRPC 的支持：

1. 添加了 `GrpcPort` 配置項，預設值為 9100
2. 實現了與 Nacos 的整合，保證端口設置不會被覆蓋
3. 提供了環境變數配置 (`GRPC_PORT`)

## 在主應用中的整合

gRPC 服務已整合到主應用中：

1. 在 `fx` 依賴注入框架中註冊了 gRPC 模塊
2. 在主應用啟動時自動啟動 gRPC 服務器
3. 在應用關閉時優雅地關閉 gRPC 連接

## 測試工具

提供了一個全面的命令行客戶端工具 (`tools/dealer_client.go`)，用於測試所有 gRPC 功能：

1. 支持所有 RPC 方法的調用
2. 提供友好的命令行界面
3. 詳細的錯誤處理和輸出格式化
4. 完整的使用說明文檔

## 未來擴展方向

1. 實現更完善的事件訂閱系統，例如使用 Redis Pub/Sub
2. 增加更多的安全機制，例如 TLS 和認證
3. 擴展監控和日誌記錄
4. 實現更多的客戶端工具和語言綁定 