package config

import (
	"testing"
)

func TestParseTiDBXmlConfig(t *testing.T) {
	// 測試用的 XML 內容
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<config>
  <db name="dbname">
    <type>tidb</type>
    <host>127.0.0.1</host>
    <port>4001</port>
    <name>dbname</name>
    <user>admin</user>
    <password>test-password</password>
  </db>
</config>`

	// 創建一個默認配置
	cfg := &AppConfig{
		Database: DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "default",
			Password: "default",
			DBName:   "default",
		},
	}

	// 解析 XML 配置
	err := parseTiDBXmlConfig(xmlContent, cfg, "g38_lottery_service")
	if err != nil {
		t.Fatalf("解析 TiDB XML 配置失敗: %v", err)
	}

	// 驗證解析結果
	if cfg.Database.Host != "10.141.1.43" {
		t.Errorf("期望 Host='10.141.1.43', 實際 Host='%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 4001 {
		t.Errorf("期望 Port=4001, 實際 Port=%d", cfg.Database.Port)
	}
	if cfg.Database.DBName != "g38_loterry_service" {
		t.Errorf("期望 DBName='g38_loterry_service', 實際 DBName='%s'", cfg.Database.DBName)
	}
	if cfg.Database.Username != "admin" {
		t.Errorf("期望 Username='admin', 實際 Username='%s'", cfg.Database.Username)
	}
	if cfg.Database.Password != "test-password" {
		t.Errorf("期望 Password='test-password', 實際 Password='%s'", cfg.Database.Password)
	}
}

func TestParseTiDBXmlConfigWithNonExistingService(t *testing.T) {
	// 測試用的 XML 內容
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<config>
  <db name="dbname">
    <type>tidb</type>
    <host>127.0.0.1</host>
    <port>4001</port>
    <name>dbname</name>
    <user>admin</user>
    <password>test-password</password>
  </db>
</config>`

	// 創建一個默認配置
	cfg := &AppConfig{
		Database: DatabaseConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "default",
			Password: "default",
			DBName:   "default",
		},
	}

	// 解析 XML 配置，嘗試查找不存在的服務
	err := parseTiDBXmlConfig(xmlContent, cfg, "non_existing_service")
	if err != nil {
		t.Fatalf("解析 TiDB XML 配置失敗: %v", err)
	}

	// 驗證配置未被修改
	if cfg.Database.Host != "localhost" {
		t.Errorf("期望 Host='localhost', 實際 Host='%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("期望 Port=3306, 實際 Port=%d", cfg.Database.Port)
	}
}
