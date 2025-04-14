package httpClient

import (
	"time"

	"go.uber.org/fx"
)

// ModuleParams 模組參數
type ModuleParams struct {
	fx.In

	Config       *Config           `optional:"true"`
	ServiceName  string            `optional:"true"`
	Timeout      time.Duration     `optional:"true"`
	EndpointList []ServiceEndpoint `optional:"true"`
}

// ModuleResult 模組結果
type ModuleResult struct {
	fx.Out

	Client HTTPClient
}

// ProvideHTTPClient 提供HTTP客戶端實例
func ProvideHTTPClient(p ModuleParams) (HTTPClient, error) {
	var opts []ClientOption

	// 設置超時
	if p.Timeout > 0 {
		opts = append(opts, WithTimeout(p.Timeout))
	} else if p.Config != nil && p.Config.DefaultTimeout > 0 {
		opts = append(opts, WithTimeout(time.Duration(p.Config.DefaultTimeout)*time.Second))
	}

	// 設置服務名稱
	if p.ServiceName != "" {
		opts = append(opts, WithServiceName(p.ServiceName))
	}

	// 設置服務端點
	if p.Config != nil && len(p.Config.ServiceEndpoints) > 0 {
		var endpoints []ServiceEndpoint
		for _, endpoint := range p.Config.ServiceEndpoints {
			endpoints = append(endpoints, endpoint)
		}
		opts = append(opts, WithServiceEndpoints(endpoints))
	}

	// 從參數中添加端點
	if len(p.EndpointList) > 0 {
		opts = append(opts, WithServiceEndpoints(p.EndpointList))
	}

	client := NewClient(opts...)
	return client, nil
}

// Module HTTP客戶端模組
var Module = fx.Options(
	fx.Provide(
		ProvideHTTPClient,
	),
)
