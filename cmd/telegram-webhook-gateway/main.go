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
	fetcher, err := cidr.NewHTTPFetcher(cfg.Telegram)
	if err != nil {
		slog.ErrorContext(ctx, "[main] failed to init cidr fetcher", slog.Any("err", err))
		os.Exit(1)
	}

	updater := cidr.NewPeriodicUpdater(store, fetcher, cfg.Telegram.CIDRUpdateInterval.Value)
	if err = updater.Update(ctx); err != nil {
		slog.ErrorContext(ctx, "[main] failed to init update cidr", slog.Any("err", err))
		os.Exit(1)
	}

	srv := gateway.NewServer(cfg, store)

	entrypoints := entrypoint.NewRunner([]entrypoint.Entrypoint{
		{
			Name: "http",
			Run:  srv.Run,
			Stop: srv.Stop,
		},
		{
			Name: "cidr-updater",
			Run: func(context.Context) error {
				updater.Run()
				return nil
			},
			Stop: func(context.Context) error {
				updater.Stop()
				return nil
			},
		},
	})

	if err = entrypoints.Run(); err != nil {
		slog.ErrorContext(ctx, "[main] failed to run entrypoints", slog.Any("err", err))
	}
}
