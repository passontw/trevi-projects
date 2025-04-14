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
	Host    string
	Port    uint64
	APIHost string
	Version string
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
	Port          string `json:"PORT" yaml:"PORT"`
	DBHost        string `json:"DB_HOST" yaml:"DB_HOST"`
	DBPort        int    `json:"DB_PORT" yaml:"DB_PORT"`
	DBName        string `json:"DB_NAME" yaml:"DB_NAME"`
	DBUser        string `json:"DB_USER" yaml:"DB_USER"`
	DBPassword    string `json:"DB_PASSWORD" yaml:"DB_PASSWORD"`
	JWTSecret     string `json:"JWT_SECRET" yaml:"JWT_SECRET"`
	JWTExpiresIn  string `json:"JWT_EXPIRES_IN" yaml:"JWT_EXPIRES_IN"`
	APIHost       string `json:"API_HOST" yaml:"API_HOST"`
	Version       string `json:"VERSION" yaml:"VERSION"`
	RedisHost     string `json:"REDIS_HOST" yaml:"REDIS_HOST"`
	RedisPort     string `json:"REDIS_PORT" yaml:"REDIS_PORT"`
	RedisUsername string `json:"REDIS_USERNAME" yaml:"REDIS_USERNAME"`
	RedisPassword string `json:"REDIS_PASSWORD" yaml:"REDIS_PASSWORD"`
	RedisDB       int    `json:"REDIS_DB" yaml:"REDIS_DB"`
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
