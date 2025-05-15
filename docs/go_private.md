# 詳細解說 GOPRIVATE 環境變量

## 一、GOPRIVATE 的基本概念

GOPRIVATE 是 Go 模組系統中的一個重要環境變量，它告訴 Go 工具鏈哪些模組路徑是私有的，不應該通過公共 Go 模組代理（如 proxy.golang.org）或校驗和數據庫（如 sum.golang.org）來獲取或驗證。

當您需要從私有儲存庫（如公司內部的 GitLab）獲取 Go 模組時，GOPRIVATE 變量非常重要，因為：

1. **直接下載**：它讓 Go 工具直接從指定的版本控制系統下載模組，而不是通過公共代理
2. **跳過校驗**：它讓 Go 工具跳過對這些模組的公共校驗和驗證
3. **保護隱私**：它確保您的私有代碼不會被發送到公共服務器

## 二、GOPRIVATE 的語法與匹配規則

### 1. 基本語法

GOPRIVATE 變量接受一個逗號分隔的模式列表，每個模式可以匹配一個或多個模組路徑：

```bash
export GOPRIVATE=gitlab.your-company.com,github.com/your-username
```

### 2. 匹配規則

GOPRIVATE 使用 Go 的路徑匹配規則，具體來說：

- **精確匹配**：完全匹配模組路徑
  ```
  gitlab.your-company.com/team/repo
  ```

- **前綴匹配**：使用尾部的星號（*）匹配所有子路徑
  ```
  gitlab.your-company.com/team/*
  ```

- **域名匹配**：僅列出域名將匹配該域名下的所有模組
  ```
  gitlab.your-company.com
  ```

- **多模式匹配**：同時指定多個模式
  ```
  gitlab.your-company.com,bitbucket.org/your-team
  ```

## 三、設置 GOPRIVATE 的方法

### 1. 臨時設置（當前 Shell 會話）

```bash
export GOPRIVATE=gitlab.your-company.com
```

### 2. 永久設置（Shell 配置文件）

在 ~/.bashrc、~/.zshrc 或對應的 Shell 配置文件中添加：

```bash
echo 'export GOPRIVATE=gitlab.your-company.com' >> ~/.bashrc
source ~/.bashrc
```

### 3. 使用 go env 命令設置（推薦）

這種方法會將設置保存在 Go 的環境配置文件中（通常是 ~/.config/go/env）：

```bash
go env -w GOPRIVATE=gitlab.your-company.com
```

### 4. 查看當前設置

```bash
go env GOPRIVATE
```

## 四、GOPRIVATE 如何影響 Go 模組下載

當 GOPRIVATE 設置後，Go 工具鏈的行為會發生以下變化：

### 1. 模組代理行為

- **匹配的模組**：直接從源代碼儲存庫獲取，不使用模組代理
- **不匹配的模組**：仍然使用模組代理（如 GOPROXY 設置的值）

### 2. 校驗和數據庫行為

- **匹配的模組**：不向校驗和數據庫請求或提交校驗和
- **不匹配的模組**：仍然使用校驗和數據庫（如 GOSUMDB 設置的值）

### 3. 版本控制系統交互

- 對於匹配的模組，Go 工具會直接與版本控制系統（如 Git）交互
- 這意味著您需要確保 Git 已正確配置以訪問私有儲存庫

## 五、實際使用示例

### 示例 1：從私有 GitLab 獲取模組

```bash
# 設置 GOPRIVATE
go env -w GOPRIVATE=gitlab.your-company.com

# 在 go.mod 中添加依賴
# require gitlab.your-company.com/team/module v1.0.0

# 獲取依賴
go get gitlab.your-company.com/team/module@v1.0.0
```

### 示例 2：使用多個私有儲存庫

```bash
# 設置多個私有儲存庫
go env -w GOPRIVATE=gitlab.your-company.com,github.com/your-enterprise

# 同時使用這些私有儲存庫中的模組
go get gitlab.your-company.com/team/module1@v1.0.0
go get github.com/your-enterprise/repo/module2@v2.0.0
```

### 示例 3：只將特定組織或團隊設為私有

```bash
# 只將特定團隊的儲存庫設為私有
go env -w GOPRIVATE=gitlab.your-company.com/team-a,gitlab.your-company.com/team-b

# 其他團隊的儲存庫仍會通過公共代理獲取
go get gitlab.your-company.com/team-a/module@v1.0.0  # 私有獲取
go get gitlab.your-company.com/team-c/module@v1.0.0  # 通過代理獲取
```

## 六、與其他 Go 模組環境變量的關係

GOPRIVATE 與其他幾個 Go 環境變量協同工作：

### 1. GOPROXY

定義 Go 模組代理服務器的 URL：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

GOPRIVATE 匹配的模組將忽略 GOPROXY 設置，直接從源下載。

### 2. GONOPROXY

用於指定哪些模組不使用代理，但仍進行校驗和驗證：

```bash
go env -w GONOPROXY=gitlab.your-company.com/public-modules
```

GOPRIVATE 實際上同時設置了 GONOPROXY 和 GONOSUMDB 的值，除非它們被明確設置。

