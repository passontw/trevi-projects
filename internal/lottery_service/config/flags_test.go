package config

import (
	"testing"
)

func TestNormalizeNacosAddr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"EmptyAddress", "", "http://127.0.0.1:8848"},
		{"NoProtocol", "example.com:8848", "http://127.0.0.1:8848"},
		{"WithHTTP", "http://example.com:8848", "http://example.com:8848"},
		{"WithHTTPS", "https://example.com:8848", "https://example.com:8848"},
		{"NoPort", "http://example.com", "http://127.0.0.1:8848"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 設置輸入
			Args.NacosAddr = tc.input
			// 調用方法
			normalizeNacosAddr()
			// 檢查結果
			if Args.NacosAddr != tc.expected {
				t.Errorf("normalizeNacosAddr() = %v, want %v", Args.NacosAddr, tc.expected)
			}
		})
	}
}

func TestGetNacosHostAndPort(t *testing.T) {
	tests := []struct {
		name      string
		nacosAddr string
		wantHost  string
		wantPort  string
		wantHttps bool
	}{
		{"HTTPWithPort", "http://example.com:8848", "example.com", "8848", false},
		{"HTTPSWithPort", "https://example.com:8848", "example.com", "8848", true},
		{"DefaultAddress", "http://127.0.0.1:8848", "127.0.0.1", "8848", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 直接設置地址，不經過normalizeNacosAddr
			Args.NacosAddr = tc.nacosAddr

			// 調用方法
			gotHost, gotPort, gotHttps := GetNacosHostAndPort()

			// 檢查結果
			if gotHost != tc.wantHost {
				t.Errorf("GetNacosHostAndPort() host = %v, want %v", gotHost, tc.wantHost)
			}
			if gotPort != tc.wantPort {
				t.Errorf("GetNacosHostAndPort() port = %v, want %v", gotPort, tc.wantPort)
			}
			if gotHttps != tc.wantHttps {
				t.Errorf("GetNacosHostAndPort() https = %v, want %v", gotHttps, tc.wantHttps)
			}
		})
	}
}

func TestGetNacosServer(t *testing.T) {
	tests := []struct {
		name      string
		nacosAddr string
		want      string
	}{
		{"HTTPWithPort", "http://example.com:8848", "example.com:8848"},
		{"HTTPSWithPort", "https://example.com:8848", "example.com:8848"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 直接設置地址，不經過normalizeNacosAddr
			Args.NacosAddr = tc.nacosAddr

			// 調用方法
			got := GetNacosServer()

			// 檢查結果
			if got != tc.want {
				t.Errorf("GetNacosServer() = %v, want %v", got, tc.want)
			}
		})
	}
}
