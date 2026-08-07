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
		return nil, ErrNotFound
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
	// INSERT INTO links (code, url) VALUES ($1, $2), retornando o erro
	return nil
}

func (ps *PostgresStore) GetLink(ctx context.Context, code string) (url string, clicks int64, err error) {
	// SELECT FROM links WHERE code = $1
	// pool.QueryRow().Scan(&url, &clicks)
	return
}

func (ps *PostgresStore) IncrementClicks(ctx context.Context, code string, delta int64) error {
	// UPDATE links SET clicks = clicks + $1 WHERE code = $2
	// Checar result.RowsAffected() == 0, caso de, retornar ErrNotFOund
	return nil
}

func (ps *PostgresStore) Ping(ctx context.Context) error {
	return ps.pool.Ping(ctx)
}

func (ps *PostgresStore) Close() {
	ps.pool.Close()
}
