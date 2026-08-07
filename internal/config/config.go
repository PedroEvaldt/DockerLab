package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Pacote que configura as configs que meu app vai precisar usar para conectar nos containers

type Config struct {
	AppPort       string
	DBHost        string
	DBPort        string
	DBUser        string
	DBName        string
	RedisAddr     string
	PublicBaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{
		AppPort:       getEnvDefault("APP_PORT", "8080"),
		DBHost:        getEnvDefault("DB_HOST", ""),
		DBPort:        getEnvDefault("DB_PORT", "5432"),
		DBUser:        getEnvDefault("DB_USER", "postgres"),
		DBName:        getEnvDefault("DB_NAME", "link"),
		RedisAddr:     getEnvDefault("REDIS_ADDR", "localhost:6379"),
		PublicBaseURL: getEnvDefault("PUBLIC_BASE_URL", "http://localhost:8080"),
	}
	if cfg.DBHost == "" {
		return nil, errors.New("DB_HOST is required")
	}
	return cfg, nil
}

func getEnvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
