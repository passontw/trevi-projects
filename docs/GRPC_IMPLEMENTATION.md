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

## 生成的代碼
生成的Go代碼位於 `internal/proto/generated/dealer/` 目錄下：

已成功生成以下文件：
- `ball.pb.go`: 包含球相關的消息和枚舉的生成代碼
- `common.pb.go`: 包含共享定義的生成代碼
- `events.pb.go`: 包含事件相關消息的生成代碼
- `game.pb.go`: 包含遊戲相關消息的生成代碼
- `service.pb.go`: 包含服務定義的生成代碼
- `service_grpc.pb.go`: 包含gRPC服務客戶端和服務器接口的生成代碼

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

# gRPC 實現教學

本文檔說明如何在專案中使用和生成 gRPC 相關代碼。

## 環境設置

確保已安裝以下工具：

1. Protocol Buffers 編譯器 (`protoc`)
2. Go 插件 (`protoc-gen-go` 和 `protoc-gen-go-grpc`)
3. 驗證規則生成器 (`protoc-gen-validate`)

```bash
# 安裝 Protocol Buffers 編譯器
brew install protobuf

# 安裝 Go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 安裝驗證規則生成器
go install github.com/envoyproxy/protoc-gen-validate@latest
```

## 文件結構

Proto 文件存放在 `internal/proto/dealer/` 目錄中，包括：

- `ball.proto`: 定義球相關的消息結構
- `common.proto`: 定義通用的枚舉和常量
- `events.proto`: 定義事件相關的消息結構
- `game.proto`: 定義遊戲相關的消息結構
- `service.proto`: 定義 gRPC 服務接口

生成的 Go 代碼將存放在 `internal/proto/generated/dealer/` 目錄中。

## 生成 gRPC 代碼

使用以下命令生成 Go 語言的 gRPC 代碼：

```bash
protoc -I=. \
  --go_out=. --go_opt=module=g38_lottery_service \
  --go-grpc_out=. --go-grpc_opt=module=g38_lottery_service \
  internal/proto/dealer/*.proto
```

這條命令會：
1. 設置 Proto 文件的搜索路徑 (-I=.)
2. 生成 Go 代碼並輸出到指定位置 (--go_out=.)
3. 使用 g38_lottery_service 作為模塊名稱 (--go_opt=module=g38_lottery_service)
4. 生成 gRPC 服務代碼 (--go-grpc_out=.)
5. 處理 internal/proto/dealer/ 目錄下的所有 .proto 文件

## 注意事項

1. 修改 Proto 文件後需要重新運行上面的命令來生成更新的代碼
2. 在引入 Proto 文件中的類型時，需要使用 generated 目錄下的包路徑
3. 確保 go_package 選項正確設置為 "g38_lottery_service/internal/proto/generated/dealer"

## 使用驗證規則

如需使用驗證規則 (validate.rules)，需要先安裝 protoc-gen-validate：

```bash
go get github.com/envoyproxy/protoc-gen-validate
```

然後在 Proto 文件中引入：

```protobuf
import "validate/validate.proto";
```

確保 validate.proto 文件在搜索路徑中（可能需要下載到本地）：

```bash
mkdir -p validate
curl -o validate/validate.proto https://raw.githubusercontent.com/envoyproxy/protoc-gen-validate/main/validate/validate.proto
```

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

## 客戶端使用示例

使用生成的客戶端代碼示例：

```go
import (
    "context"
    "google.golang.org/grpc"
    pb "g38_lottery_service/internal/proto/generated/dealer"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    
    client := pb.NewDealerServiceClient(conn)
    
    // 使用客戶端調用方法
    resp, err := client.CreateGame(context.Background(), &pb.CreateGameRequest{
        // 設置請求參數
    })
}
```

## 進一步開發

為了完善 gRPC 實現，您可能需要：

1. 實現服務器端的服務處理程序
2. 添加身份驗證和授權機制
3. 實現事件訂閱系統
4. 添加錯誤處理和日誌記錄
5. 集成到主應用程序中

## 實施結果

gRPC 代碼生成已成功完成。`protoc` 命令成功處理了以下文件：
- internal/proto/dealer/ball.proto
- internal/proto/dealer/common.proto
- internal/proto/dealer/events.proto
- internal/proto/dealer/game.proto
- internal/proto/dealer/service.proto

生成的 Go 代碼現在可以在應用程序中使用。我們已完成的工作包括：

1. 設置正確的 import 路徑 (`g38_lottery_service/internal/proto/generated/dealer`)
2. 修復服務實現中的方法簽名以匹配新生成的 proto 定義
3. 註釋掉在最新 proto 定義中不存在的方法
4. 確保正確處理全部訊息類型
5. 成功解決validate.proto導入問題

應用程序啟動時，gRPC 服務器將在配置的端口（默認 9100）上監聽請求。對於連接問題或啟動失敗，請檢查端口是否被其他應用程序占用。

## 故障排除

如果遇到服務無法啟動或連接問題，請嘗試以下步驟：

1. 檢查端口占用：`lsof -i :9100`
2. 刪除可能存在的臨時文件：`rm -f ./internal/proto/generated/go.mod`
3. 執行 `go mod tidy` 更新依賴
4. 如果服務正在運行，可以使用 `grpcurl` 工具測試連接：
   ```
   grpcurl -plaintext localhost:9100 list dealer.DealerService
   ``` 