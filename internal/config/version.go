package config

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// 這些變數會在編譯時通過 Makefile 設置
var (
	AppVersion string
	GitCommit  string
	BuildDate  string
)

// Version 包含應用程序的版本信息
type Version struct {
	Version    string `json:"version"`     // 語義化版本號 (x.y.z)
	APIVersion string `json:"api_version"` // API 版本 (v1, v2...)
	AppName    string `json:"app_name"`    // 應用名稱
	BuildDate  string `json:"build_date"`  // 構建日期
	GitCommit  string `json:"git_commit"`  // Git 提交 hash
	GoVersion  string `json:"go_version"`  // Go 版本
	Platform   string `json:"platform"`    // 運行平台
	BuildEnv   string `json:"build_env"`   // 構建環境 (development/production)
}

var (
	// 全局版本信息，在初始化時從環境變量獲取
	currentVersion Version
)

// 初始化版本信息
func init() {
	// 優先使用編譯時注入的版本信息
	appVersion := AppVersion
	if appVersion == "" {
		appVersion = getEnv("APP_VERSION", "0.1.0")
	}

	// 預設使用 "v1" 作為 API 版本
	apiVersion := "v1"

	// 嘗試從 APP_VERSION 擷取主要版本號
	if parts := strings.Split(appVersion, "."); len(parts) > 0 {
		apiVersion = "v" + parts[0]
	}

	// 取得 Git 提交信息
	gitCommit := GitCommit
	if gitCommit == "" {
		gitCommit = getEnv("GIT_COMMIT", "unknown")
	}

	// 取得構建日期
	buildDate := BuildDate
	if buildDate == "" {
		buildDate = getEnv("BUILD_DATE", time.Now().Format(time.RFC3339))
	}

	// 從環境變量獲取版本信息，或使用默認值
	currentVersion = Version{
		Version:    appVersion,
		APIVersion: apiVersion,
		AppName:    getEnv("APP_NAME", "G38 Lottery Service"),
		BuildDate:  buildDate,
		GitCommit:  gitCommit,
		GoVersion:  runtime.Version(),
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BuildEnv:   getEnv("BUILD_ENV", "development"),
	}
}

// GetVersion 返回當前版本信息
func GetVersion() Version {
	return currentVersion
}

// VersionString 返回格式化的版本信息字符串
func VersionString() string {
	v := GetVersion()
	return fmt.Sprintf(
		"%s v%s\nBuild Date: %s\nGit Commit: %s\nGo Version: %s\nPlatform: %s\nBuild Environment: %s",
		v.AppName, v.Version, v.BuildDate, v.GitCommit, v.GoVersion, v.Platform, v.BuildEnv,
	)
}

// ShortVersionString 返回簡短的版本信息字符串
func ShortVersionString() string {
	v := GetVersion()
	return fmt.Sprintf("%s v%s (%s)", v.AppName, v.Version, v.BuildEnv)
}

// IsDevelopment 判斷是否為開發環境
func IsDevelopment() bool {
	return currentVersion.BuildEnv == "development"
}

// IsProduction 判斷是否為生產環境
func IsProduction() bool {
	return currentVersion.BuildEnv == "production"
}

// GetAPIVersion 返回 API 版本
func GetAPIVersion() string {
	return currentVersion.APIVersion
}
