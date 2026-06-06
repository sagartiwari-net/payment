package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppEnv   string
	AppPort  string
	AppURL   string
	DBHost   string
	DBPort   string
	DBName   string
	DBUser   string
	DBPass   string
	RedisURL string
	LogLevel string

	PhonePeMerchantID string
	PhonePeSaltKey    string
	PhonePeSaltIndex  string
	PhonePeEnv        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()

	viper.AutomaticEnv()

	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("APP_URL", "http://localhost:8080")
	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_NAME", "paymentsystem")
	viper.SetDefault("DB_USER", "paymentsystem")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("REDIS_URL", "redis://127.0.0.1:6379")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("PHONEPE_MERCHANT_ID", "")
	viper.SetDefault("PHONEPE_SALT_KEY", "")
	viper.SetDefault("PHONEPE_SALT_INDEX", "1")
	viper.SetDefault("PHONEPE_ENV", "PRODUCTION")

	cfg := &Config{
		AppEnv:   viper.GetString("APP_ENV"),
		AppPort:  viper.GetString("APP_PORT"),
		AppURL:   strings.TrimRight(viper.GetString("APP_URL"), "/"),
		DBHost:   viper.GetString("DB_HOST"),
		DBPort:   viper.GetString("DB_PORT"),
		DBName:   viper.GetString("DB_NAME"),
		DBUser:   viper.GetString("DB_USER"),
		DBPass:   viper.GetString("DB_PASSWORD"),
		RedisURL: viper.GetString("REDIS_URL"),
		LogLevel: viper.GetString("LOG_LEVEL"),

		PhonePeMerchantID: viper.GetString("PHONEPE_MERCHANT_ID"),
		PhonePeSaltKey:    viper.GetString("PHONEPE_SALT_KEY"),
		PhonePeSaltIndex:  viper.GetString("PHONEPE_SALT_INDEX"),
		PhonePeEnv:        viper.GetString("PHONEPE_ENV"),
	}

	if cfg.DBPass == "" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("DB_PASSWORD is required in production")
	}

	return cfg, nil
}

func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC&multiStatements=true",
		c.DBUser,
		c.DBPass,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}
