package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/config"
	"github.com/RisheDubs/inkwell/apps/api/internal/httpapi"
	mongostore "github.com/RisheDubs/inkwell/apps/api/internal/store/mongo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	store, err := mongostore.Connect(connectCtx, cfg.MongoURI)
	cancel()
	if err != nil {
		slog.Error("connect to mongodb", "err", err)
		os.Exit(1)
	}
	defer func() { _ = store.Close(context.Background()) }()
	slog.Info("connected to mongodb")

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("api starting", "addr", srv.Addr, "log_level", cfg.LogLevel.String())
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
