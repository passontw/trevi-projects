package config

import (
	"testing"
)

func TestParseRedisXmlConfig(t *testing.T) {
	// 測試案例 1: 單節點配置
	xmlSingleNode := `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<config>
  <redis host="127.0.0.1" port="6379" password="test123" db="0" is_cluster="false">
  </redis>
</config>`

	config := &AppConfig{
		Redis: RedisConfig{
			Host:     "",
			Port:     0,
			Username: "default",
			Password: "",
			DB:       0,
		},
	}

	err := parseRedisXmlConfig(xmlSingleNode, config)
	if err != nil {
		t.Errorf("解析單節點配置失敗: %v", err)
	}

	// 驗證單節點配置結果
	if config.Redis.Host != "127.0.0.1" {
		t.Errorf("單節點 Host 解析錯誤，預期: 127.0.0.1, 實際: %s", config.Redis.Host)
	}
	if config.Redis.Port != 6379 {
		t.Errorf("單節點 Port 解析錯誤，預期: 6379, 實際: %d", config.Redis.Port)
	}
	if config.Redis.Password != "test123" {
		t.Errorf("單節點 Password 解析錯誤，預期: test123, 實際: %s", config.Redis.Password)
	}
	if config.Redis.DB != 0 {
		t.Errorf("單節點 DB 解析錯誤，預期: 0, 實際: %d", config.Redis.DB)
	}
	// 驗證 Username 是否為空
	if config.Redis.Username != "" {
		t.Errorf("單節點 Username 應該為空，實際: %s", config.Redis.Username)
	}

	// 測試案例 2: 集群配置
	xmlCluster := `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<config>
  <redis host="" port="" password="safesync" db="0" is_cluster="true">
    <nodes>
      <node>10.141.1.32:7000</node>
      <node>10.141.1.33:7001</node>
    </nodes>
  </redis>
</config>`

	config = &AppConfig{
		Redis: RedisConfig{
			Host:     "",
			Port:     0,
			Username: "default",
			Password: "",
			DB:       0,
		},
	}

	err = parseRedisXmlConfig(xmlCluster, config)
	if err != nil {
		t.Errorf("解析集群配置失敗: %v", err)
	}

	// 驗證集群配置結果
	if config.Redis.Host != "10.141.1.32" {
		t.Errorf("集群 Host 解析錯誤，預期: 10.141.1.32, 實際: %s", config.Redis.Host)
	}
	if config.Redis.Port != 7000 {
		t.Errorf("集群 Port 解析錯誤，預期: 7000, 實際: %d", config.Redis.Port)
	}
	if config.Redis.Password != "safesync" {
		t.Errorf("集群 Password 解析錯誤，預期: safesync, 實際: %s", config.Redis.Password)
	}
	if config.Redis.DB != 0 {
		t.Errorf("集群 DB 解析錯誤，預期: 0, 實際: %d", config.Redis.DB)
	}
	// 驗證 Username 是否為空
	if config.Redis.Username != "" {
		t.Errorf("集群 Username 應該為空，實際: %s", config.Redis.Username)
	}

	// 測試案例 3: 空 XML
	err = parseRedisXmlConfig("", config)
	if err == nil {
		t.Errorf("空 XML 應該返回錯誤")
	}

	// 測試案例 4: 無效 XML
	err = parseRedisXmlConfig("<invalid>", config)
	if err == nil {
		t.Errorf("無效 XML 應該返回錯誤")
	}

	// 測試案例 5: 帶有 username 屬性的配置
	xmlWithUsername := `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<config>
  <redis host="127.0.0.1" port="6379" username="redis_user" password="test123" db="0" is_cluster="false">
  </redis>
</config>`

	config = &AppConfig{
		Redis: RedisConfig{
			Host:     "",
			Port:     0,
			Username: "default", // 設定預設用戶名，確保被覆蓋為空
			Password: "",
			DB:       0,
		},
	}

	err = parseRedisXmlConfig(xmlWithUsername, config)
	if err != nil {
		t.Errorf("解析帶有 username 配置失敗: %v", err)
	}

	// 驗證 Username 是否為空（即使 XML 中有值，仍應被強制設為空以避免認證問題）
	if config.Redis.Username != "" {
		t.Errorf("即使 XML 中指定 Username 也應該被清空，實際: %s", config.Redis.Username)
	}
}
