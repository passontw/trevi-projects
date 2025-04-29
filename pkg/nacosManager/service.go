package nacosManager

import (
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// RegisterInstanceParam 創建服務註冊參數
func RegisterInstanceParam(config interface{}) vo.RegisterInstanceParam {
	// 從配置中提取所需的信息
	var serviceName, serviceIP string
	var servicePort uint64
	var metadata map[string]string

	// 根據配置類型提取信息
	switch cfg := config.(type) {
	case map[string]interface{}:
		// 如果是 map 類型的配置
		if name, ok := cfg["serviceName"].(string); ok {
			serviceName = name
		}
		if ip, ok := cfg["serviceIP"].(string); ok {
			serviceIP = ip
		}
		if port, ok := cfg["servicePort"].(float64); ok {
			servicePort = uint64(port)
		}
	default:
		// 嘗試通過反射獲取配置
		// 這裡簡化處理，實際使用時可能需要根據傳入的配置類型進行更詳細的處理
		if _, ok := config.(struct {
			Server struct {
				ServiceName string
				ServiceIP   string
				ServicePort int
			}
		}); ok {
			// 提取 Server.ServiceName, Server.ServiceIP, Server.ServicePort
			// 由於類型信息缺失，這裡僅作為示例
			serviceName = "g38_lottery_service" // 默認值，實際使用時應該從配置獲取
			serviceIP = "127.0.0.1"             // 默認值，實際使用時應該從配置獲取
			servicePort = 8080                  // 默認值，實際使用時應該從配置獲取
		}
	}

	// 設置默認值，以防配置中缺少相關字段
	if serviceName == "" {
		serviceName = "g38_lottery_service"
	}
	if serviceIP == "" {
		serviceIP = "127.0.0.1"
	}
	if servicePort == 0 {
		servicePort = 8080
	}

	// 添加一些基本的元數據
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["version"] = "1.0.0"
	metadata["env"] = "dev"

	// 構建並返回註冊參數
	return vo.RegisterInstanceParam{
		Ip:          serviceIP,
		Port:        servicePort,
		ServiceName: serviceName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    metadata,
	}
}
