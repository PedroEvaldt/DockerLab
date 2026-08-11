package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Pacote que configura as configs que meu app vai precisar usar para conectar nos containers

type Config struct {
	AppPort       string
	DBHost        string
	DBPort        string
	DBUser        string
	DBName        string
	DBPassword    string
	RedisAddr     string
	PublicBaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	password, err := loadPassword()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppPort:       getEnvDefault("APP_PORT", "8080"),
		DBHost:        getEnvDefault("DB_HOST", ""),
		DBPort:        getEnvDefault("DB_PORT", "5432"),
		DBUser:        getEnvDefault("DB_USER", "postgres"),
		DBName:        getEnvDefault("DB_NAME", "links"),
		DBPassword:    password,
		RedisAddr:     getEnvDefault("REDIS_ADDR", "localhost:6379"),
		PublicBaseURL: getEnvDefault("PUBLIC_BASE_URL", "http://localhost:8080"),
	}
	if cfg.DBHost == "" {
		return nil, errors.New("DB_HOST is required")
	}

	if cfg.DBPassword == "" {
		return nil, errors.New("DB_PASSWORD is required")
	}
	return cfg, nil
}

func loadPassword() (string, error) {
	if file := os.Getenv("POSTGRES_PASSWORD_FILE"); file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return os.Getenv("DB_PASSWORD"), nil
}

func getEnvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
