package infra

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/lpphub/goweb/pkg/config"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type JWTConfig struct {
	Secret            string
	ExpireTime        int64
	RefreshExpireTime int64
}

type ServerConfig struct {
	Port int
}

type Config struct {
	Database DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Server   ServerConfig
}

func LoadConfig() (*Config, error) {
	return config.LoadConf[Config]("./config/config.yml")
}