### 3. GOSUMDB

定義用於驗證模組校驗和的數據庫：

```bash
go env -w GOSUMDB=sum.golang.org
```

GOPRIVATE 匹配的模組將忽略 GOSUMDB 設置，不進行校驗和驗證。

### 4. GONOSUMDB

用於指定哪些模組不使用校驗和數據庫：

```bash
go env -w GONOSUMDB=gitlab.your-company.com/experimental
```

### 5. 優先級關係

如果一個模組路徑：
- 匹配 GOPRIVATE → 忽略代理和校驗和數據庫
- 匹配 GONOPROXY → 忽略代理，但仍使用校驗和數據庫
- 匹配 GONOSUMDB → 使用代理，但忽略校驗和數據庫

## 七、GOPRIVATE 的高級使用

### 1. 在 Dockerfile 中設置

```dockerfile
FROM golang:1.20

# 設置環境變量
ENV GOPRIVATE=gitlab.your-company.com

# 複製 Git 憑證 (如需要)
COPY .netrc /root/.netrc
RUN chmod 600 /root/.netrc

# 後續步驟...
```

### 2. 在 CI/CD 中設置

GitLab CI/CD 示例：

```yaml
build:
  image: golang:1.20
  script:
    - go env -w GOPRIVATE=gitlab.your-company.com
    - echo "machine gitlab.your-company.com login ${CI_REGISTRY_USER} password ${CI_REGISTRY_PASSWORD}" > ~/.netrc
    - chmod 600 ~/.netrc
    - go mod tidy
    - go build
```

### 3. 臨時覆蓋 GOPRIVATE 設置

對於單次命令：

```bash
GOPRIVATE=gitlab.your-company.com go get gitlab.your-company.com/team/repo
```

### 4. 使用通配符

```bash
# 所有 .internal.com 域名下的儲存庫都被視為私有
go env -w GOPRIVATE=*.internal.com
```

## 八、常見問題與解決方案

### 1. 身份驗證問題

**問題**：設置了 GOPRIVATE，但 `go get` 仍返回 401 或 403 錯誤。

**解決方案**：GOPRIVATE 只是告訴 Go 直接從儲存庫獲取代碼，您仍然需要配置 Git 憑證：

```bash
# 配置 .netrc 文件
echo "machine gitlab.your-company.com login your-username password your-token" >> ~/.netrc
chmod 600 ~/.netrc

# 或使用 Git 憑證助手
git config --global credential.helper store
git credential approve <<EOF
protocol=https
host=gitlab.your-company.com
username=your-username
password=your-token
EOF

# 或使用 URL 重寫
git config --global url."https://username:token@gitlab.your-company.com/".insteadOf "https://gitlab.your-company.com/"
```

### 2. SSL 證書問題

**問題**：遇到 SSL 證書驗證錯誤。

**解決方案**：如果 GitLab 服務器使用自簽名證書，您可以使用 GOINSECURE 環境變量：

```bash
go env -w GOINSECURE=gitlab.your-company.com
```

或配置 Git 忽略 SSL 驗證（不推薦用於生產環境）：

```bash
git config --global http.sslVerify false
```

### 3. 模塊路徑問題

**問題**：獲取的模組版本與預期不符。

**解決方案**：檢查模組路徑和版本是否正確，尤其是大小寫：

```bash
# 檢查可用的模組版本
GOPRIVATE=gitlab.your-company.com go list -m -versions gitlab.your-company.com/team/module
```

### 4. 清除緩存

如果存在緩存問題，可以清除模組緩存：

```bash
go clean -modcache
```

## 九、最佳實踐

1. **精確匹配**：盡量精確指定私有模組路徑，避免過度匹配

2. **模組版本標記**：在私有儲存庫中使用適當的語義化版本標籤

3. **保存憑證**：確保 Git 憑證安全存儲且正確設置

4. **文檔化**：將 GOPRIVATE 設置記錄在項目文檔中，以便新團隊成員能夠快速配置

5. **自動化設置**：考慮使用腳本自動化這些設置，特別是對於新開發環境

6. **穩定性考慮**：考慮在本地或組織內部運行私有模組代理（如 Athens）以提高可靠性

## 十、進階應用：企業內部 Go 模組管理

對於較大的企業，可以考慮更先進的設置：

1. **內部模組代理**：設置私有 Go 模組代理服務器，如 Athens 或 GoCenter
   
2. **Vanity URLs**：使用 Go 的 Import 路徑重定向功能，讓模組路徑更穩定
   
3. **模組鏡像**：定期將常用公共模組鏡像到內部儲存庫
   
4. **依賴掃描**：集成安全掃描工具以檢查私有模組中的潛在漏洞

## 總結

GOPRIVATE 環境變量是使用私有 Go 模組的關鍵配置，它讓 Go 工具鏈知道哪些模組應該直接從源碼儲存庫獲取，而不通過公共代理和校驗和數據庫。正確設置 GOPRIVATE 變量，配合適當的 Git 憑證配置，可以讓您的 Go 項目無縫使用私有模組，包括來自本地 HTTP GitLab 儲存庫的模組。
