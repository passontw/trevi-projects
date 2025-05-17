package config

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// 配置文件名常量
const (
	DefaultConfigFilename         = "lotterysvr.xml"
	DefaultDbconfigFilename       = "dbconfig.xml"
	DefaultRedisconfigFilename    = "redisconfig.xml"
	DefaultRocketmqconfigFilename = "rocketmq.xml"
)

// CommandLineArgs 存儲從命令行解析的參數
type CommandLineArgs struct {
	// Nacos 相關配置
	NacosAddr           string
	NacosNamespace      string
	NacosGroup          string
	NacosUsername       string
	NacosPassword       string
	NacosDataId         string
	NacosRedisDataId    string
	NacosTidbDataId     string
	NacosRocketMQDataId string
	EnableNacos         bool

	// 服務設定
	ServiceName string
	ServicePort string
	ServerMode  string
	LogLevel    string
	LogDir      string
	LogFormat   string

	// 是否已處理命令行參數
	parsed bool
}

// 全局變量，用於存儲解析後的命令行參數
var Args CommandLineArgs

// InitFlags 初始化並解析命令行參數
func InitFlags() {
	// 如果已經解析過命令行參數，則直接返回
	if Args.parsed {
		return
	}

	// 嘗試先載入 .env 文件
	log.Println("嘗試載入 .env 文件...")

	// 指定可能的 .env 文件位置的優先順序
	envFiles := []string{
		".env",                       // 項目根目錄
		"./cmd/lottery_service/.env", // 彩票服務目錄
		"./cmd/host_service/.env",    // 主持人服務目錄
		fmt.Sprintf("./cmd/%s/.env", os.Getenv("AIR_SERVICE")), // 根據 AIR_SERVICE 環境變量動態選擇
	}

	// 嘗試載入 .env 文件（按優先順序）
	var envLoaded bool
	for _, envFile := range envFiles {
		if _, err := os.Stat(envFile); err == nil {
			if err := godotenv.Load(envFile); err == nil {
				log.Printf("成功載入 .env 文件: %s", envFile)
				// 輸出已載入的重要環境變數
				logEnvVar("NACOS_ADDR")
				logEnvVar("NACOS_NAMESPACE")
				logEnvVar("NACOS_GROUP")
				logEnvVar("NACOS_DATAID")
				envLoaded = true
				break
			}
		}
	}

	if !envLoaded {
		log.Printf("警告: 找不到或無法載入 .env 文件")
		log.Println("將使用命令行參數和環境變數")
	}

	// Nacos 相關配置
	flag.StringVar(&Args.NacosAddr, "nacos_addr", getEnv("NACOS_ADDR", "http://127.0.0.1:8848"), "Nacos server address (format: [http(s)://]host[:port])")
	flag.StringVar(&Args.NacosNamespace, "nacos_namespace", getEnv("NACOS_NAMESPACE", "public"), "Nacos namespace")
	flag.StringVar(&Args.NacosGroup, "nacos_group", getEnv("NACOS_GROUP", "DEFAULT_GROUP"), "Nacos group")
	flag.StringVar(&Args.NacosUsername, "nacos_username", getEnv("NACOS_USERNAME", "nacos"), "Nacos username")
	flag.StringVar(&Args.NacosPassword, "nacos_password", getEnv("NACOS_PASSWORD", "nacos"), "Nacos password")
	flag.StringVar(&Args.NacosDataId, "nacos_dataid", getEnv("NACOS_DATAID", DefaultConfigFilename), "Nacos data ID")
	flag.StringVar(&Args.NacosRedisDataId, "nacos_redis_dataid", getEnv("NACOS_REDIS_DATAID", DefaultRedisconfigFilename), "Nacos Redis config data ID")
	flag.StringVar(&Args.NacosTidbDataId, "nacos_tidb_dataid", getEnv("NACOS_TIDB_DATAID", DefaultDbconfigFilename), "Nacos TiDB config data ID")
	flag.StringVar(&Args.NacosRocketMQDataId, "nacos_rocketmq_dataid", getEnv("NACOS_ROCKETMQ_DATAID", DefaultRocketmqconfigFilename), "Nacos RocketMQ config data ID")

	// 服務設定
	flag.StringVar(&Args.ServiceName, "service_name", getEnv("SERVICE_NAME", "g38_lottery_service"), "Service name")
	flag.StringVar(&Args.ServicePort, "service_port", getEnv("SERVICE_PORT", "8080"), "Service port")
	flag.StringVar(&Args.ServerMode, "server_mode", getEnv("SERVER_MODE", "dev"), "Server mode (dev, prod)")
	flag.StringVar(&Args.LogLevel, "log_level", getEnv("LOG_LEVEL", "debug"), "Log level")
	flag.StringVar(&Args.LogDir, "log_dir", getEnv("LOG_DIR", "logs"), "Log directory")
	flag.StringVar(&Args.LogFormat, "log_format", getEnv("LOG_FORMAT", "hour"), "Log time format: hour or day")

	// 布爾型參數
	enableNacos := getEnvAsBool("ENABLE_NACOS", true)
	flag.BoolVar(&Args.EnableNacos, "enable_nacos", enableNacos, "Enable Nacos configuration")

	// 解析命令行參數
	flag.Parse()

	// 檢查並標準化 NacosAddr 格式
	normalizeNacosAddr()

	// 將解析後的命令行參數設置到環境變量中
	setEnvironmentVariables()

	// 標記為已解析
	Args.parsed = true

	// 輸出解析後的參數（僅在開發模式下）
	if Args.ServerMode == "dev" {
		log.Println("已解析的命令行參數:")
		log.Printf("  Nacos 地址: %s", Args.NacosAddr)
		log.Printf("  Nacos 命名空間: %s", Args.NacosNamespace)
		log.Printf("  Nacos 組: %s", Args.NacosGroup)
		log.Printf("  Nacos 用戶名: %s", Args.NacosUsername)
		log.Printf("  Nacos 數據ID: %s", Args.NacosDataId)
		log.Printf("  Nacos Redis 數據ID: %s", Args.NacosRedisDataId)
		log.Printf("  Nacos TiDB 數據ID: %s", Args.NacosTidbDataId)
		log.Printf("  Nacos RocketMQ 數據ID: %s", Args.NacosRocketMQDataId)
		log.Printf("  是否啟用 Nacos: %v", Args.EnableNacos)
		log.Printf("  服務名稱: %s", Args.ServiceName)
		log.Printf("  服務端口: %s", Args.ServicePort)
		log.Printf("  服務器模式: %s", Args.ServerMode)
		log.Printf("  日誌級別: %s", Args.LogLevel)
		log.Printf("  日誌目錄: %s", Args.LogDir)
		log.Printf("  日誌時間格式: %s", Args.LogFormat)
	}
}

