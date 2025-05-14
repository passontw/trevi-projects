package nacosManager

import (
	"reflect"
	"testing"
)

func TestParseGenericXMLConfig(t *testing.T) {
	testCases := []struct {
		name     string
		xmlInput string
		want     map[string]interface{}
		wantErr  bool
	}{
		{
			name: "基本 XML 配置",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <API_PORT>8000</API_PORT>
  <DEALER_WS_PORT>8080</DEALER_WS_PORT>
  <GRPC_PORT>9100</GRPC_PORT>
  <ROCKETMQ_ENABLED>true</ROCKETMQ_ENABLED>
  <ROCKETMQ_PRODUCER_GROUP>roommsg_to_livesvr</ROCKETMQ_PRODUCER_GROUP>
  <ROCKETMQ_CONSUMER_GROUP>lottery-consumer-group</ROCKETMQ_CONSUMER_GROUP>
</config>`,
			want: map[string]interface{}{
				"API_PORT":                8000,
				"DEALER_WS_PORT":          8080,
				"GRPC_PORT":               9100,
				"ROCKETMQ_ENABLED":        true,
				"ROCKETMQ_PRODUCER_GROUP": "roommsg_to_livesvr",
				"ROCKETMQ_CONSUMER_GROUP": "lottery-consumer-group",
			},
			wantErr: false,
		},
		{
			name:     "空配置",
			xmlInput: "",
			want:     nil,
			wantErr:  true,
		},
		{
			name: "不完整 XML",
			xmlInput: `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <API_PORT>8000</API_PORT>
`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseGenericXMLConfig(tc.xmlInput)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseGenericXMLConfig() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseGenericXMLConfig() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseJSONConfig(t *testing.T) {
	testCases := []struct {
		name      string
		jsonInput string
		want      map[string]interface{}
		wantErr   bool
	}{
		{
			name: "基本 JSON 配置",
			jsonInput: `{
    "API_PORT": 8000,
    "DEALER_WS_PORT": 8080,
    "GRPC_PORT": 9100,
    "ROCKETMQ_ENABLED": true,
    "ROCKETMQ_NAME_SERVERS": ["172.237.27.51:9876"],
    "ROCKETMQ_PRODUCER_GROUP": "roommsg_to_livesvr",
    "ROCKETMQ_CONSUMER_GROUP": "lottery-consumer-group" 
}`,
			want: map[string]interface{}{
				"API_PORT":                8000.0,
				"DEALER_WS_PORT":          8080.0,
				"GRPC_PORT":               9100.0,
				"ROCKETMQ_ENABLED":        true,
				"ROCKETMQ_NAME_SERVERS":   []interface{}{"172.237.27.51:9876"},
				"ROCKETMQ_PRODUCER_GROUP": "roommsg_to_livesvr",
				"ROCKETMQ_CONSUMER_GROUP": "lottery-consumer-group",
			},
			wantErr: false,
		},
		{
			name:      "空配置",
			jsonInput: "",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "不完整 JSON",
			jsonInput: `{"API_PORT": 8000,`,
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseJSONConfig(tc.jsonInput)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseJSONConfig() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseJSONConfig() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectAndParseConfig(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "XML 配置",
			input: `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <API_PORT>8000</API_PORT>
  <DEALER_WS_PORT>8080</DEALER_WS_PORT>
</config>`,
			want: map[string]interface{}{
				"API_PORT":       8000,
				"DEALER_WS_PORT": 8080,
			},
			wantErr: false,
		},
		{
			name: "JSON 配置",
			input: `{
    "API_PORT": 8000,
    "DEALER_WS_PORT": 8080
}`,
			want: map[string]interface{}{
				"API_PORT":       8000.0,
				"DEALER_WS_PORT": 8080.0,
			},
			wantErr: false,
		},
		{
			name:    "空配置",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "不支持的格式",
			input:   "API_PORT=8000\nDEALER_WS_PORT=8080",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DetectAndParseConfig(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("DetectAndParseConfig() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("DetectAndParseConfig() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseYourXMLConfig(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <API_PORT>8000</API_PORT>
  <DEALER_WS_PORT>8080</DEALER_WS_PORT>
  <GRPC_PORT>9100</GRPC_PORT>
  <ROCKETMQ_ENABLED>true</ROCKETMQ_ENABLED>
  <ROCKETMQ_PRODUCER_GROUP>roommsg_to_livesvr</ROCKETMQ_PRODUCER_GROUP>
  <ROCKETMQ_CONSUMER_GROUP>lottery-consumer-group</ROCKETMQ_CONSUMER_GROUP>
</config>`

	expected := map[string]interface{}{
		"API_PORT":                8000,
		"DEALER_WS_PORT":          8080,
		"GRPC_PORT":               9100,
		"ROCKETMQ_ENABLED":        true,
		"ROCKETMQ_PRODUCER_GROUP": "roommsg_to_livesvr",
		"ROCKETMQ_CONSUMER_GROUP": "lottery-consumer-group",
	}

	config, err := ParseGenericXMLConfig(xmlContent)
	if err != nil {
		t.Fatalf("解析XML配置失敗: %v", err)
	}

	// 檢查解析結果
	for key, expectedValue := range expected {
		if value, ok := config[key]; !ok {
			t.Errorf("缺少配置項: %s", key)
		} else if !reflect.DeepEqual(value, expectedValue) {
			t.Errorf("配置項 %s 的值不符: 期望 %v, 實際 %v", key, expectedValue, value)
		}
	}
}

func TestParseYourJSONConfig(t *testing.T) {
	jsonContent := `{
    "API_PORT": 8000,
    "DEALER_WS_PORT": 8080,
    "GRPC_PORT": 9100,
    "ROCKETMQ_ENABLED": true,
    "ROCKETMQ_NAME_SERVERS": ["172.237.27.51:9876"],
    "ROCKETMQ_PRODUCER_GROUP": "roommsg_to_livesvr",
    "ROCKETMQ_CONSUMER_GROUP": "lottery-consumer-group" 
}`

	expected := map[string]interface{}{
		"API_PORT":                8000.0, // JSON 數字會被解析為 float64
		"DEALER_WS_PORT":          8080.0,
		"GRPC_PORT":               9100.0,
		"ROCKETMQ_ENABLED":        true,
		"ROCKETMQ_NAME_SERVERS":   []interface{}{"172.237.27.51:9876"},
		"ROCKETMQ_PRODUCER_GROUP": "roommsg_to_livesvr",
		"ROCKETMQ_CONSUMER_GROUP": "lottery-consumer-group",
	}

	config, err := ParseJSONConfig(jsonContent)
	if err != nil {
		t.Fatalf("解析JSON配置失敗: %v", err)
	}

	// 檢查解析結果
	for key, expectedValue := range expected {
		if value, ok := config[key]; !ok {
			t.Errorf("缺少配置項: %s", key)
		} else if !reflect.DeepEqual(value, expectedValue) {
			t.Errorf("配置項 %s 的值不符: 期望 %v, 實際 %v", key, expectedValue, value)
		}
	}
}

func TestDetectAndParseYourConfig(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected map[string]interface{}
	}{
		{
			name: "XML 配置",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <API_PORT>8000</API_PORT>
  <DEALER_WS_PORT>8080</DEALER_WS_PORT>
</config>`,
			expected: map[string]interface{}{
				"API_PORT":       8000,
				"DEALER_WS_PORT": 8080,
			},
		},
		{
			name: "JSON 配置",
			content: `{
    "API_PORT": 8000,
    "DEALER_WS_PORT": 8080
}`,
			expected: map[string]interface{}{
				"API_PORT":       8000.0,
				"DEALER_WS_PORT": 8080.0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := DetectAndParseConfig(tc.content)
			if err != nil {
				t.Fatalf("解析配置失敗: %v", err)
			}

			// 檢查解析結果
			for key, expectedValue := range tc.expected {
				if value, ok := config[key]; !ok {
					t.Errorf("缺少配置項: %s", key)
				} else if !reflect.DeepEqual(value, expectedValue) {
					t.Errorf("配置項 %s 的值不符: 期望 %v, 實際 %v",
						key, expectedValue, value)
				}
			}
		})
	}
}
