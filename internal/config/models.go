package config

import (
	"time"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Nacos       NacosConfig
	EnableNacos bool
}

type ServerConfig struct {
	Host         string
	Port         uint64
	APIHost      string
	Version      string
	PlayerWSPort uint64 // 遊戲端 WebSocket 端口
	DealerWSPort uint64 // 荷官端 WebSocket 端口
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

type NacosConfig struct {
	Host        string
	Port        uint64
	NamespaceId string
	Group       string
	DataId      string
	Username    string
	Password    string
}

type NacosAppConfig struct {
	Port          string `json:"PORT,omitempty" yaml:"PORT,omitempty" json:"port,omitempty"`
	PlayerWSPort  string `json:"PLAYER_WS_PORT,omitempty" yaml:"PLAYER_WS_PORT,omitempty" json:"player_ws_port,omitempty"`
	DealerWSPort  string `json:"DEALER_WS_PORT,omitempty" yaml:"DEALER_WS_PORT,omitempty" json:"dealer_ws_port,omitempty"`
	DBHost        string `json:"DB_HOST,omitempty" yaml:"DB_HOST,omitempty" json:"db_host,omitempty"`
	DBPort        int    `json:"DB_PORT,omitempty" yaml:"DB_PORT,omitempty" json:"db_port,omitempty"`
	DBName        string `json:"DB_NAME,omitempty" yaml:"DB_NAME,omitempty" json:"db_name,omitempty"`
	DBUser        string `json:"DB_USER,omitempty" yaml:"DB_USER,omitempty" json:"db_user,omitempty"`
	DBPassword    string `json:"DB_PASSWORD,omitempty" yaml:"DB_PASSWORD,omitempty" json:"db_password,omitempty"`
	JWTSecret     string `json:"JWT_SECRET,omitempty" yaml:"JWT_SECRET,omitempty" json:"jwt_secret,omitempty"`
	JWTExpiresIn  string `json:"JWT_EXPIRES_IN,omitempty" yaml:"JWT_EXPIRES_IN,omitempty" json:"jwt_expires_in,omitempty"`
	APIHost       string `json:"API_HOST,omitempty" yaml:"API_HOST,omitempty" json:"api_host,omitempty"`
	Version       string `json:"VERSION,omitempty" yaml:"VERSION,omitempty" json:"version,omitempty"`
	RedisHost     string `json:"REDIS_HOST,omitempty" yaml:"REDIS_HOST,omitempty" json:"redis_host,omitempty"`
	RedisPort     string `json:"REDIS_PORT,omitempty" yaml:"REDIS_PORT,omitempty" json:"redis_port,omitempty"`
	RedisUsername string `json:"REDIS_USERNAME,omitempty" yaml:"REDIS_USERNAME,omitempty" json:"redis_username,omitempty"`
	RedisPassword string `json:"REDIS_PASSWORD,omitempty" yaml:"REDIS_PASSWORD,omitempty" json:"redis_password,omitempty"`
	RedisDB       int    `json:"REDIS_DB,omitempty" yaml:"REDIS_DB,omitempty" json:"redis_db,omitempty"`
}

func (c *Config) GetDatabaseHost() string {
	return c.Database.Host
}

func (c *Config) GetDatabasePort() int {
	return c.Database.Port
}

func (c *Config) GetDatabaseUser() string {
	return c.Database.User
}

func (c *Config) GetDatabasePassword() string {
	return c.Database.Password
}

func (c *Config) GetDatabaseName() string {
	return c.Database.Name
}

func (c *Config) IsNacosEnabled() bool {
	return c.EnableNacos
}

func (c *Config) GetNacosGroup() string {
	return c.Nacos.Group
}

func (c *Config) GetNacosDataId() string {
	return c.Nacos.DataId
}
