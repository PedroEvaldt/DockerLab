package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PedroEvaldt/shortener/internal/config"
	"github.com/PedroEvaldt/shortener/internal/httpapi"
	"github.com/PedroEvaldt/shortener/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("aplicação deu erro", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Error loading config: %w", err)
	}
	ctxBoot, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pg, err := store.NewPostgresStore(ctxBoot, *cfg)
	if err != nil {
		return fmt.Errorf("Error getting database from config: %w", err)
	}
	defer pg.Close()
	rc, err := store.NewRedisStore(ctxBoot, *cfg)
	if err != nil {
		return fmt.Errorf("Error getting redis client from config: %w", err)
	}
	defer rc.Close()

	ctxApi, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(os.Args) > 1 && os.Args[1] == "worker" {
		return runWorker(ctxApi, cfg, pg, rc)
	}
	return runAPI(ctxApi, cfg, pg, rc)
}

func runAPI(ctx context.Context, cfg *config.Config, pg *store.PostgresStore, rc *store.RedisStore) {
	apiServer := httpapi.New(pg, rc, cfg.PublicBaseURL, slog.Default())
	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           apiServer.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go srv.ListenAndServe()
	<-ctx.Done()
	srv.Shutdown(ctx)
}

func runWorker(ctx context.Context, cfg *config.Config, pg *store.PostgresStore, rc *store.RedisStore) {
	fmt.Printf("oi")
}