// logEnvVar 輸出環境變數的值（用於調試）
func logEnvVar(name string) {
	value := os.Getenv(name)
	if value != "" {
		log.Printf("環境變數 %s = %s", name, value)
	} else {
		log.Printf("環境變數 %s 未設置", name)
	}
}

// normalizeNacosAddr 驗證並標準化 Nacos 地址格式
// 標準格式: http://host:port 或 https://host:port
// 如果格式不符合，嘗試自動修復或使用默認值
func normalizeNacosAddr() {
	addr := Args.NacosAddr

	// 如果地址為空，使用默認地址
	if addr == "" {
		// 首先從環境變量嘗試獲取完整地址
		envAddr := os.Getenv("NACOS_ADDR")
		if envAddr != "" {
			addr = envAddr
			log.Printf("從環境變量取得 Nacos 地址: %s", addr)
		} else {
			// 嘗試從環境變量獲取主機和端口分別組合
			host := os.Getenv("NACOS_HOST")
			port := os.Getenv("NACOS_PORT")

			if host != "" {
				if port == "" {
					port = "8848" // 默認端口
				}
				addr = fmt.Sprintf("http://%s:%s", host, port)
				log.Printf("從環境變量 NACOS_HOST 和 NACOS_PORT 組合 Nacos 地址: %s", addr)
			} else {
				// 嘗試判斷是否在 Docker 環境中
				if _, err := os.Stat("/.dockerenv"); err == nil {
					// 在 Docker 環境中，默認使用服務名稱 nacos
					addr = "http://nacos:8848"
					log.Printf("檢測到 Docker 環境，使用服務名稱作為主機: %s", addr)
				} else {
					addr = "http://127.0.0.1:8848"
					log.Printf("警告: 未設置任何 Nacos 地址相關環境變量，使用默認地址: %s", addr)
				}
			}
		}
		Args.NacosAddr = addr
	} else {
		log.Printf("使用指定的 Nacos 地址: %s", addr)
	}

	// 檢查地址格式
	// 1. 檢查是否包含協議前綴
	hasPrefix := false
	if strings.HasPrefix(addr, "http://") {
		hasPrefix = true
	} else if strings.HasPrefix(addr, "https://") {
		hasPrefix = true
	}

	if !hasPrefix {
		log.Printf("錯誤: Nacos 地址格式不正確: %s，必須使用 http:// 或 https:// 開頭", addr)

		// 嘗試自動修復：如果包含冒號，假設是 host:port 格式
		if strings.Contains(addr, ":") {
			fixedAddr := "http://" + addr
			log.Printf("嘗試自動修復 Nacos 地址: %s -> %s", addr, fixedAddr)
			Args.NacosAddr = fixedAddr
		} else {
			// 如果是純主機名或 IP，添加協議和默認端口
			fixedAddr := "http://" + addr + ":8848"
			log.Printf("嘗試自動修復 Nacos 地址: %s -> %s", addr, fixedAddr)
			Args.NacosAddr = fixedAddr
		}
		return
	}

	// 2. 驗證地址必須包含主機名部分
	host, port, _ := GetNacosHostAndPort()
	if host == "" {
		log.Printf("錯誤: Nacos 地址格式不正確: %s，主機名不能為空", addr)

		// 嘗試判斷是否在 Docker 環境中
		if _, err := os.Stat("/.dockerenv"); err == nil {
			log.Printf("檢測到 Docker 環境，使用服務名稱作為主機")
			Args.NacosAddr = "http://nacos:8848"
		} else {
			log.Printf("將使用默認地址: http://127.0.0.1:8848")
			Args.NacosAddr = "http://127.0.0.1:8848"
		}
		return
	}

	// 3. 確保地址包含端口
	if !strings.Contains(addr, ":") || strings.Count(addr, ":") == 1 && strings.HasPrefix(addr, "http://") {
		// 地址中沒有端口，添加默認端口
		protocol := "http://"
		if strings.HasPrefix(addr, "https://") {
			protocol = "https://"
		}

		// 提取主機部分
		hostPart := addr
		if strings.HasPrefix(addr, "http://") {
			hostPart = addr[7:]
		} else if strings.HasPrefix(addr, "https://") {
			hostPart = addr[8:]
		}

		// 如果主機部分包含端口，則不需要添加
		if !strings.Contains(hostPart, ":") {
			modifiedAddr := protocol + hostPart + ":8848"
			log.Printf("Nacos 地址未指定端口，添加默認端口: %s -> %s", addr, modifiedAddr)
			Args.NacosAddr = modifiedAddr
		}
	}

	// 輸出最終的 Nacos 連接信息，幫助診斷
	host, port, isHttps := GetNacosHostAndPort()
	protocol := "HTTP"
	if isHttps {
		protocol = "HTTPS"
	}

	log.Printf("最終 Nacos 連接信息: 地址=%s, 服務器=%s, 主機=%s, 端口=%s, 協議=%s",
		Args.NacosAddr, GetNacosServer(), host, port, protocol)
}

