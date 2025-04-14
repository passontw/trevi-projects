package docs

import "github.com/swaggo/swag"

var SwaggerInfo = &swag.Spec{
	Version: "1.0",
	Host: "localhost:8080",
	BasePath: "/api/v1",
	Schemes: []string{"http", "https"},
	Title: "g38_lottery_servic API",
	Description: "API for g38_lottery_servic",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
