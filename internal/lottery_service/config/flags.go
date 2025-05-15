package config

import (
	"flag"
	"log"
	"os"
	"strings"
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

// normalizeNacosAddr 驗證 Nacos 地址格式
// 標準格式: http://host:port 或 https://host:port
// 如果格式不符合，將顯示錯誤訊息並使用默認值
func normalizeNacosAddr() {
	addr := Args.NacosAddr

	// 如果地址為空，使用默認地址
	if addr == "" {
		addr = "http://127.0.0.1:8848"
		log.Printf("警告: Nacos 地址為空，使用默認地址: %s", addr)
		Args.NacosAddr = addr
		return
	}

	// 驗證地址格式必須是 http:// 或 https:// 開頭
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		log.Printf("錯誤: Nacos 地址格式不正確: %s，必須使用 http:// 或 https:// 開頭", addr)
		log.Printf("將使用默認地址: http://127.0.0.1:8848")
		Args.NacosAddr = "http://127.0.0.1:8848"
		return
	}

	// 驗證地址必須包含端口
	host, port, _ := GetNacosHostAndPort()
	if host == "" || port == "8848" && !strings.Contains(addr, ":8848") {
		log.Printf("錯誤: Nacos 地址格式不正確: %s，必須指定端口 (例如 http://host:port)", addr)
		log.Printf("將使用默認地址: http://127.0.0.1:8848")
		Args.NacosAddr = "http://127.0.0.1:8848"
		return
	}

	Args.NacosAddr = addr
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

// 注意：這個文件使用了 config.go 中定義的 getEnv 和 getEnvAsBool 函數
