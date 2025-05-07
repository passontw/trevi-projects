package config

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestParseIntValue(t *testing.T) {
	testCases := []struct {
		name          string
		input         interface{}
		expectedValue int
		expectedValid bool
	}{
		{"integer", 8080, 8080, true},
		{"float64", 8080.0, 8080, true},
		{"string_number", "8080", 8080, true},
		{"string_float", "8080.0", 8080, true},
		{"json_number_valid", json.Number("8080"), 8080, true},
		{"nil", nil, 0, false},
		{"invalid_string", "not_a_number", 0, false},
		{"bool", true, 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, valid := parseIntValue(tc.input)
			if valid != tc.expectedValid {
				t.Errorf("Valid flag: expected %v, got %v", tc.expectedValid, valid)
			}
			if valid && value != tc.expectedValue {
				t.Errorf("Value: expected %d, got %d", tc.expectedValue, value)
			}
		})
	}
}

func TestMapJsonToAppConfig(t *testing.T) {
	// 測試不同格式的 DEALER_WS_PORT 值
	testCases := []struct {
		name         string
		wsPortValue  interface{}
		expectedPort int
	}{
		{"integer", 8080, 8080},
		{"float", 8080.0, 8080},
		{"string", "8080", 8080},
		{"json_number", json.Number("8080"), 8080},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 建立測試數據
			jsonMap := map[string]interface{}{
				"DEALER_WS_PORT": tc.wsPortValue,
			}

			// 測試配置物件
			config := createDefaultConfig()
			// 保存原始配置的端口值
			originalPort := config.Server.DealerWsPort
			// 設置不同的端口，確保測試能看到變化
			config.Server.DealerWsPort = 0

			// 執行測試
			err := mapJsonToAppConfig(jsonMap, config)
			if err != nil {
				t.Fatalf("測試案例 %s 失敗，發生錯誤: %v", tc.name, err)
			}

			// 驗證結果
			if config.Server.DealerWsPort != tc.expectedPort {
				t.Errorf("測試案例 %s 失敗: 期望端口為 %d, 但得到 %d. 原始端口值: %d",
					tc.name, tc.expectedPort, config.Server.DealerWsPort, originalPort)
			}
		})
	}
}

func TestParseNacosConfig(t *testing.T) {
	// 測試 Nacos JSON 的解析
	testCases := []struct {
		name         string
		jsonContent  string
		expectedPort int
	}{
		{
			name: "正確的數字類型",
			jsonContent: `{
				"DEALER_WS_PORT": 8080
			}`,
			expectedPort: 8080,
		},
		{
			name: "字符串類型的數字",
			jsonContent: `{
				"DEALER_WS_PORT": "8080"
			}`,
			expectedPort: 8080,
		},
		{
			name: "帶註釋的 JSON",
			jsonContent: `{
				// 這是一個註釋
				"DEALER_WS_PORT": 8080
			}`,
			expectedPort: 8080,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 初始化配置
			config := createDefaultConfig()
			originalPort := config.Server.DealerWsPort
			// 設置不同的端口，確保測試能看到變化
			config.Server.DealerWsPort = 0

			// 解析配置
			err := parseNacosConfig(tc.jsonContent, config)
			if err != nil {
				t.Fatalf("解析 Nacos 配置失敗: %v", err)
			}

			// 驗證結果
			if config.Server.DealerWsPort != tc.expectedPort {
				t.Errorf("期望端口為 %d, 但得到 %d. 原始端口值: %d",
					tc.expectedPort, config.Server.DealerWsPort, originalPort)
			}
		})
	}
}