// setEnvironmentVariables 將解析後的命令行參數設置到環境變量中
func setEnvironmentVariables() {
	// Nacos 相關配置
	os.Setenv("NACOS_ADDR", Args.NacosAddr)
	os.Setenv("NACOS_NAMESPACE", Args.NacosNamespace)
	os.Setenv("NACOS_GROUP", Args.NacosGroup)
	os.Setenv("NACOS_USERNAME", Args.NacosUsername)
	os.Setenv("NACOS_PASSWORD", Args.NacosPassword)
	os.Setenv("NACOS_DATAID", Args.NacosDataId)
	os.Setenv("NACOS_REDIS_DATAID", Args.NacosRedisDataId)
	os.Setenv("NACOS_TIDB_DATAID", Args.NacosTidbDataId)
	os.Setenv("NACOS_ROCKETMQ_DATAID", Args.NacosRocketMQDataId)
	os.Setenv("ENABLE_NACOS", formatBool(Args.EnableNacos))

	// 設置 NACOS_HOST 和 NACOS_PORT 環境變數以向後相容
	host, port, _ := GetNacosHostAndPort()
	os.Setenv("NACOS_HOST", host)
	os.Setenv("NACOS_PORT", port)

	// 服務設定
	os.Setenv("SERVICE_NAME", Args.ServiceName)
	os.Setenv("SERVICE_PORT", Args.ServicePort)
	os.Setenv("SERVER_MODE", Args.ServerMode)
	os.Setenv("LOG_LEVEL", Args.LogLevel)
	os.Setenv("LOG_DIR", Args.LogDir)
	os.Setenv("LOG_FORMAT", Args.LogFormat)
}

