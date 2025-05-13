# gRPC 實現與 Proto 架構文檔

本文檔整合了 `g38_lottery_service` 彩票服務中的 gRPC 實現及 Proto 架構相關資訊，包括實現摘要、架構說明、遷移計劃和最佳實踐指南。

## 目錄

1. [概述](#概述)
2. [當前 Proto 架構](#當前-proto-架構)
3. [服務實現](#服務實現)
4. [Proto 遷移計劃](#proto-遷移計劃) 
5. [遷移進度](#遷移進度)
6. [Proto 架構最佳實踐](#proto-架構最佳實踐)
7. [開發指南](#開發指南)
8. [客戶端使用示例](#客戶端使用示例)
9. [故障排除](#故障排除)

## 概述

我們實現了一個基於 gRPC 的通信層，允許外部系統與彩票服務進行交互。主要特點包括：

1. **模塊化的 Proto 定義**：按服務、版本和功能組織文件
2. **共享定義機制**：集中管理共享類型，減少重複
3. **標準化的版本控制**：明確的版本策略和向後兼容性保障
4. **適配器設計**：平滑過渡新舊 API 結構
5. **全面的服務實現**：支持遊戲生命週期管理、球類抽取等核心功能

## 當前 Proto 架構

新的 proto 架構如下：

```
/proto
  /api                     # API 定義
    /v1                    # API 版本 1
      /lottery             # 彩票服務
        - service.proto    # 服務定義
        - game.proto       # 遊戲相關消息
        - ball.proto       # 球相關消息
        - events.proto     # 事件相關消息
      /dealer              # 荷官服務
        - service.proto    # 服務定義
        - game.proto       # 遊戲相關消息
        - ball.proto       # 球相關消息 
        - events.proto     # 事件相關消息
      /player              # 玩家通訊服務
        - service.proto    # 服務定義
        - messages.proto   # 消息定義
      /mq                  # 消息隊列相關
        - messages.proto   # 消息定義
  /common                  # 共享定義
    - common.proto         # 共享類型與枚舉
    - error.proto          # 錯誤定義
  /third_party             # 第三方依賴
    /validate
      - validate.proto     # 驗證規則
```

生成的 Go 代碼位於 `internal/generated/` 目錄下，按照相應的包結構組織。

## 服務實現

目前我們已實現了以下主要服務：

### 1. DealerService

負責遊戲流程的核心操作，已完成遷移的功能包括：

- **遊戲生命週期管理**
  - 開始新局 (`StartNewRound`)
  - 取消遊戲 (`CancelGame`)
  - 獲取遊戲狀態 (`GetGameStatus`)

- **球類抽取操作**
  - 抽取常規球 (`DrawBall`)
  - 抽取額外球 (`DrawExtraBall`)
  - 抽取 JP 球 (`DrawJackpotBall`)
  - 抽取幸運球 (`DrawLuckyBall`)

- **事件訂閱**
  - 訂閱遊戲事件 (`SubscribeGameEvents`) - 流式 RPC

### 2. LotteryService

提供彩票相關的核心功能，同樣已完成遷移：

- 開始新遊戲回合
- 抽取各類型的球
- 遊戲狀態管理
- 頭獎相關操作

### 3. PlayerCommunicationService

用於玩家通訊的服務，已完成遷移：

- 事件訂閱系統
- 玩家信息管理
- 遊戲歷史記錄

## Proto 遷移計劃

由於原有架構存在一些限制，我們制定了分階段的遷移計劃：

### 階段一：設置新架構（已完成）

1. 建立新的目錄結構
2. 定義共享類型和枚舉
3. 重構服務定義
4. 添加詳細文檔註釋

### 階段二：逐步實施（進行中，約 90% 完成）

1. 使用新版 proto 開發服務
2. 通過適配器連接新舊系統
3. 逐步遷移各服務功能

### 階段三：切換和穩定（計劃中）

1. 完成所有關鍵服務的遷移
2. 執行全面的集成和系統測試
3. 在測試環境中完全切換到新的 proto 定義

### 階段四：清理（計劃中）

1. 移除適配層代碼
2. 廢棄舊的 proto 定義
3. 更新所有文檔和開發指南

## 遷移進度

目前遷移工作進展順利，已完成約 90% 的計劃內容：

### 已完成的工作

1. **所有核心服務的基本遷移**
   - LotteryService 完全遷移
   - DealerService 完全遷移
   - PlayerCommunicationService 完全遷移

2. **適配器實現**
   - 開發了完整的適配器層，支持新舊 API 結構間的轉換
   - 解決了 Ball、GameData、GameStage 等關鍵結構的映射問題
   - 實現了事件系統的兼容處理

3. **重要功能修復與優化**
   - 修復了 CancelGame 功能的返回值處理
   - 優化了 StartJackpotRound 方法的參數處理
   - 增強了事件訂閱系統 (SubscribeGameEvents)
   - 實現了 MQ 消息適配器

4. **基礎設施改進**
   - 標準化了 Proto 引用路徑
   - 完善了錯誤處理機制
   - 添加了全面的日誌記錄

### 進行中的工作

1. **最終兼容性測試**
   - 全面測試所有 API 功能
   - 檢查邊緣情況處理

2. **文檔更新**
   - 更新開發指南
   - 完善 API 使用文檔

### 遇到並解決的主要問題

1. **字段映射差異**
   - 問題：新舊 API 結構中的字段名稱和類型不一致
   - 解決：開發專用的映射函數，處理所有特殊情況

2. **枚舉值不匹配**
   - 問題：GameStage 等枚舉在不同版本中定義不同
   - 解決：創建詳細的枚舉映射表，確保正確轉換

3. **事件結構變化**
   - 問題：GameEvent 結構差異較大
   - 解決：重新設計事件數據結構的映射方式

4. **消息格式差異**
   - 問題：MQ 消息格式變更
   - 解決：開發 ProducerWrapper 類封裝現有生產者

## Proto 架構最佳實踐

我們在開發過程中總結出以下最佳實踐：

### 組織原則

1. **按服務分層** - 每個服務都有自己的目錄
2. **按版本管理** - 不同版本的 API 放在不同的目錄
3. **共享定義集中** - 所有共享類型都在 common 目錄
4. **明確的依賴** - 清晰表明每個文件的依賴關係
5. **一致的命名** - 遵循統一的命名規範

### 命名規範

- **文件命名**：使用小寫和下劃線，基於功能命名
- **包命名**：使用點號分隔的層次結構 (例如: `api.v1.lottery`)
- **消息命名**：使用 PascalCase (例如: `GameData`)
- **服務命名**：使用 PascalCase，並以 Service 結尾 (例如: `LotteryService`)
- **方法命名**：使用 PascalCase (例如: `StartNewRound`)
- **枚舉命名**：使用 PascalCase (例如: `GameStage`)
- **枚舉值**：使用全大寫和下劃線 (例如: `GAME_STAGE_PREPARATION`)

### Protobuf 引用路徑

在使用 Proto 文件時，我們採用以下標準：

1. **使用模組相對路徑**：相對於 Proto 根目錄，如 `import "api/v1/dealer/ball.proto"`
2. **避免使用短路徑**：即使文件在同一目錄，也使用完整相對路徑
3. **避免使用上級目錄**：不使用 `../` 進行引用
4. **保持引用一致性**：所有 Proto 文件使用相同的引用方式

## 開發指南

### 環境設置

確保已安裝以下工具：

```bash
# 安裝 Protocol Buffers 編譯器
brew install protobuf

# 安裝 Go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 安裝驗證規則生成器
go install github.com/envoyproxy/protoc-gen-validate@latest
```

### 生成 gRPC 代碼

使用以下命令生成 Go 語言的 gRPC 代碼：

```bash
protoc -I=. \
  --go_out=. --go_opt=module=g38_lottery_service \
  --go-grpc_out=. --go-grpc_opt=module=g38_lottery_service \
  proto/api/v1/**/*.proto proto/common/*.proto
```

### 新增服務流程

1. 在 `/proto/api/v1/{service_name}` 下創建新的 proto 文件
2. 定義服務接口和消息類型
3. 在 `/proto/common` 中添加任何需要共享的類型
4. 生成代碼
5. 實現服務邏輯和適配器

### 適配器開發指南

開發適配器時，需遵循以下原則：

1. **字段完整映射**：確保所有字段都有對應的映射
2. **預設值處理**：為可能缺失的字段提供合理預設值
3. **錯誤處理**：實現全面的錯誤處理和日誌記錄
4. **單元測試**：為適配器編寫全面的單元測試

## 客戶端使用示例

使用生成的客戶端代碼示例：

```go
import (
    "context"
    "google.golang.org/grpc"
    dealerpb "g38_lottery_service/internal/generated/api/v1/dealer"
)

func main() {
    conn, err := grpc.Dial("localhost:9100", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("連接失敗: %v", err)
    }
    defer conn.Close()
    
    client := dealerpb.NewDealerServiceClient(conn)
    
    // 使用客戶端調用方法
    resp, err := client.StartNewRound(context.Background(), &dealerpb.StartNewRoundRequest{
        RoomId: "SG01",
    })
}
```

## 故障排除

如果遇到服務無法啟動或連接問題，請嘗試以下步驟：

1. **檢查端口占用**：`lsof -i :9100`
2. **刪除可能存在的臨時文件**：`rm -f ./internal/proto/generated/go.mod`
3. **更新依賴**：執行 `go mod tidy` 
4. **測試連接**：使用 `grpcurl` 工具測試連接：
   ```
   grpcurl -plaintext localhost:9100 list dealer.DealerService
   ```
5. **查看引用路徑**：確保所有 import 路徑正確

### 常見問題

1. **"... is not defined" 錯誤**
   - 檢查引用路徑是否正確
   - 確保所有依賴的 proto 文件都被正確包含

2. **適配器字段映射問題**
   - 檢查新舊 API 結構是否有差異
   - 確保所有新字段都有適當的默認值或映射

3. **事件訂閱問題**
   - 檢查事件通道是否正確初始化
   - 確保心跳機制正常工作 