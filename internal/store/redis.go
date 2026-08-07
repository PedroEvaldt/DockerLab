package store

import (
	"github.com/PedroEvaldt/shortener/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(cfg config.Config) *RedisStore {
	client := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	return &RedisStore{
		client: client,
	}
}