// formatBool 將布爾值轉換為字符串
func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// GetNacosHostAndPort 從 NacosAddr 解析主機和端口
// 返回格式: 主機, 端口, 是否為 HTTPS
func GetNacosHostAndPort() (string, string, bool) {
	addr := Args.NacosAddr
	isHttps := strings.HasPrefix(addr, "https://")

	// 移除協議前綴
	if strings.HasPrefix(addr, "http://") {
		addr = addr[7:]
	} else if strings.HasPrefix(addr, "https://") {
		addr = addr[8:]
	} else {
		// 不符合標準格式，使用默認值
		log.Printf("警告: 從不符合標準格式的地址解析主機和端口: %s", addr)
		return "127.0.0.1", "8848", false
	}

	// 分割主機和端口
	host := addr
	port := "8848" // 預設端口

	// 尋找冒號分隔符
	colonPos := strings.LastIndex(addr, ":")
	if colonPos != -1 {
		host = addr[:colonPos]
		if colonPos+1 < len(addr) {
			port = addr[colonPos+1:]
		}
	} else {
		// 沒有指定端口，使用默認值
		log.Printf("警告: 地址未指定端口: %s，使用默認端口 8848", addr)
	}

	return host, port, isHttps
}

// GetNacosServer 返回適合創建 Nacos 客戶端的地址格式 (host:port)
func GetNacosServer() string {
	host, port, _ := GetNacosHostAndPort()
	return host + ":" + port
}

