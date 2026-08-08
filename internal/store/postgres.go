package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/PedroEvaldt/shortener/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, cfg config.Config) (*PostgresStore, error) {
	dns := fmt.Sprintf("postgres://%s:postgres@%s:%s/%s?sslmode=disable", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	pool, err := pgxpool.New(ctx, dns)
	if err != nil {
		return nil, err
	}
	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{
		pool: pool,
	}, nil
}

func (ps *PostgresStore) SaveLink(ctx context.Context, code, url string) error {
	sql := "INSERT INTO links (code, url) VALUES ($1, $2);"
	_, err := ps.pool.Exec(ctx, sql, code, url)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStore) GetLink(ctx context.Context, code string) (url string, clicks int64, err error) {
	sql := "SELECT url, clicks FROM links WHERE code = $1;"
	err = ps.pool.QueryRow(ctx, sql, code).Scan(&url, &clicks)
	if err != nil {
		return "", 0, err
	}
	return url, clicks, nil
}

func (ps *PostgresStore) DeleteLink(ctx context.Context, code string) error {
	sql := "DELETE from links WHERE code = $1;"
	_, err := ps.pool.Exec(ctx, sql, code)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStore) IncrementClicks(ctx context.Context, code string, delta int64) error {
	sql := "UPDATE links SET clicks = clicks + $1 WHERE code = $2;"
	tag, err := ps.pool.Exec(ctx, sql, delta, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (ps *PostgresStore) Ping(ctx context.Context) error {
	return ps.pool.Ping(ctx)
}

func (ps *PostgresStore) Close() {
	ps.pool.Close()
}