func TestJsonNumberHandling(t *testing.T) {
	// 測試 JSON 解碼時數字的處理
	jsonContent := `{"port": 8080, "port_str": "8080"}`
	var result map[string]interface{}

	// 標準解析（將數字轉為 float64）
	err := json.Unmarshal([]byte(jsonContent), &result)
	if err != nil {
		t.Fatalf("解析 JSON 失敗: %v", err)
	}

	t.Logf("標準解析結果: port 類型 = %T, 值 = %v", result["port"], result["port"])
	t.Logf("標準解析結果: port_str 類型 = %T, 值 = %v", result["port_str"], result["port_str"])

	// 使用 json.UseNumber()
	decoder := json.NewDecoder(strings.NewReader(jsonContent))
	decoder.UseNumber()
	var result2 map[string]interface{}
	if err := decoder.Decode(&result2); err != nil {
		t.Fatalf("使用 UseNumber 解析 JSON 失敗: %v", err)
	}

	t.Logf("UseNumber 解析結果: port 類型 = %T, 值 = %v", result2["port"], result2["port"])
	t.Logf("UseNumber 解析結果: port_str 類型 = %T, 值 = %v", result2["port_str"], result2["port_str"])

	// 測試數值解析函數
	testParseIntValue := func(v interface{}) (int, bool) {
		// 這是 parseIntValue 函數的簡化版
		t.Logf("嘗試解析值: %v (類型: %T)", v, v)

		switch v := v.(type) {
		case int:
			t.Logf("識別為 int 類型")
			return v, true
		case float64:
			t.Logf("識別為 float64 類型")
			return int(v), true
		case string:
			t.Logf("識別為 string 類型")
			if intVal, err := strconv.Atoi(v); err == nil {
				t.Logf("字符串可直接轉換為整數: %d", intVal)
				return intVal, true
			}
			if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
				t.Logf("字符串可轉換為浮點數然後轉為整數: %d", int(floatVal))
				return int(floatVal), true
			}
			t.Logf("字符串無法轉換為數值")
		case json.Number:
			t.Logf("識別為 json.Number 類型")
			if intVal, err := v.Int64(); err == nil {
				t.Logf("json.Number 可轉換為整數: %d", intVal)
				return int(intVal), true
			}
			if floatVal, err := v.Float64(); err == nil {
				t.Logf("json.Number 可轉換為浮點數然後轉為整數: %d", int(floatVal))
				return int(floatVal), true
			}
			t.Logf("json.Number 無法轉換為數值")
		default:
			t.Logf("未知類型: %T", v)
		}
		return 0, false
	}

	// 測試不同類型的值
	tests := []struct {
		name  string
		value interface{}
	}{
		{"整數", 8080},
		{"浮點數", 8080.0},
		{"字符串數字", "8080"},
		{"字符串浮點數", "8080.5"},
		{"json.Number", result2["port"]},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := testParseIntValue(tc.value)
			if ok {
				t.Logf("成功解析為: %d", val)
			} else {
				t.Logf("無法解析")
			}
		})
	}
}

func TestNacosConfigParsing(t *testing.T) {
	// 這不是一個完整的測試，只是演示 Nacos 配置解析的過程
	t.Log("模擬 Nacos 配置解析過程:")

	// 模擬從 Nacos 獲取的 JSON 配置
	nacosJson := `{
		"DEALER_WS_PORT": 8080,
		"REDIS_HOST": "redis-server",
		"REDIS_PORT": 6379
	}`

	// 預處理 JSON 並檢查有效性
	processedContent := preprocessJsonContent(nacosJson)
	if !isValidJson(processedContent) {
		t.Fatalf("JSON 預處理後不是有效的 JSON")
	}

	// 解析 JSON 並將值作為 json.Number 處理
	var jsonMap map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(processedContent))
	decoder.UseNumber() // 使用 json.Number 而不是 float64
	if err := decoder.Decode(&jsonMap); err != nil {
		t.Fatalf("解析 JSON 失敗: %v", err)
	}

	// 輸出解析後的值類型
	for key, value := range jsonMap {
		t.Logf("配置項: %s, 值: %v, 類型: %T", key, value, value)
	}

	// 測試整數解析
	if dealerPort, exists := jsonMap["DEALER_WS_PORT"]; exists {
		t.Logf("讀取到 DEALER_WS_PORT: %v (類型: %T)", dealerPort, dealerPort)

		// 使用 config.go 中的代碼片段進行解析
		portValue, ok := func(v interface{}) (int, bool) {
			// 對應 parseIntValue 函數的簡化版
			switch v := v.(type) {
			case int:
				return v, true
			case float64:
				return int(v), true
			case string:
				if intVal, err := strconv.Atoi(v); err == nil {
					return intVal, true
				}
				if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
					return int(floatVal), true
				}
			case json.Number:
				if intVal, err := v.Int64(); err == nil {
					return int(intVal), true
				}
				if floatVal, err := v.Float64(); err == nil {
					return int(floatVal), true
				}
			}
			return 0, false
		}(dealerPort)

		if ok {
			t.Logf("解析後的 DEALER_WS_PORT 值: %d", portValue)
		} else {
			t.Errorf("無法解析 DEALER_WS_PORT 值: %v", dealerPort)
		}
	} else {
		t.Log("DEALER_WS_PORT 未在配置中找到")
	}
}
