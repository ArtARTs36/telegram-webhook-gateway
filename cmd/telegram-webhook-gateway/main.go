package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/artarts36/go-entrypoint"
	"github.com/artarts36/telegram-webhook-gateway/internal/cidr"
	"github.com/artarts36/telegram-webhook-gateway/internal/config"
	"github.com/artarts36/telegram-webhook-gateway/internal/gateway"
	"github.com/cappuccinotm/slogx"
	"github.com/cappuccinotm/slogx/slogm"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.ErrorContext(ctx, "[main] failed to load config from environment", "err", err)
		os.Exit(1)
	}

	slogx.RequestIDKey = "x-request-id"
	slog.SetDefault(slog.New(slogx.NewChain(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.Log.Level,
	}), slogm.RequestID())))

	store := cidr.NewStore()
	fetcher := cidr.NewHTTPFetcher(cfg.Telegram.CIDRURL)

	updater := cidr.NewPeriodicUpdater(store, fetcher, cfg.Telegram.CIDRUpdateInterval.Value)
	updater.Start()

	srv := gateway.NewServer(cfg, store)

	entrypoints := entrypoint.NewRunner([]entrypoint.Entrypoint{
		{
			Name: "http",
			Run:  srv.Run,
		},
	})

	if err = entrypoints.Run(); err != nil {
		slog.ErrorContext(ctx, "[main] failed to run entrypoints", slog.Any("err", err))
	}
}
