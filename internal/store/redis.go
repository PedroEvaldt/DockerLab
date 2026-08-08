package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/PedroEvaldt/shortener/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(ctx context.Context, cfg config.Config) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisStore{
		client: client,
	}, nil
}

func (r *RedisStore) IncrementClick(ctx context.Context, code string) error {
	err := r.client.Incr(ctx, fmt.Sprintf("clicks:%s", code)).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisStore) PendingClicks(ctx context.Context, code string) (int64, error) {
	val, err := r.client.Get(ctx, fmt.Sprintf("clicks:%s", code)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (r *RedisStore) DrainClicks(ctx context.Context) (map[string]int64, error) {
	clicks := make(map[string]int64)
	var cursor uint64

	for {
		var keys []string
		var err error

		keys, cursor, err = r.client.Scan(ctx, cursor, "clicks:*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			val, err := r.client.GetDel(ctx, key).Int64()
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				return nil, err
			}
			code := strings.TrimPrefix(key, "clicks:")
			clicks[code] = val
		}
		if cursor == 0 {
			break
		}
	}
	return clicks, nil
}

func (r *RedisStore) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return err
	}
	return nil
}

func (r *RedisStore) Close() error {
	if err := r.client.Close(); err != nil {
		return err
	}
	return nil
}
