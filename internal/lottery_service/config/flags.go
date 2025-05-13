package config

import (
	"flag"
	"log"
	"os"
)

// CommandLineArgs 存儲從命令行解析的參數
type CommandLineArgs struct {
	// Nacos 相關配置
	NacosHost        string
	NacosPort        string
	NacosNamespace   string
	NacosGroup       string
	NacosUsername    string
	NacosPassword    string
	NacosDataId      string
	NacosRedisDataId string
	NacosTidbDataId  string
	EnableNacos      bool

	// 服務設定
	ServiceName string
	ServicePort string
	ServerMode  string
	LogLevel    string

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
	flag.StringVar(&Args.NacosHost, "nacos_host", getEnv("NACOS_HOST", "10.1.7.31"), "Nacos server host")
	flag.StringVar(&Args.NacosPort, "nacos_port", getEnv("NACOS_PORT", "8848"), "Nacos server port")
	flag.StringVar(&Args.NacosNamespace, "nacos_namespace", getEnv("NACOS_NAMESPACE", "g38_develop_game_service"), "Nacos namespace")
	flag.StringVar(&Args.NacosGroup, "nacos_group", getEnv("NACOS_GROUP", "DEFAULT_GROUP"), "Nacos group")
	flag.StringVar(&Args.NacosUsername, "nacos_username", getEnv("NACOS_USERNAME", "nacos"), "Nacos username")
	flag.StringVar(&Args.NacosPassword, "nacos_password", getEnv("NACOS_PASSWORD", "nacos"), "Nacos password")
	flag.StringVar(&Args.NacosDataId, "nacos_dataid", getEnv("NACOS_DATAID", "g38_lottery"), "Nacos data ID")
	flag.StringVar(&Args.NacosRedisDataId, "nacos_redis_dataid", getEnv("NACOS_REDIS_DATAID", "redisconfig.xml"), "Nacos Redis config data ID")
	flag.StringVar(&Args.NacosTidbDataId, "nacos_tidb_dataid", getEnv("NACOS_TIDB_DATAID", "dbconfig.xml"), "Nacos TiDB config data ID")

	// 服務設定
	flag.StringVar(&Args.ServiceName, "service_name", getEnv("SERVICE_NAME", "g38_lottery_service"), "Service name")
	flag.StringVar(&Args.ServicePort, "service_port", getEnv("SERVICE_PORT", "8080"), "Service port")
	flag.StringVar(&Args.ServerMode, "server_mode", getEnv("SERVER_MODE", "dev"), "Server mode (dev, prod)")
	flag.StringVar(&Args.LogLevel, "log_level", getEnv("LOG_LEVEL", "debug"), "Log level")

	// 布爾型參數
	enableNacos := getEnvAsBool("ENABLE_NACOS", true)
	flag.BoolVar(&Args.EnableNacos, "enable_nacos", enableNacos, "Enable Nacos configuration")

	// 解析命令行參數
	flag.Parse()

	// 將解析後的命令行參數設置到環境變量中
	setEnvironmentVariables()

	// 標記為已解析
	Args.parsed = true

	// 輸出解析後的參數（僅在開發模式下）
	if Args.ServerMode == "dev" {
		log.Println("已解析的命令行參數:")
		log.Printf("  Nacos 主機: %s", Args.NacosHost)
		log.Printf("  Nacos 端口: %s", Args.NacosPort)
		log.Printf("  Nacos 命名空間: %s", Args.NacosNamespace)
		log.Printf("  Nacos 組: %s", Args.NacosGroup)
		log.Printf("  Nacos 用戶名: %s", Args.NacosUsername)
		log.Printf("  Nacos 數據ID: %s", Args.NacosDataId)
		log.Printf("  Nacos Redis 數據ID: %s", Args.NacosRedisDataId)
		log.Printf("  Nacos TiDB 數據ID: %s", Args.NacosTidbDataId)
		log.Printf("  是否啟用 Nacos: %v", Args.EnableNacos)
		log.Printf("  服務名稱: %s", Args.ServiceName)
		log.Printf("  服務端口: %s", Args.ServicePort)
		log.Printf("  服務器模式: %s", Args.ServerMode)
		log.Printf("  日誌級別: %s", Args.LogLevel)
	}
}

// setEnvironmentVariables 將解析後的命令行參數設置到環境變量中
func setEnvironmentVariables() {
	// Nacos 相關配置
	os.Setenv("NACOS_HOST", Args.NacosHost)
	os.Setenv("NACOS_PORT", Args.NacosPort)
	os.Setenv("NACOS_NAMESPACE", Args.NacosNamespace)
	os.Setenv("NACOS_GROUP", Args.NacosGroup)
	os.Setenv("NACOS_USERNAME", Args.NacosUsername)
	os.Setenv("NACOS_PASSWORD", Args.NacosPassword)
	os.Setenv("NACOS_DATAID", Args.NacosDataId)
	os.Setenv("NACOS_REDIS_DATAID", Args.NacosRedisDataId)
	os.Setenv("NACOS_TIDB_DATAID", Args.NacosTidbDataId)
	os.Setenv("ENABLE_NACOS", formatBool(Args.EnableNacos))

	// 服務設定
	os.Setenv("SERVICE_NAME", Args.ServiceName)
	os.Setenv("SERVICE_PORT", Args.ServicePort)
	os.Setenv("SERVER_MODE", Args.ServerMode)
	os.Setenv("LOG_LEVEL", Args.LogLevel)
}

// formatBool 將布爾值轉換為字符串
func formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// 注意：這個文件使用了 config.go 中定義的 getEnv 和 getEnvAsBool 函數