// CheckNacosConnection 檢查 Nacos 連接是否可用
// 嘗試向 Nacos 服務器發送簡單請求，驗證連接狀態
func CheckNacosConnection() error {
	host, port, _ := GetNacosHostAndPort()
	server := GetNacosServer()
	addr := Args.NacosAddr

	log.Printf("正在檢查 Nacos 連接: %s (服務器: %s, 主機: %s, 端口: %s)",
		addr, server, host, port)

	// 檢查環境變量是否被正確設置
	envAddr := os.Getenv("NACOS_ADDR")
	if envAddr != "" && envAddr != addr {
		log.Printf("警告: 環境變量 NACOS_ADDR (%s) 與當前使用的地址 (%s) 不一致",
			envAddr, addr)
	}

	// 嘗試使用 net.Dial 檢查 TCP 連接是否可用
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", server, timeout)

	if err != nil {
		log.Printf("無法連接到 Nacos 服務器 %s: %v", server, err)

		// 檢查是否為常見的連接錯誤
		if strings.Contains(err.Error(), "connection refused") {
			log.Printf("連接被拒絕，請確認 Nacos 服務器是否啟動")
			log.Printf("檢查事項: ")
			log.Printf(" - Nacos 服務器是否正在運行")
			log.Printf(" - 防火牆是否允許 8848 端口")

			// 檢查是否在 Docker 環境中使用了錯誤的主機名
			if _, dockerErr := os.Stat("/.dockerenv"); dockerErr == nil {
				log.Printf(" - 在 Docker 環境中，主機名應使用服務名稱 'nacos' 而非 IP 地址")
				log.Printf(" - 當前配置的主機名: %s", host)

				// 提供修復建議
				if host == "127.0.0.1" || host == "localhost" {
					log.Printf(" - Docker 環境中建議將 NACOS_ADDR 設置為 http://nacos:8848")
					log.Printf(" - 請修改 .env 文件或使用環境變量設置")
				}
			} else {
				// 在本地環境中，檢查本地 Nacos 服務
				log.Printf(" - 本地環境應確認 Nacos 服務是否已啟動")
				log.Printf(" - 嘗試在瀏覽器中訪問 %s/nacos/ 檢查 Nacos 控制台", addr)
			}
		} else if strings.Contains(err.Error(), "i/o timeout") {
			log.Printf("連接超時，請檢查網絡連接和 Nacos 服務器地址")
			log.Printf("檢查事項: ")
			log.Printf(" - 網絡連接是否正常")
			log.Printf(" - Nacos 服務器地址是否正確")
			log.Printf(" - 防火牆或安全組是否允許連接")
		} else if strings.Contains(err.Error(), "no such host") {
			log.Printf("無法解析主機名 %s，請檢查 DNS 設置或使用 IP 地址", host)
			log.Printf("檢查事項: ")
			log.Printf(" - 主機名是否拼寫正確")
			log.Printf(" - DNS 設置是否正確")
			log.Printf(" - 如在容器環境中，請確保網絡設置允許服務發現")

			// 如果使用 localhost 或 127.0.0.1，提供特殊提示
			if host == "localhost" || host == "127.0.0.1" {
				log.Printf(" - 使用 localhost 或 127.0.0.1 時，只能連接本機的 Nacos")
				log.Printf(" - 如果 Nacos 運行在其他容器或主機上，請使用其實際 IP 或服務名")
			}
		}

		// 嘗試提供環境詳細信息
		log.Printf("環境檢查:")
		isDocker := false
		if _, dockerErr := os.Stat("/.dockerenv"); dockerErr == nil {
			isDocker = true
			log.Printf(" - 檢測到 Docker 容器環境")
		} else {
			log.Printf(" - 檢測到本地運行環境")
		}

		// 提供設置建議
		log.Printf("請嘗試以下解決方法:")
		log.Printf(" 1. 確認 .env 文件中設置了正確的 NACOS_ADDR: %s", envAddr)
		log.Printf(" 2. 使用環境變量設置: export NACOS_ADDR=http://正確的IP或主機名:8848")
		log.Printf(" 3. 檢查 Nacos 服務器是否已啟動並可訪問")

		// 返回詳細的錯誤信息
		return fmt.Errorf("無法連接到 Nacos 服務器 %s: %w (運行環境: %s)",
			server, err,
			map[bool]string{true: "Docker 容器", false: "本機"}[isDocker])
	}

	defer conn.Close()
	log.Printf("成功連接到 Nacos 服務器: %s", server)

	// 進一步檢查 Nacos API 是否可用
	scheme := "http"
	if strings.HasPrefix(addr, "https://") {
		scheme = "https"
	}

	apiURL := fmt.Sprintf("%s://%s/nacos/v1/cs/configs?dataId=test&group=test", scheme, server)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	log.Printf("嘗試訪問 Nacos API: %s", apiURL)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.SetBasicAuth(Args.NacosUsername, Args.NacosPassword)
	resp, apiErr := client.Do(req)

	if apiErr != nil {
		log.Printf("無法訪問 Nacos API: %v", apiErr)
		log.Printf("雖然 TCP 連接成功，但 API 訪問失敗，可能是以下原因:")
		log.Printf(" - Nacos 服務未正常啟動或正在啟動中")
		log.Printf(" - API 路徑不正確，當前使用: %s", apiURL)
		log.Printf(" - 身份驗證問題，檢查用戶名和密碼")

		// 仍然返回 nil，因為 TCP 連接成功
		return nil
	}
	defer resp.Body.Close()

	log.Printf("成功訪問 Nacos API，狀態碼: %d", resp.StatusCode)

	// 檢查是否需要認證
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		log.Printf("Nacos API 需要認證，請檢查 NACOS_USERNAME 和 NACOS_PASSWORD 設置")
	} else if resp.StatusCode == 200 {
		log.Printf("Nacos API 訪問成功，連接正常")
	}

	return nil
}

// 注意：這個文件使用了 config.go 中定義的 getEnv 和 getEnvAsBool 函數
