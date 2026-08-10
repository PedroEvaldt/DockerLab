package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/PedroEvaldt/shortener/internal/store"
)

func Run(ctx context.Context, pg *store.PostgresStore, rc *store.RedisStore, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			flush(flushCtx, pg, rc, slog.Default())
			cancel()
			return nil
		case <-t.C:
			flush(ctx, pg, rc, slog.Default())
		}
	}
}

func flush(ctx context.Context, pg *store.PostgresStore, rc *store.RedisStore, logger *slog.Logger) {
	clicksMap, err := rc.DrainClicks(ctx)
	if err != nil {
		logger.Warn("failed colecting redis clicks", "error", err)
		return
	}
	for key, value := range clicksMap {
		err := pg.IncrementClicks(ctx, key, value)
		if err != nil {
			logger.Warn("failed incrementing clicks", "key", key, "error", err)
			continue
		}
	}
}
