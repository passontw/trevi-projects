package testing

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DockerManager 提供測試時管理 Docker 容器的功能
type DockerManager struct {
	composeFile string
	projectName string
	services    []string
}

// NewDockerManager 創建新的 Docker 管理器
func NewDockerManager(composeFile, projectName string, services []string) *DockerManager {
	return &DockerManager{
		composeFile: composeFile,
		projectName: projectName,
		services:    services,
	}
}

// StartContainers 啟動所有測試容器
func (dm *DockerManager) StartContainers() error {
	fmt.Println("正在啟動測試容器...")

	// 檢查 docker-compose 文件是否存在
	if _, err := os.Stat(dm.composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose 文件不存在: %s", dm.composeFile)
	}

	// 啟動容器
	args := []string{
		"-f", dm.composeFile,
		"-p", dm.projectName,
		"up", "-d",
	}

	// 如果有指定服務，則只啟動指定的服務
	if len(dm.services) > 0 {
		args = append(args, dm.services...)
	}

	cmd := exec.Command("docker-compose", args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("啟動容器失敗: %v\n輸出: %s\n錯誤: %s",
			err, outBuf.String(), errBuf.String())
	}

	fmt.Println("容器已啟動，等待服務就緒...")

	return dm.waitForServicesReady()
}

// StopContainers 停止並刪除所有測試容器
func (dm *DockerManager) StopContainers() error {
	fmt.Println("正在停止並刪除測試容器...")

	cmd := exec.Command("docker-compose",
		"-f", dm.composeFile,
		"-p", dm.projectName,
		"down", "--volumes", "--remove-orphans")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("停止容器失敗: %v\n輸出: %s\n錯誤: %s",
			err, outBuf.String(), errBuf.String())
	}

	fmt.Println("容器已停止並刪除")
	return nil
}

// waitForServicesReady 等待所有服務準備就緒
func (dm *DockerManager) waitForServicesReady() error {
	// 這裡可以根據不同服務實現不同的就緒檢查
	// 例如等待 TiDB 可以接受連接

	// 簡單實現：等待 10 秒
	fmt.Println("等待服務就緒 (10秒)...")
	time.Sleep(10 * time.Second)

	return nil
}

// ExecuteSQL 在 TiDB 中執行 SQL 指令
func (dm *DockerManager) ExecuteSQL(sqlFile string) error {
	if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
		return fmt.Errorf("SQL 文件不存在: %s", sqlFile)
	}

	// 從 compose 文件所在目錄獲取絕對路徑，用於挂載
	absPath, err := filepath.Abs(sqlFile)
	if err != nil {
		return fmt.Errorf("無法獲取 SQL 文件的絕對路徑: %v", err)
	}

	// 目錄和文件名
	dir := filepath.Dir(absPath)
	file := filepath.Base(absPath)

	// 在 TiDB 容器中執行 SQL 文件
	cmd := exec.Command("docker-compose",
		"-f", dm.composeFile,
		"-p", dm.projectName,
		"exec", "-T", "tidb",
		"sh", "-c", fmt.Sprintf("mysql -h127.0.0.1 -P4000 -uroot < /sql/%s", file))

	// 設置環境變數以掛載 SQL 目錄
	cmd.Env = append(os.Environ(), fmt.Sprintf("SQL_DIR=%s", dir))

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("執行 SQL 失敗: %v\n輸出: %s\n錯誤: %s",
			err, outBuf.String(), errBuf.String())
	}

	fmt.Printf("SQL 文件 %s 執行成功\n", sqlFile)
	return nil
}

// GetContainerIP 獲取容器的 IP 地址
func (dm *DockerManager) GetContainerIP(service string) (string, error) {
	cmd := exec.Command("docker-compose",
		"-f", dm.composeFile,
		"-p", dm.projectName,
		"exec", service,
		"sh", "-c", "ip addr show eth0 | grep 'inet ' | awk '{print $2}' | cut -d/ -f1")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("獲取容器 IP 失敗: %v\n錯誤: %s", err, errBuf.String())
	}

	ip := strings.TrimSpace(outBuf.String())
	if ip == "" {
		return "", fmt.Errorf("無法獲取容器 %s 的 IP 地址", service)
	}

	return ip, nil
}
