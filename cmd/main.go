package main

import (
	"context"
	"log"
	"time"

	"github.com/PedroEvaldt/shortener/internal/config"
	"github.com/PedroEvaldt/shortener/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error loading config")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ps, err := store.NewPostgresStore(ctx, cfg)
	if err != nil {
		cancel()
		log.Fatal("Error getting database from config")
	}
	store.SaveLink(ctx, "teste01", "https://google.com")
	store.GetLink(ctx, "teste01")
	store.IncrementClicks(ctx, "teste01", 5)
	store.GetLink(ctx, "teste01")
	store.GetLink(ctx, "naoexiste")
}
