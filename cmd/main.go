package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/PedroEvaldt/shortener/internal/config"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ps, err := store.NewPostgresStore(ctx, *cfg)
	if err != nil {
		return fmt.Errorf("Error getting database from config: %w", err)
	}

	err = ps.SaveLink(ctx, "teste01", "https://google.com")
	if err != nil {
		return fmt.Errorf("save link: %w", err)
	}
	url, clicks, err := ps.GetLink(ctx, "teste01")
	if err != nil {
		return fmt.Errorf("get link: %w", err)
	}
	if url != "https://google.com" {
		return fmt.Errorf("expected https://google.com; got %s", url)
	}
	err = ps.IncrementClicks(ctx, "teste01", 5)
	if err != nil {
		return fmt.Errorf("increment link: %w", err)
	}
	url, clicks, err = ps.GetLink(ctx, "teste01")
	if err != nil {
		return fmt.Errorf("get link: %w", err)
	}
	if clicks != 5 {
		return fmt.Errorf("expected 5; got %d", clicks)
	}
	url, clicks, err = ps.GetLink(ctx, "naoexiste")
	if err == nil {
		return fmt.Errorf("expected error; got nil")
	}
	return nil
}
