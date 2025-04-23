package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"g38_lottery_service/internal/config"
)

// PrintVersion 打印版本信息並退出
func PrintVersion() {
	fmt.Println(config.VersionString())
	os.Exit(0)
}

// PrintVersionJSON 以JSON格式打印版本信息並退出
func PrintVersionJSON() {
	v := config.GetVersion()
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling version data: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
	os.Exit(0)
}

// HandleVersionFlag 處理命令行的版本標誌
func HandleVersionFlag() {
	for i, arg := range os.Args {
		if arg == "--version" || arg == "-v" {
			PrintVersion()
		} else if arg == "--version-json" {
			PrintVersionJSON()
		} else if i == 1 && arg == "version" {
			PrintVersion()
		}
	}
}
