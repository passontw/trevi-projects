# Proto 架構遷移計劃

本文檔描述了如何從當前的 proto 架構遷移到新的架構，同時確保服務的正常運行。

## 現有架構問題

當前的 proto 文件存放結構存在以下限制：

1. 所有 proto 文件都集中在一個服務目錄下，缺乏清晰的組織結構
2. 缺少明確的版本控制策略
3. 沒有共享定義的機制
4. 隨著服務數量增加，會變得難以維護
5. 引用路徑複雜且不一致

## 目標架構

新的 proto 架構如下：

```
/proto
  /api
    /v1
      /lottery
        - service.proto (LotteryService)
        - game.proto
        - ball.proto
        - events.proto
      /payment (未來服務)
        - service.proto (PaymentService)
        - transaction.proto
    /v2 (未來版本)
  /common
    - common.proto (共享類型)
    - error.proto (通用錯誤)
    - pagination.proto (分頁控制)
  /third_party
    /google
      /api
        - annotations.proto
    /validate
      - validate.proto
  /buf.yaml (Buf 配置)
  /buf.gen.yaml (代碼生成配置)
```

## 遷移策略

### 階段一：設置新架構，不改變現有代碼

1. 創建新的目錄結構
2. 將現有 proto 文件複製到新結構，並適當調整
3. 設置 Buf 配置
4. 創建適配器代碼，以支持新舊結構並存

### 階段二：將生成代碼導入測試環境

1. 使用 Buf 生成新的代碼
2. 實現完整的適配器文件
3. 在測試環境中驗證 gRPC 服務的兼容性

### 階段三：逐步遷移服務實現

1. 創建新的服務實現，使用新的 proto 定義
2. 通過適配器將新的實現連接到現有系統
3. 逐步重構現有服務

### 階段四：完全遷移

1. 移除舊的 proto 文件和生成的代碼
2. 移除適配器代碼
3. 更新所有引用

## 實施步驟

### 立即執行（無需重啟服務）

1. 創建新的目錄結構
2. 將現有 proto 文件以重構形式複製到新結構
3. 創建 Buf 配置文件
4. 創建遷移文檔和計劃

### 未來計劃（需要重啟服務）

1. 安裝 Buf 和相關工具
2. 生成新的代碼
3. 實現適配器代碼
4. 開發新版本的服務實現
5. 測試和驗證
6. 部署新版本並完全切換

## 新增服務流程

在新架構下，添加新服務的流程：

1. 在 `/proto/api/v1/{service_name}` 下創建新的 proto 文件
2. 定義服務接口和消息類型
3. 在 `/proto/common` 中添加任何需要共享的類型
4. 使用 Buf 生成代碼
5. 實現服務邏輯

## 版本控制策略

版本控制遵循以下規則：

1. 路徑中包含明確的版本號 (v1, v2 等)
2. 使用語義化版本控制 (Semantic Versioning)
3. 破壞性變更必須創建新版本
4. 向後兼容的變更可以在同一版本中進行 